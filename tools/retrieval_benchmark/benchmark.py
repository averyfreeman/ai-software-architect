"""Measure a portable JSON scan against a SQLite FTS5 retrieval index.

The benchmark deliberately keeps Markdown/YAML/JSON-like documents canonical.
SQLite is a disposable derived index used only for retrieval evidence and is
never required by the Git BBQ runtime.
"""

from __future__ import annotations

import argparse
import hashlib
import json
import math
import re
import sqlite3
import sys
import tempfile
import time
from collections.abc import Iterable, Mapping, Sequence
from dataclasses import dataclass
from pathlib import Path
from typing import Any, Protocol

SCHEMA_VERSION = "1.0"
DEFAULT_TOP_K = (1, 3, 5)
TOKEN_PATTERN = re.compile(r"[^\W_]+", re.UNICODE)


@dataclass(frozen=True)
class Document:
    """One canonical document supplied to a derived retriever."""

    document_id: str
    title: str
    body: str

    @property
    def text(self) -> str:
        return f"{self.title}\n{self.body}"

    @property
    def tokens(self) -> tuple[str, ...]:
        return _tokenize(self.text)

    @property
    def token_count(self) -> int:
        return len(self.tokens)


@dataclass(frozen=True)
class QueryCase:
    """A query with binary relevance and answer-support annotations."""

    query_id: str
    query: str
    relevant_documents: frozenset[str]
    support_terms: tuple[str, ...]


@dataclass(frozen=True)
class StaleCase:
    """A source replacement used to prove stale entries disappear on rebuild."""

    document_id: str
    query: str
    replacement: Document
    replacement_query: str


@dataclass(frozen=True)
class BenchmarkFixture:
    """Validated corpus, query judgments, and stale-source scenario."""

    documents: tuple[Document, ...]
    queries: tuple[QueryCase, ...]
    stale_case: StaleCase
    source_path: Path
    source_sha256: str


@dataclass(frozen=True)
class SearchHit:
    document_id: str
    score: float
    rank: int


class Retriever(Protocol):
    def search(self, query: str, limit: int) -> list[SearchHit]:
        """Return stable ranked hits for one query."""


class JsonScanRetriever:
    """Small, dependency-free lexical baseline over canonical documents."""

    def __init__(self, documents: Sequence[Document]) -> None:
        self._documents = tuple(documents)

    def search(self, query: str, limit: int) -> list[SearchHit]:
        if limit <= 0:
            return []
        query_tokens = set(_tokenize(query))
        scored: list[tuple[int, str]] = []
        for document in self._documents:
            overlap = len(query_tokens & set(document.tokens))
            if overlap:
                scored.append((overlap, document.document_id))
        scored.sort(key=lambda item: (-item[0], item[1]))
        return [
            SearchHit(document_id=document_id, score=float(score), rank=rank)
            for rank, (score, document_id) in enumerate(scored[:limit], start=1)
        ]


class SQLiteFTS5Retriever:
    """Disposable SQLite FTS5 index with explicit rebuild and recovery seams."""

    def __init__(self, database_path: Path) -> None:
        self.path = database_path
        self.path.parent.mkdir(parents=True, exist_ok=True)
        self._connection: sqlite3.Connection | None = None
        self._open()

    @staticmethod
    def available() -> bool:
        return sqlite_fts5_available()

    def _open(self) -> None:
        connection = sqlite3.connect(self.path)
        self._connection = connection
        try:
            connection.execute(
                "CREATE VIRTUAL TABLE IF NOT EXISTS documents "
                "USING fts5(document_id UNINDEXED, title, body)"
            )
            connection.commit()
        except BaseException:
            connection.close()
            self._connection = None
            raise

    def _require_connection(self) -> sqlite3.Connection:
        if self._connection is None:
            raise RuntimeError("retrieval index is closed")
        return self._connection

    def close(self) -> None:
        if self._connection is not None:
            self._connection.close()
            self._connection = None

    def __enter__(self) -> SQLiteFTS5Retriever:
        return self

    def __exit__(self, *_: object) -> None:
        self.close()

    def rebuild(self, documents: Sequence[Document]) -> int:
        """Replace the derived index from canonical documents and return its count."""

        self.close()
        if self.path.exists():
            self.path.unlink()
        self._open()
        connection = self._require_connection()
        connection.executemany(
            "INSERT INTO documents(document_id, title, body) VALUES (?, ?, ?)",
            ((doc.document_id, doc.title, doc.body) for doc in documents),
        )
        connection.commit()
        return len(documents)

    def count(self) -> int:
        connection = self._require_connection()
        row = connection.execute("SELECT count(*) FROM documents").fetchone()
        if row is None:
            return 0
        return int(row[0])

    def indexed_ids(self) -> tuple[str, ...]:
        connection = self._require_connection()
        rows = connection.execute(
            "SELECT document_id FROM documents ORDER BY document_id"
        ).fetchall()
        return tuple(str(row[0]) for row in rows)

    def indexed_content_digest(self) -> str:
        """Return a canonical digest of the indexed document content."""

        connection = self._require_connection()
        rows = connection.execute(
            "SELECT document_id, title, body FROM documents ORDER BY document_id"
        ).fetchall()
        documents = tuple(
            Document(document_id=str(row[0]), title=str(row[1]), body=str(row[2]))
            for row in rows
        )
        return _document_digest(documents)

    def search(self, query: str, limit: int) -> list[SearchHit]:
        if limit <= 0:
            return []
        match_query = _fts_query(query)
        if not match_query:
            return []
        connection = self._require_connection()
        rows = connection.execute(
            "SELECT document_id, bm25(documents) AS score "
            "FROM documents WHERE documents MATCH ? "
            "ORDER BY score ASC, document_id ASC LIMIT ?",
            (match_query, limit),
        ).fetchall()
        return [
            SearchHit(document_id=str(row[0]), score=float(row[1]), rank=rank)
            for rank, row in enumerate(rows, start=1)
        ]


def sqlite_fts5_available() -> bool:
    """Return whether this Python build exposes SQLite's FTS5 module."""

    try:
        with sqlite3.connect(":memory:") as connection:
            connection.execute("CREATE VIRTUAL TABLE fts5_probe USING fts5(value)")
        return True
    except sqlite3.Error:
        return False


def load_fixture(path: Path) -> BenchmarkFixture:
    """Load and validate a versioned benchmark fixture."""

    raw = json.loads(path.read_text(encoding="utf-8"))
    root = _mapping(raw, "fixture")
    if root.get("schema_version") != SCHEMA_VERSION:
        raise ValueError("unsupported retrieval fixture schema_version")

    documents = tuple(_load_documents(root.get("documents")))
    document_ids = {document.document_id for document in documents}
    queries = tuple(_load_queries(root.get("queries"), document_ids))
    stale_root = _mapping(root.get("stale_case"), "stale_case")
    stale_id = _nonempty_string(stale_root, "document_id", "stale_case")
    if stale_id not in document_ids:
        raise ValueError(f"stale_case document_id is unknown: {stale_id}")
    replacement_root = _mapping(stale_root.get("replacement"), "replacement")
    replacement = Document(
        document_id=stale_id,
        title=_nonempty_string(replacement_root, "title", "replacement"),
        body=_nonempty_string(replacement_root, "body", "replacement"),
    )
    stale_case = StaleCase(
        document_id=stale_id,
        query=_nonempty_string(stale_root, "query", "stale_case"),
        replacement=replacement,
        replacement_query=_nonempty_string(
            stale_root, "replacement_query", "stale_case"
        ),
    )
    return BenchmarkFixture(
        documents=documents,
        queries=queries,
        stale_case=stale_case,
        source_path=path,
        source_sha256=hashlib.sha256(path.read_bytes()).hexdigest(),
    )


def benchmark_fixture(
    fixture: BenchmarkFixture,
    *,
    iterations: int = 25,
    top_k: Sequence[int] = DEFAULT_TOP_K,
) -> dict[str, Any]:
    """Run the full M-16 evidence suite and return a JSON-ready report."""

    if iterations < 1:
        raise ValueError("iterations must be positive")
    normalized_top_k = tuple(sorted({int(value) for value in top_k}))
    if not normalized_top_k or normalized_top_k[0] < 1:
        raise ValueError("top_k must contain positive values")
    if not sqlite_fts5_available():
        raise RuntimeError("the supported Python SQLite build does not provide FTS5")

    document_lookup = {document.document_id: document for document in fixture.documents}
    max_k = normalized_top_k[-1]
    json_retriever = JsonScanRetriever(fixture.documents)

    with tempfile.TemporaryDirectory(prefix="m16-retrieval-") as temporary:
        database_path = Path(temporary) / "retrieval.sqlite3"
        fts_retriever = SQLiteFTS5Retriever(database_path)
        try:
            initial_count = fts_retriever.rebuild(fixture.documents)
            backends = {
                "json_scan": _evaluate_backend(
                    json_retriever,
                    fixture.queries,
                    document_lookup,
                    normalized_top_k,
                    iterations,
                ),
                "sqlite_fts5": _evaluate_backend(
                    fts_retriever,
                    fixture.queries,
                    document_lookup,
                    normalized_top_k,
                    iterations,
                ),
            }
            rebuild = _rebuild_evidence(fts_retriever, fixture.documents)
            stale = _stale_evidence(fts_retriever, fixture)
            fts_retriever.rebuild(fixture.documents)
            recovery = _recovery_evidence(
                fts_retriever, fixture.documents, fixture.queries[0].query, max_k
            )
        finally:
            fts_retriever.close()

    return {
        "schema_version": SCHEMA_VERSION,
        "benchmark": "m16-retrieval",
        "fixture": {
            "path": _display_fixture_path(fixture.source_path),
            "sha256": fixture.source_sha256,
            "document_count": len(fixture.documents),
            "query_count": len(fixture.queries),
            "initial_index_count": initial_count,
        },
        "parameters": {
            "top_k": list(normalized_top_k),
            "latency_iterations": iterations,
            "latency_clock": "time.perf_counter_ns",
        },
        "backends": backends,
        "rebuild": rebuild,
        "stale_documents": stale,
        "failure_recovery": recovery,
        "decision": {
            "sqlite_fts5_available": True,
            "sqlite_fts5_selected": True,
            "sqlite_vec_considered": False,
            "sqlite_vec_reason": (
                "The benchmark establishes a portable FTS5 baseline first; "
                "no vector dependency is justified by this small deterministic corpus."
            ),
        },
        "limitations": [
            "This fixture is representative deterministic evidence, not production-scale volume.",
            "Latency values are descriptive and must be compared on like-for-like hosts.",
        ],
    }


def _evaluate_backend(
    retriever: Retriever,
    queries: Sequence[QueryCase],
    document_lookup: Mapping[str, Document],
    top_k: Sequence[int],
    iterations: int,
) -> dict[str, Any]:
    max_k = top_k[-1]
    ranked: dict[str, list[SearchHit]] = {}
    for query in queries:
        ranked[query.query_id] = retriever.search(query.query, max_k)

    quality: dict[str, dict[str, list[float]]] = {
        "precision": {},
        "recall": {},
        "mrr": {},
        "ndcg": {},
        "answer_support_coverage": {},
    }
    query_results: list[dict[str, Any]] = []
    for query in queries:
        hits = ranked[query.query_id]
        query_results.append(
            {
                "query_id": query.query_id,
                "retrieved": [hit.document_id for hit in hits],
                "relevant": sorted(query.relevant_documents),
            }
        )
        for cutoff in top_k:
            prefix = hits[:cutoff]
            quality["precision"].setdefault(str(cutoff), []).append(
                _precision(prefix, query.relevant_documents, cutoff)
            )
            quality["recall"].setdefault(str(cutoff), []).append(
                _recall(prefix, query.relevant_documents)
            )
            quality["mrr"].setdefault(str(cutoff), []).append(
                _mrr(prefix, query.relevant_documents)
            )
            quality["ndcg"].setdefault(str(cutoff), []).append(
                _ndcg(prefix, query.relevant_documents)
            )
            quality["answer_support_coverage"].setdefault(str(cutoff), []).append(
                _support_coverage(prefix, query.support_terms, document_lookup)
            )

    aggregate_quality = {
        metric: {
            cutoff: _rounded_mean(values)
            for cutoff, values in cutoff_values.items()
        }
        for metric, cutoff_values in quality.items()
    }
    latency_samples: list[float] = []
    for _ in range(iterations):
        for query in queries:
            started = time.perf_counter_ns()
            retriever.search(query.query, max_k)
            latency_samples.append((time.perf_counter_ns() - started) / 1_000_000)
    full_context_tokens = sum(document.token_count for document in document_lookup.values())
    retrieved_context_tokens = [
        sum(document_lookup[hit.document_id].token_count for hit in ranked[query.query_id])
        for query in queries
    ]
    reduction_ratios = [
        (full_context_tokens - selected) / full_context_tokens
        for selected in retrieved_context_tokens
    ]
    return {
        "quality": aggregate_quality,
        "query_results": query_results,
        "latency_ms": _latency_summary(latency_samples),
        "context": {
            "full_corpus_tokens": full_context_tokens,
            "mean_top_k_tokens": _rounded_mean(retrieved_context_tokens),
            "mean_reduction_ratio": _rounded_mean(reduction_ratios),
        },
    }


def _rebuild_evidence(
    retriever: SQLiteFTS5Retriever, documents: Sequence[Document]
) -> dict[str, Any]:
    expected_ids = tuple(sorted(document.document_id for document in documents))
    expected_digest = _document_digest(documents)
    timings: list[float] = []
    counts: list[int] = []
    indexed_digests: list[str] = []
    for source in (documents, tuple(reversed(documents))):
        started = time.perf_counter_ns()
        counts.append(retriever.rebuild(source))
        timings.append((time.perf_counter_ns() - started) / 1_000_000)
        indexed_digests.append(retriever.indexed_content_digest())
    indexed_ids = retriever.indexed_ids()
    return {
        "runs": len(timings),
        "counts": counts,
        "latency_ms": {
            "first": round(timings[0], 6),
            "repeat": round(timings[1], 6),
        },
        "expected_document_digest": expected_digest,
        "indexed_content_digests": indexed_digests,
        "indexed_document_ids": list(indexed_ids),
        "matches_canonical": (
            indexed_digests[-1] == expected_digest
            and indexed_ids == expected_ids
            and retriever.count() == len(documents)
        ),
        "repeat_matches_canonical": (
            counts == [len(documents), len(documents)]
            and indexed_digests == [expected_digest, expected_digest]
        ),
    }


def _stale_evidence(
    retriever: SQLiteFTS5Retriever, fixture: BenchmarkFixture
) -> dict[str, Any]:
    stale_case = fixture.stale_case
    before = [hit.document_id for hit in retriever.search(stale_case.query, 5)]
    updated_documents = tuple(
        stale_case.replacement if doc.document_id == stale_case.document_id else doc
        for doc in fixture.documents
    )
    retriever.rebuild(updated_documents)
    after_old_query = [hit.document_id for hit in retriever.search(stale_case.query, 5)]
    after_replacement_query = [
        hit.document_id for hit in retriever.search(stale_case.replacement_query, 5)
    ]
    return {
        "document_id": stale_case.document_id,
        "before_rebuild_old_query_hits": before,
        "after_rebuild_old_query_hits": after_old_query,
        "after_rebuild_replacement_query_hits": after_replacement_query,
        "old_content_removed": stale_case.document_id in before
        and stale_case.document_id not in after_old_query,
        "replacement_content_indexed": stale_case.document_id in after_replacement_query,
    }


def _recovery_evidence(
    retriever: SQLiteFTS5Retriever,
    documents: Sequence[Document],
    query: str,
    limit: int,
) -> dict[str, Any]:
    retriever.rebuild(documents)
    path = retriever.path
    retriever.close()
    path.write_bytes(b"not a sqlite database")
    corruption_detected = False
    try:
        broken = SQLiteFTS5Retriever(path)
    except sqlite3.Error:
        corruption_detected = True
    else:
        broken.close()
    if path.exists():
        path.unlink()
    recovered = SQLiteFTS5Retriever(path)
    recovered_count = recovered.rebuild(documents)
    recovered_hits = recovered.search(query, limit)
    recovered.close()
    return {
        "corruption_detected": corruption_detected,
        "recovered": corruption_detected and recovered_count == len(documents),
        "recovered_document_count": recovered_count,
        "post_recovery_hits": [hit.document_id for hit in recovered_hits],
    }


def _precision(hits: Sequence[SearchHit], relevant: frozenset[str], cutoff: int) -> float:
    return sum(hit.document_id in relevant for hit in hits[:cutoff]) / cutoff


def _recall(hits: Sequence[SearchHit], relevant: frozenset[str]) -> float:
    return sum(hit.document_id in relevant for hit in hits) / len(relevant)


def _mrr(hits: Sequence[SearchHit], relevant: frozenset[str]) -> float:
    for hit in hits:
        if hit.document_id in relevant:
            return 1 / hit.rank
    return 0.0


def _ndcg(hits: Sequence[SearchHit], relevant: frozenset[str]) -> float:
    dcg = sum(
        (1 / math.log2(rank + 1))
        for rank, hit in enumerate(hits, start=1)
        if hit.document_id in relevant
    )
    ideal_length = min(len(hits), len(relevant))
    ideal = sum(1 / math.log2(rank + 1) for rank in range(1, ideal_length + 1))
    return dcg / ideal if ideal else 0.0


def _support_coverage(
    hits: Sequence[SearchHit],
    support_terms: Sequence[str],
    document_lookup: Mapping[str, Document],
) -> float:
    if not support_terms:
        return 1.0
    tokens = {
        token
        for hit in hits
        for token in document_lookup[hit.document_id].tokens
    }
    covered = sum(set(_tokenize(term)) <= tokens for term in support_terms)
    return covered / len(support_terms)


def _latency_summary(samples: Sequence[float]) -> dict[str, Any]:
    if not samples:
        raise ValueError("latency requires at least one sample")
    return {
        "samples": len(samples),
        "p50_ms": round(_percentile(samples, 0.50), 6),
        "p95_ms": round(_percentile(samples, 0.95), 6),
        "min_ms": round(min(samples), 6),
        "max_ms": round(max(samples), 6),
    }


def _percentile(values: Sequence[float], probability: float) -> float:
    ordered = sorted(values)
    if len(ordered) == 1:
        return ordered[0]
    position = (len(ordered) - 1) * probability
    lower = math.floor(position)
    upper = math.ceil(position)
    if lower == upper:
        return ordered[lower]
    fraction = position - lower
    return ordered[lower] + (ordered[upper] - ordered[lower]) * fraction


def _rounded_mean(values: Iterable[float | int]) -> float:
    values = list(values)
    if not values:
        return 0.0
    return round(float(sum(values)) / len(values), 6)


def _document_digest(documents: Sequence[Document]) -> str:
    canonical = [
        {"id": doc.document_id, "title": doc.title, "body": doc.body}
        for doc in sorted(documents, key=lambda item: item.document_id)
    ]
    encoded = json.dumps(canonical, ensure_ascii=False, separators=(",", ":"))
    return hashlib.sha256(encoded.encode("utf-8")).hexdigest()


def _display_fixture_path(path: Path) -> str:
    repository_root = Path(__file__).resolve().parents[2]
    try:
        return path.resolve().relative_to(repository_root).as_posix()
    except ValueError:
        return path.name


def _tokenize(value: str) -> tuple[str, ...]:
    return tuple(match.group(0).casefold() for match in TOKEN_PATTERN.finditer(value))


def _fts_query(value: str) -> str:
    tokens: list[str] = []
    seen: set[str] = set()
    for token in _tokenize(value):
        if token not in seen:
            tokens.append(token)
            seen.add(token)
    return " OR ".join(f'"{token}"' for token in tokens)


def _mapping(value: object, label: str) -> dict[str, Any]:
    if not isinstance(value, dict):
        raise ValueError(f"{label} must be an object")
    return value


def _nonempty_string(mapping: Mapping[str, Any], key: str, label: str) -> str:
    value = mapping.get(key)
    if not isinstance(value, str) or not value.strip():
        raise ValueError(f"{label}.{key} must be a non-empty string")
    return value.strip()


def _load_documents(value: object) -> list[Document]:
    if not isinstance(value, list) or not value:
        raise ValueError("fixture.documents must be a non-empty array")
    documents: list[Document] = []
    seen: set[str] = set()
    for index, raw in enumerate(value):
        mapping = _mapping(raw, f"documents[{index}]")
        document = Document(
            document_id=_nonempty_string(mapping, "id", f"documents[{index}]"),
            title=_nonempty_string(mapping, "title", f"documents[{index}]"),
            body=_nonempty_string(mapping, "body", f"documents[{index}]"),
        )
        if document.document_id in seen:
            raise ValueError(f"duplicate document id: {document.document_id}")
        seen.add(document.document_id)
        documents.append(document)
    return documents


def _load_queries(value: object, document_ids: set[str]) -> list[QueryCase]:
    if not isinstance(value, list) or not value:
        raise ValueError("fixture.queries must be a non-empty array")
    queries: list[QueryCase] = []
    seen: set[str] = set()
    for index, raw in enumerate(value):
        mapping = _mapping(raw, f"queries[{index}]")
        label = f"queries[{index}]"
        query_id = _nonempty_string(mapping, "id", label)
        if query_id in seen:
            raise ValueError(f"duplicate query id: {query_id}")
        seen.add(query_id)
        raw_relevant = mapping.get("relevant")
        raw_support = mapping.get("support_terms")
        if not isinstance(raw_relevant, list) or not raw_relevant:
            raise ValueError(f"{label}.relevant must be a non-empty array")
        if not isinstance(raw_support, list):
            raise ValueError(f"{label}.support_terms must be an array")
        relevant = frozenset(
            _nonempty_string({"value": item}, "value", f"{label}.relevant")
            for item in raw_relevant
        )
        unknown = sorted(relevant - document_ids)
        if unknown:
            raise ValueError(f"{label}.relevant contains unknown ids: {unknown}")
        support_terms = tuple(
            _nonempty_string({"value": item}, "value", f"{label}.support_terms")
            for item in raw_support
        )
        queries.append(
            QueryCase(
                query_id=query_id,
                query=_nonempty_string(mapping, "query", label),
                relevant_documents=relevant,
                support_terms=support_terms,
            )
        )
    return queries


def _default_fixture_path() -> Path:
    return Path(__file__).resolve().parents[2] / "tests/fixtures/retrieval/m16.json"


def _build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--fixture", type=Path, default=_default_fixture_path())
    parser.add_argument(
        "--output",
        type=Path,
        default=Path(".tmp/retrieval/m16-report.json"),
        help="generated JSON report path",
    )
    parser.add_argument("--iterations", type=int, default=25)
    return parser


def main(argv: Sequence[str] | None = None) -> int:
    args = _build_parser().parse_args(argv)
    try:
        fixture = load_fixture(args.fixture)
        report = benchmark_fixture(fixture, iterations=args.iterations)
        args.output.parent.mkdir(parents=True, exist_ok=True)
        args.output.write_text(
            json.dumps(report, ensure_ascii=False, indent=2, sort_keys=True) + "\n",
            encoding="utf-8",
        )
    except (OSError, ValueError, RuntimeError, sqlite3.Error) as error:
        print(f"retrieval benchmark failed: {error}", file=sys.stderr)
        return 2
    print(args.output)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
