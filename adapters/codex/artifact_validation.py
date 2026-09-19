# SPDX-FileCopyrightText: 2026 Leonardo Muffato (AUTOSOFT Engineering - www.autosoft-engineering.de)
# SPDX-License-Identifier: MIT

"""Small seams shared by artifact reconstruction and tool-surface policy."""

from __future__ import annotations


def patch_text_from_tool_input(value: object) -> str | None:
    """Return exactly one complete apply-patch payload, if one is present."""

    if isinstance(value, str):
        return value if "*** Begin Patch" in value else None
    candidates: list[str] = []

    def collect(current: object, depth: int = 0) -> None:
        if depth > 4:
            return
        if isinstance(current, str):
            if "*** Begin Patch" in current and "*** End Patch" in current:
                candidates.append(current)
            return
        if isinstance(current, dict):
            for nested in current.values():
                collect(nested, depth + 1)
        elif isinstance(current, list):
            for nested in current:
                collect(nested, depth + 1)

    collect(value)
    return candidates[0] if len(candidates) == 1 else None
