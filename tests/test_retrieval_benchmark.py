"""Deterministic M-16 retrieval and SQLite FTS5 gate tests."""

from __future__ import annotations

import json
from pathlib import Path

import pytest

from tools.retrieval_benchmark.benchmark import (
    DEFAULT_TOP_K,
    SQLiteFTS5Retriever,
    benchmark_fixture,
    load_fixture,
    main,
    sqlite_fts5_available,
)

FIXTURE = Path(__file__).parent / "fixtures" / "retrieval" / "m16.json"


@pytest.fixture
def retrieval_fixture():
    if not sqlite_fts5_available():
        pytest.skip("SQLite FTS5 is unavailable on this host")
    return load_fixture(FIXTURE)


def test_m16_fixture_is_deterministic(retrieval_fixture):
    assert retrieval_fixture.source_sha256
    assert len(retrieval_fixture.documents) == 13
    assert len(retrieval_fixture.queries) == 8
    assert retrieval_fixture.stale_case.document_id == "adr-stale-source"


def test_m16_benchmark_measures_quality_latency_context_and_lifecycle(
    retrieval_fixture,
):
    report = benchmark_fixture(retrieval_fixture, iterations=3)

    assert report["benchmark"] == "m16-retrieval"
    assert report["fixture"]["path"] == "tests/fixtures/retrieval/m16.json"
    assert report["parameters"]["latency_iterations"] == 3
    assert report["parameters"]["top_k"] == list(DEFAULT_TOP_K)
    assert report["decision"]["sqlite_fts5_selected"] is True
    assert report["decision"]["sqlite_vec_considered"] is False

    fts = report["backends"]["sqlite_fts5"]
    assert fts["quality"]["recall"]["3"] >= 0.5
    assert fts["quality"]["mrr"]["5"] >= 0.5
    assert fts["quality"]["ndcg"]["5"] >= 0.5
    assert fts["quality"]["answer_support_coverage"]["5"] >= 0.5
    assert fts["latency_ms"]["samples"] == 3 * len(retrieval_fixture.queries)
    assert fts["latency_ms"]["p50_ms"] >= 0
    assert fts["latency_ms"]["p95_ms"] >= fts["latency_ms"]["p50_ms"]
    assert 0 < fts["context"]["mean_reduction_ratio"] < 1

    assert report["rebuild"]["matches_canonical"] is True
    assert report["rebuild"]["repeat_matches_canonical"] is True
    assert report["rebuild"]["indexed_content_digests"] == [
        report["rebuild"]["expected_document_digest"],
        report["rebuild"]["expected_document_digest"],
    ]
    assert report["stale_documents"]["old_content_removed"] is True
    assert report["stale_documents"]["replacement_content_indexed"] is True
    assert report["failure_recovery"]["corruption_detected"] is True
    assert report["failure_recovery"]["recovered"] is True


def test_sqlite_rebuild_replaces_existing_documents(tmp_path, retrieval_fixture):
    database = tmp_path / "retrieval.sqlite3"
    with SQLiteFTS5Retriever(database) as retriever:
        assert retriever.rebuild(retrieval_fixture.documents) == 13
        assert retriever.count() == 13
        assert retriever.rebuild(retrieval_fixture.documents[:2]) == 2
        assert retriever.count() == 2
        assert retriever.indexed_ids() == (
            "adr-canonical-artifacts",
            "adr-hook-lifecycle",
        )


def test_m16_cli_writes_generated_report(tmp_path, retrieval_fixture):
    output = tmp_path / "m16-report.json"
    assert main(
        [
            "--fixture",
            str(FIXTURE),
            "--output",
            str(output),
            "--iterations",
            "2",
        ]
    ) == 0

    report = json.loads(output.read_text(encoding="utf-8"))
    assert report["fixture"]["document_count"] == 13
    assert report["fixture"]["query_count"] == 8
    assert report["failure_recovery"]["recovered"] is True
