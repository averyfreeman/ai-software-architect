from pathlib import Path

import pytest

from adapters.codex import build_plugin
from adapters.codex.runtime_targets import target_for_name
from scripts.package_codex_release import package_release


def test_package_release_rejects_symlinked_plugin_member(
    tmp_path: Path, monkeypatch: pytest.MonkeyPatch
) -> None:
    output_parent = tmp_path / "dist" / "codex"
    output = output_parent / "ai-software-architect"
    monkeypatch.setattr(build_plugin, "OUTPUT_PARENT", output_parent)
    monkeypatch.setattr(build_plugin, "OUTPUT", output)
    runtime = tmp_path / "runtime"
    runtime.mkdir()
    executable = runtime / "ai-architect-runtime"
    executable.write_bytes(b"reviewed-darwin-test-runtime")
    executable.chmod(0o755)
    target = target_for_name("aarch64-darwin")
    assembled = build_plugin.assemble(
        runtime,
        plugin_version="0.2.3+symlink.test",
        target=target,
    )

    outside = tmp_path / "outside-secret.txt"
    outside.write_text("must not enter the archive\n", encoding="utf-8")
    link = assembled / "leaked.txt"
    try:
        link.symlink_to(outside)
    except OSError as exc:
        pytest.skip(f"symlinks unavailable: {exc}")

    with pytest.raises(ValueError, match="symlink"):
        package_release(
            assembled,
            tmp_path / "release",
            target=target,
            plugin_version="0.2.3+symlink.test",
        )
