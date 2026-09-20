from __future__ import annotations

import json
from pathlib import Path

from adapters.codex import build_plugin
from adapters.codex.runtime_targets import target_for_name
from scripts.prepare_m17_candidate_evidence import prepare_candidate_evidence


def test_go_candidate_evidence_is_sanitized_and_keeps_cutover_blocked(
    tmp_path: Path,
    monkeypatch,
) -> None:
    output_parent = tmp_path / "dist" / "codex"
    output = output_parent / "ai-software-architect"
    monkeypatch.setattr(build_plugin, "OUTPUT_PARENT", output_parent)
    monkeypatch.setattr(build_plugin, "OUTPUT", output)
    runtime = tmp_path / "runtime"
    runtime.mkdir()
    executable = runtime / "ai-architect-runtime.exe"
    executable.write_bytes(b"go-runtime-candidate")

    plugin = build_plugin.assemble(
        runtime,
        plugin_version="0.2.3+m17.test",
        target=target_for_name("windows-x86_64"),
        runtime_implementation="go",
    )
    evidence = prepare_candidate_evidence(
        plugin,
        target_name="windows-x86_64",
    )

    assert evidence["status"] == "repository-prepared-not-release-ready"
    assert evidence["candidate"]["runtime_implementation"] == "go"
    assert evidence["candidate"]["provenance_sha256"]
    assert evidence["cutover"]["authorized"] is False
    assert evidence["cutover"]["python_compatibility_paths_retained"] is True
    serialized = json.dumps(evidence)
    assert str(tmp_path) not in serialized
