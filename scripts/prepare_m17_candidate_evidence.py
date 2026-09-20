"""Generate sanitized repository-owned evidence for a Go runtime candidate."""

from __future__ import annotations

import argparse
import hashlib
import json
import subprocess
import sys
from pathlib import Path
from typing import Any

ROOT = Path(__file__).resolve().parents[1]
if str(ROOT) not in sys.path:
    sys.path.insert(0, str(ROOT))

from adapters.codex import validate_plugin  # noqa: E402
from adapters.codex.runtime_targets import (  # noqa: E402
    RUNTIME_TARGETS,
    target_for_name,
)


def _sha256(path: Path) -> str:
    digest = hashlib.sha256()
    with path.open("rb") as stream:
        for chunk in iter(lambda: stream.read(1024 * 1024), b""):
            digest.update(chunk)
    return digest.hexdigest()


def _source_commit() -> str | None:
    try:
        result = subprocess.run(
            ["git", "rev-parse", "HEAD"],  # noqa: S607
            cwd=ROOT,
            check=True,
            capture_output=True,
            text=True,
        )
    except (OSError, subprocess.CalledProcessError):
        return None
    value = result.stdout.strip()
    return value or None


def prepare_candidate_evidence(
    plugin_path: Path,
    *,
    target_name: str,
    runtime_implementation: str = "go",
    archive_path: Path | None = None,
) -> dict[str, Any]:
    target = target_for_name(target_name)
    plugin_path = plugin_path.resolve(strict=True)
    validate_plugin.validate(
        plugin_path,
        target=target,
        runtime_implementation=runtime_implementation,
    )
    manifest = json.loads(
        (plugin_path / ".codex-plugin" / "plugin.json").read_text("utf-8")
    )
    provenance_path = plugin_path / "provenance.json"
    provenance_bytes = provenance_path.read_bytes()
    provenance = json.loads(provenance_bytes)
    runtime = provenance["runtime"]
    executable = plugin_path / runtime["executable"]
    evidence: dict[str, Any] = {
        "schema_version": "1.0.0",
        "status": "repository-prepared-not-release-ready",
        "source_commit": _source_commit(),
        "candidate": {
            "plugin_name": manifest["name"],
            "plugin_version": manifest["version"],
            "target": target.name,
            "runtime_implementation": runtime["implementation"],
            "runtime_source": runtime["source"],
            "runtime_sha256": _sha256(executable),
            "provenance_sha256": hashlib.sha256(provenance_bytes).hexdigest(),
        },
        "package": {
            "archive_name": archive_path.name if archive_path is not None else None,
            "archive_sha256": _sha256(archive_path) if archive_path is not None else None,
        },
        "repository_gates": {
            "go_runtime_build": "prepared",
            "package_validation": "passed",
            "runtime_smoke": "external-candidate-run-required",
            "m15_exploratory_evidence": "blocked",
            "codex_desktop_lifecycle": "manual-gate-required",
            "clean_machine_behavior": "manual-gate-required",
            "release_evidence": "candidate-specific-record-required",
        },
        "cutover": {
            "authorized": False,
            "python_compatibility_paths_retained": True,
            "reason": (
                "External M-15, Desktop lifecycle, clean-machine, behavioral, and "
                "release-evidence gates remain outside repository automation."
            ),
        },
    }
    return evidence


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--plugin", type=Path, default=Path("dist/codex/ai-software-architect"))
    parser.add_argument("--output", type=Path, default=Path(".tmp/m17/go-candidate-evidence.json"))
    parser.add_argument("--archive", type=Path)
    parser.add_argument("--target", choices=sorted(RUNTIME_TARGETS), required=True)
    parser.add_argument(
        "--runtime-implementation",
        choices=("python", "go"),
        default="go",
    )
    args = parser.parse_args()
    evidence = prepare_candidate_evidence(
        args.plugin,
        target_name=args.target,
        runtime_implementation=args.runtime_implementation,
        archive_path=args.archive,
    )
    output = args.output if args.output.is_absolute() else ROOT / args.output
    output.parent.mkdir(parents=True, exist_ok=True)
    output.write_text(json.dumps(evidence, indent=2, sort_keys=True) + "\n", encoding="utf-8")
    print(output)


if __name__ == "__main__":
    main()
