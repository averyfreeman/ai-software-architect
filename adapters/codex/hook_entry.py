# SPDX-FileCopyrightText: 2026 Leonardo Muffato (AUTOSOFT Engineering - www.autosoft-engineering.de)
# SPDX-License-Identifier: MIT

"""Codex edge adapter for the typed hook lifecycle."""

from __future__ import annotations

import json
import os
import sys
import time
from pathlib import Path
from typing import Any

try:
    from adapters.codex.hook_lifecycle import (
        CodexResponseAdapter,
        HookEvent,
        HookLifecycle,
    )
    from adapters.codex.hook_models import HookPayload
    from adapters.codex.state_store import FileStateStore
except ModuleNotFoundError as exc:
    if exc.name != "adapters":
        raise
    from hook_lifecycle import (  # type: ignore[import-not-found, no-redef]
        CodexResponseAdapter,
        HookEvent,
        HookLifecycle,
    )
    from hook_models import HookPayload  # type: ignore[import-not-found, no-redef]
    from state_store import FileStateStore  # type: ignore[import-not-found, no-redef]

MAX_HOOK_INPUT_BYTES = 1_000_000
MAX_STATE_AGE_SECONDS = 86_400
MAX_CONTINUATION_AGE_SECONDS = 3_600
MAX_STATE_FILES = 512


def _production_lifecycle(
    plugin_data: Path,
    plugin_root: Path | None = None,
) -> HookLifecycle:
    return HookLifecycle(
        FileStateStore(
            plugin_data,
            max_state_age_seconds=MAX_STATE_AGE_SECONDS,
            max_continuation_age_seconds=MAX_CONTINUATION_AGE_SECONDS,
            max_state_files=MAX_STATE_FILES,
        ),
        plugin_root=plugin_root,
    )


def _dispatch(
    payload: dict[str, Any],
    plugin_data: Path,
    plugin_root: Path | None,
    expected_event: str,
) -> dict[str, Any]:
    HookPayload.model_validate(payload)
    event = HookEvent.from_payload(payload, expected_event)
    outcome = _production_lifecycle(plugin_data, plugin_root).handle(event)
    return CodexResponseAdapter.render(outcome)


def handle_user_prompt_submit(
    payload: dict[str, Any],
    plugin_data: Path,
    plugin_root: Path | None = None,
) -> dict[str, Any]:
    """Compatibility adapter for the UserPromptSubmit hook entry point."""

    return _dispatch(payload, plugin_data, plugin_root, "UserPromptSubmit")


def handle_pre_tool_use(
    payload: dict[str, Any],
    plugin_data: Path,
    plugin_root: Path | None = None,
) -> dict[str, Any]:
    """Compatibility adapter for the PreToolUse hook entry point."""

    return _dispatch(payload, plugin_data, plugin_root, "PreToolUse")


def handle_post_tool_use(
    payload: dict[str, Any],
    plugin_data: Path,
) -> dict[str, Any]:
    """Compatibility adapter for the PostToolUse hook entry point."""

    return _dispatch(payload, plugin_data, None, "PostToolUse")


def handle_post_compact(
    payload: dict[str, Any],
    plugin_data: Path,
) -> dict[str, Any]:
    """Compatibility adapter for the PostCompact hook entry point."""

    return _dispatch(payload, plugin_data, None, "PostCompact")


def handle_stop(
    payload: dict[str, Any],
    plugin_data: Path,
) -> dict[str, Any]:
    """Compatibility adapter for the Stop hook entry point."""

    return _dispatch(payload, plugin_data, None, "Stop")


def _cleanup_stale_contexts(
    plugin_data: Path,
    *,
    now: float | None = None,
) -> None:
    """Compatibility helper retained for callers migrating to ``FileStateStore``."""

    clock = time.time if now is None else lambda: now
    FileStateStore(
        plugin_data,
        clock=clock,
        max_state_age_seconds=MAX_STATE_AGE_SECONDS,
        max_continuation_age_seconds=MAX_CONTINUATION_AGE_SECONDS,
        max_state_files=MAX_STATE_FILES,
    ).cleanup()


__all__ = [
    "MAX_CONTINUATION_AGE_SECONDS",
    "MAX_HOOK_INPUT_BYTES",
    "MAX_STATE_AGE_SECONDS",
    "MAX_STATE_FILES",
    "handle_hook",
    "handle_post_compact",
    "handle_post_tool_use",
    "handle_pre_tool_use",
    "handle_stop",
    "handle_user_prompt_submit",
    "main",
]


def handle_hook(
    payload: dict[str, Any],
    plugin_data: Path,
    plugin_root: Path | None = None,
    expected_event: str | None = None,
) -> dict[str, Any]:
    """Validate one raw payload and dispatch it through one lifecycle instance."""

    HookPayload.model_validate(payload)
    event = HookEvent.from_payload(payload, expected_event)
    outcome = _production_lifecycle(plugin_data, plugin_root).handle(event)
    return CodexResponseAdapter.render(outcome)


def _runtime_failure_response(expected_event: str | None, diagnostic: str) -> dict[str, Any]:
    message = (
        "AI Software Architect control-plane validation failed; the protected operation "
        "was not allowed."
        + diagnostic
    )
    if expected_event == "PreToolUse":
        return {
            "hookSpecificOutput": {
                "hookEventName": "PreToolUse",
                "permissionDecision": "deny",
                "permissionDecisionReason": message,
            },
            "systemMessage": message,
        }
    if expected_event == "PostToolUse":
        return {
            "continue": False,
            "stopReason": message,
            "systemMessage": message,
        }
    return {"systemMessage": message}


def main(expected_event: str | None = None) -> None:
    """Read one bounded hook event from stdin and emit one JSON response."""

    try:
        raw = sys.stdin.buffer.read(MAX_HOOK_INPUT_BYTES + 1)
        if len(raw) > MAX_HOOK_INPUT_BYTES:
            raise ValueError("hook input exceeds the bounded input size")
        payload = json.loads(raw.decode("utf-8"))
        if not isinstance(payload, dict):
            raise TypeError("hook input must be a JSON object")
        plugin_data_text = os.environ.get("PLUGIN_DATA")
        if not plugin_data_text:
            raise ValueError("PLUGIN_DATA is unavailable")
        plugin_root_text = os.environ.get("PLUGIN_ROOT")
        plugin_root = Path(plugin_root_text) if plugin_root_text else None
        response = handle_hook(
            payload,
            Path(plugin_data_text),
            plugin_root,
            expected_event=expected_event,
        )
    except Exception as exc:
        diagnostic = ""
        if os.environ.get("AI_ARCHITECT_HOOK_DEBUG") == "1":
            diagnostic = f" Diagnostic: {type(exc).__name__}: {exc}"
        response = _runtime_failure_response(expected_event, diagnostic)
    sys.stdout.write(json.dumps(response, separators=(",", ":")))
    sys.stdout.flush()


if __name__ == "__main__":
    main()
