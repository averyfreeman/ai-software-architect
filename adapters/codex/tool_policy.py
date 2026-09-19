# SPDX-FileCopyrightText: 2026 Leonardo Muffato (AUTOSOFT Engineering - www.autosoft-engineering.de)
# SPDX-License-Identifier: MIT

"""Pure fail-closed policy for Codex tool calls during architect turns."""

from __future__ import annotations

import re
from pathlib import Path

try:
    from adapters.codex.activation_policy import (
        CodexTurnContext,
        repository_snapshot_command,
    )
    from adapters.codex.artifact_paths import is_canonical_artifact_path
    from adapters.codex.artifact_validation import patch_text_from_tool_input
except ModuleNotFoundError as exc:
    if exc.name != "adapters":
        raise
    from activation_policy import (  # type: ignore[import-not-found, no-redef]
        CodexTurnContext,
        repository_snapshot_command,
    )
    from artifact_paths import (  # type: ignore[import-not-found, no-redef]
        is_canonical_artifact_path,
    )
    from artifact_validation import (  # type: ignore[import-not-found, no-redef]
        patch_text_from_tool_input,
    )

SHELL_TOOL_NAMES = {"bash", "exec_command", "shell_command"}
PATCH_TOOL_NAMES = {"apply_patch", "edit", "write"}
WEB_LOOKUP_TOOL_NAMES = {"websearch", "web_search", "search_query", "web__run"}
STATIC_POWERSHELL_COMMANDS = {
    "get-childitem",
    "get-content",
    "select-string",
    "test-path",
}
STATIC_GIT_SUBCOMMANDS = {"diff", "log", "ls-files", "show", "status"}
SHELL_COMPOSITION_PATTERN = re.compile(r"[\r\n;&|{}<>`] | \$", flags=re.VERBOSE)
PATCH_FILE_PATTERN = re.compile(
    r"^\*\*\* (?:Add|Update|Delete) File: (.+)$|^\*\*\* Move to: (.+)$",
    flags=re.MULTILINE,
)


def _normalized_local_tool_name(value: object) -> str | None:
    if not isinstance(value, str):
        return None
    return value.rsplit(".", 1)[-1].casefold()


def _command_from_tool_input(value: object) -> str | None:
    if isinstance(value, str):
        return value
    if not isinstance(value, dict):
        return None
    command = value.get("command")
    return command if isinstance(command, str) else None


def _shell_denial_reason(
    tool_input: object,
    *,
    plugin_root: Path | None = None,
) -> str | None:
    command = _command_from_tool_input(tool_input)
    if command is None:
        return (
            "The AI Software Architect could not verify this shell command's "
            "arguments, so it was denied. Use host-native static reads instead."
        )
    stripped = command.strip()
    if (
        plugin_root is not None
        and stripped == repository_snapshot_command(plugin_root)
    ):
        return None
    if not stripped or SHELL_COMPOSITION_PATTERN.search(stripped):
        return (
            "AI Software Architect shell inspection is fail-closed: use exactly one "
            "allowlisted static read command without pipelines, variables, call "
            "operators, script blocks, redirection, or command composition."
        )
    words = stripped.split()
    executable = words[0].casefold().removesuffix(".exe")
    if executable in STATIC_POWERSHELL_COMMANDS:
        return None
    if executable == "git" and len(words) >= 2:
        subcommand = words[1].casefold()
        unsafe_options = {"--ext-diff", "--textconv"}
        if subcommand in STATIC_GIT_SUBCOMMANDS and not unsafe_options.intersection(
            word.casefold().split("=", 1)[0] for word in words[2:]
        ):
            return None
    if executable in {"rg", "ripgrep"}:
        unsafe_options = {"--pre", "--pre-glob"}
        if not unsafe_options.intersection(
            word.casefold().split("=", 1)[0] for word in words[1:]
        ):
            return None
    return (
        "AI Software Architect analysis treats repository code as untrusted data. "
        "Only allowlisted static file and Git reads are permitted; interpreters, "
        "test runners, package tools, scripts, and other executables are denied."
    )


def patch_is_limited_to_architecture_artifacts(
    tool_input: object,
    workspace: Path | None = None,
) -> bool:
    patch = patch_text_from_tool_input(tool_input)
    if patch is None:
        if not isinstance(tool_input, dict):
            return False
        targets = tuple(
            tool_input[key]
            for key in ("file_path", "path", "target")
            if isinstance(tool_input.get(key), str)
        )
        return len(targets) == 1 and is_canonical_artifact_path(targets[0], workspace)
    targets = tuple(
        (match.group(1) or match.group(2)).strip().replace("\\", "/")
        for match in PATCH_FILE_PATTERN.finditer(patch)
    )
    return bool(targets) and all(
        is_canonical_artifact_path(target, workspace)
        for target in targets
    )


def tool_denial_reason(
    context: CodexTurnContext,
    tool_name_value: object,
    tool_input: object = None,
    workspace: Path | None = None,
    plugin_root: Path | None = None,
) -> str | None:
    if not context.active:
        return None
    local_tool_name = _normalized_local_tool_name(tool_name_value)
    if local_tool_name in WEB_LOOKUP_TOOL_NAMES:
        return (
            "AI Software Architect canonical references are bundled with the plugin; "
            "web lookup is disabled during the architecture workflow. Use the "
            "reference paths supplied by the active skill context."
        )
    if local_tool_name in SHELL_TOOL_NAMES:
        reason = _shell_denial_reason(tool_input, plugin_root=plugin_root)
        if reason is not None:
            return reason
    if local_tool_name in PATCH_TOOL_NAMES:
        if not patch_is_limited_to_architecture_artifacts(tool_input, workspace):
            return (
                "The AI Software Architect never writes application code. Its patch "
                "surface is limited to `.ai-architect/project-context.md`, "
                "`.ai-architect/architecture-contract.yaml`, "
                "`.ai-architect/implementation-plan.md`, and canonical "
                "`.ai-architect/decisions/ADR-NNN[-slug].md` files."
            )
    return None
