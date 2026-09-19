# SPDX-FileCopyrightText: 2026 Leonardo Muffato (AUTOSOFT Engineering - www.autosoft-engineering.de)
# SPDX-License-Identifier: MIT

"""State seams for the Codex hook lifecycle."""

from __future__ import annotations

import hashlib
import json
import os
import time
from collections.abc import Callable
from dataclasses import asdict
from pathlib import Path
from typing import Protocol

try:
    from adapters.codex.activation_policy import CodexTurnContext, CodexTurnRoute
    from adapters.codex.continuation import (
        ContinuationManager,
        SessionContinuation,
        WorkflowCheckpoint,
        WorkflowCheckpointManager,
    )
except ModuleNotFoundError as exc:
    if exc.name != "adapters":
        raise
    from activation_policy import (  # type: ignore[import-not-found, no-redef]
        CodexTurnContext,
        CodexTurnRoute,
    )
    from continuation import (  # type: ignore[import-not-found, no-redef]
        ContinuationManager,
        SessionContinuation,
        WorkflowCheckpoint,
        WorkflowCheckpointManager,
    )

Clock = Callable[[], float]


class StateStore(Protocol):
    """Minimal persistence interface owned by the lifecycle implementation."""

    def load_context(self, session_id: str, turn_id: str) -> CodexTurnContext: ...

    def save_context(
        self,
        session_id: str,
        turn_id: str,
        context: CodexTurnContext,
    ) -> None: ...

    def clear_context(self, session_id: str, turn_id: str) -> None: ...

    def open_continuation(
        self,
        session_id: str,
        continuation: SessionContinuation,
    ) -> None: ...

    def consume_continuation(self, session_id: str) -> SessionContinuation | None: ...

    def cancel_continuation(self, session_id: str) -> None: ...

    def load_checkpoint(self, session_id: str) -> WorkflowCheckpoint | None: ...

    def save_checkpoint(
        self,
        session_id: str,
        checkpoint: WorkflowCheckpoint,
    ) -> None: ...

    def cleanup(self) -> None: ...


def _inactive_context() -> CodexTurnContext:
    return CodexTurnContext(active=False, route=CodexTurnRoute.INACTIVE)


class FileStateStore:
    """Production adapter retaining the established hashed control-plane files."""

    def __init__(
        self,
        plugin_data: Path,
        *,
        clock: Clock = time.time,
        max_state_age_seconds: int = 86_400,
        max_continuation_age_seconds: int = 3_600,
        max_state_files: int = 512,
    ) -> None:
        self._plugin_data = plugin_data
        self._clock = clock
        self._max_state_age_seconds = max_state_age_seconds
        self._max_state_files = max_state_files
        self._continuations = ContinuationManager(
            plugin_data,
            max_age_seconds=max_continuation_age_seconds,
            clock=clock,
        )
        self._checkpoints = WorkflowCheckpointManager(plugin_data, clock=clock)

    def _turn_state_path(self, session_id: str, turn_id: str) -> Path:
        if not session_id:
            raise ValueError("hook session_id is unavailable")
        if not turn_id:
            raise ValueError("hook turn_id is unavailable")
        identity = f"{session_id}\0{turn_id}"
        digest = hashlib.sha256(identity.encode("utf-8", errors="replace")).hexdigest()
        return self._plugin_data / "control-plane" / f"turn-{digest}.json"

    def load_context(self, session_id: str, turn_id: str) -> CodexTurnContext:
        try:
            raw = json.loads(self._turn_state_path(session_id, turn_id).read_text("utf-8"))
            return CodexTurnContext(
                active=raw["active"],
                route=CodexTurnRoute(raw["route"]),
                reference_paths=tuple(raw.get("reference_paths", ())),
            )
        except (
            OSError,
            KeyError,
            TypeError,
            ValueError,
            json.JSONDecodeError,
        ):
            return _inactive_context()

    def save_context(
        self,
        session_id: str,
        turn_id: str,
        context: CodexTurnContext,
    ) -> None:
        path = self._turn_state_path(session_id, turn_id)
        path.parent.mkdir(parents=True, exist_ok=True)
        temporary = path.with_suffix(".tmp")
        temporary.write_text(
            json.dumps(asdict(context), sort_keys=True) + "\n",
            encoding="utf-8",
            newline="\n",
        )
        temporary.replace(path)
        now = self._clock()
        os.utime(path, (now, now))

    def clear_context(self, session_id: str, turn_id: str) -> None:
        try:
            self._turn_state_path(session_id, turn_id).unlink(missing_ok=True)
        except (OSError, ValueError):
            pass

    def open_continuation(
        self,
        session_id: str,
        continuation: SessionContinuation,
    ) -> None:
        self._continuations.open(session_id, continuation)

    def consume_continuation(self, session_id: str) -> SessionContinuation | None:
        return self._continuations.consume(session_id)

    def cancel_continuation(self, session_id: str) -> None:
        self._continuations.cancel(session_id)

    def load_checkpoint(self, session_id: str) -> WorkflowCheckpoint | None:
        return self._checkpoints.load(session_id)

    def save_checkpoint(
        self,
        session_id: str,
        checkpoint: WorkflowCheckpoint,
    ) -> None:
        self._checkpoints.save(session_id, checkpoint)

    def cleanup(self) -> None:
        state_root = self._plugin_data / "control-plane"
        try:
            candidates = [
                (path.stat().st_mtime, path)
                for path in state_root.glob("*.json")
                if path.is_file()
            ]
        except OSError:
            return
        current_time = self._clock()
        candidates.sort(reverse=True)
        for index, (modified, path) in enumerate(candidates):
            if (
                index >= self._max_state_files
                or current_time - modified > self._max_state_age_seconds
            ):
                try:
                    path.unlink(missing_ok=True)
                except OSError:
                    pass


class InMemoryStateStore:
    """Deterministic test adapter with the same lifecycle operations as the file store."""

    def __init__(
        self,
        *,
        clock: Clock = time.time,
        max_state_age_seconds: int = 86_400,
        max_continuation_age_seconds: int = 3_600,
        max_state_files: int = 512,
    ) -> None:
        self._clock = clock
        self._max_state_age_seconds = max_state_age_seconds
        self._max_continuation_age_seconds = max_continuation_age_seconds
        self._max_state_files = max_state_files
        self._contexts: dict[tuple[str, str], tuple[float, CodexTurnContext]] = {}
        self._continuations: dict[str, tuple[float, SessionContinuation]] = {}
        self._checkpoints: dict[str, WorkflowCheckpoint] = {}

    def load_context(self, session_id: str, turn_id: str) -> CodexTurnContext:
        stored = self._contexts.get((session_id, turn_id))
        if stored is None:
            return _inactive_context()
        modified, context = stored
        if self._clock() - modified > self._max_state_age_seconds:
            self._contexts.pop((session_id, turn_id), None)
            return _inactive_context()
        return context

    def save_context(
        self,
        session_id: str,
        turn_id: str,
        context: CodexTurnContext,
    ) -> None:
        self._contexts[(session_id, turn_id)] = (self._clock(), context)

    def clear_context(self, session_id: str, turn_id: str) -> None:
        self._contexts.pop((session_id, turn_id), None)

    def open_continuation(
        self,
        session_id: str,
        continuation: SessionContinuation,
    ) -> None:
        self._continuations[session_id] = (self._clock(), continuation)

    def consume_continuation(self, session_id: str) -> SessionContinuation | None:
        stored = self._continuations.pop(session_id, None)
        if stored is None:
            return None
        modified, continuation = stored
        if self._clock() - modified > self._max_continuation_age_seconds:
            return None
        return continuation

    def cancel_continuation(self, session_id: str) -> None:
        self._continuations.pop(session_id, None)

    def load_checkpoint(self, session_id: str) -> WorkflowCheckpoint | None:
        return self._checkpoints.get(session_id)

    def save_checkpoint(
        self,
        session_id: str,
        checkpoint: WorkflowCheckpoint,
    ) -> None:
        self._checkpoints[session_id] = checkpoint

    def cleanup(self) -> None:
        current_time = self._clock()
        stale_contexts = [
            key
            for key, (modified, _) in self._contexts.items()
            if current_time - modified > self._max_state_age_seconds
        ]
        for key in stale_contexts:
            self._contexts.pop(key, None)
        if len(self._contexts) > self._max_state_files:
            oldest = sorted(
                self._contexts.items(),
                key=lambda item: item[1][0],
            )
            for key, _ in oldest[: len(self._contexts) - self._max_state_files]:
                self._contexts.pop(key, None)


FileBackedStateStore = FileStateStore


__all__ = [
    "Clock",
    "FileBackedStateStore",
    "FileStateStore",
    "InMemoryStateStore",
    "StateStore",
]
