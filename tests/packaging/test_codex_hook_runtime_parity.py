# SPDX-FileCopyrightText: 2026 Leonardo Muffato (AUTOSOFT Engineering - www.autosoft-engineering.de)
# SPDX-License-Identifier: MIT

from pathlib import Path

import pytest

from adapters.codex.hook_entry import handle_hook


def test_pre_tool_use_without_turn_id_fails_closed_before_dispatch(
    tmp_path: Path,
) -> None:
    payload = {
        "hook_event_name": "PreToolUse",
        "session_id": "session-1",
        "tool_name": "bash",
        "tool_input": {"command": "git status"},
    }

    with pytest.raises(ValueError, match="turn_id"):
        handle_hook(payload, tmp_path, expected_event="PreToolUse")
