# SPDX-FileCopyrightText: 2026 Leonardo Muffato (AUTOSOFT Engineering - www.autosoft-engineering.de)
# SPDX-License-Identifier: MIT

"""Typed, single-use continuation state for Codex architecture sessions."""

from __future__ import annotations

import hashlib
import json
import os
import time
from collections.abc import Callable
from dataclasses import asdict, dataclass
from enum import StrEnum
from pathlib import Path

from pydantic import BaseModel, ConfigDict, Field

try:
    from adapters.codex.activation_policy import CodexTurnContext, CodexTurnRoute
except ModuleNotFoundError as exc:
    if exc.name != "adapters":
        raise
    from activation_policy import (  # type: ignore[import-not-found, no-redef]
        CodexTurnContext,
        CodexTurnRoute,
    )


class PendingInteraction(StrEnum):
    CLARIFICATION = "clarification"
    DECISION = "decision"


class WorkflowPhase(StrEnum):
    CLARIFY = "clarify"
    APPROVE = "approve"


class ApprovalTransition(StrEnum):
    RESUME_DESIGN = "resume_design"
    RECORD_AND_HANDOFF = "record_and_handoff"


class CheckpointPhase(StrEnum):
    ACTIVE = "active"
    CLARIFY = "clarify"
    AWAIT_DECISION = "await_decision"
    DECISION_RESPONSE = "decision_response"
    RECORD_AND_HANDOFF = "record_and_handoff"
    COMPLETE = "complete"


class WorkflowCheckpoint(BaseModel):
    """Minimal durable state that survives turn boundaries and compaction."""

    model_config = ConfigDict(extra="forbid", strict=True)

    phase: CheckpointPhase
    expected_artifacts: list[str] = Field(default_factory=list, max_length=4)
    artifact_bundle_validated: bool = False


@dataclass(frozen=True)
class SessionContinuation:
    context: CodexTurnContext
    interaction: PendingInteraction
    phase: WorkflowPhase
    approval_transition: ApprovalTransition


class ContinuationManager:
    def __init__(
        self,
        plugin_data: Path,
        *,
        max_age_seconds: int,
        clock: Callable[[], float] = time.time,
    ) -> None:
        self._root = plugin_data / "control-plane"
        self._max_age_seconds = max_age_seconds
        self._clock = clock

    def _path(self, session_id: str) -> Path:
        digest = hashlib.sha256(session_id.encode("utf-8", errors="replace")).hexdigest()
        return self._root / f"continuation-{digest}.json"

    def open(self, session_id: str, continuation: SessionContinuation) -> None:
        path = self._path(session_id)
        path.parent.mkdir(parents=True, exist_ok=True)
        temporary = path.with_suffix(".tmp")
        temporary.write_text(
            json.dumps(asdict(continuation), sort_keys=True) + "\n",
            encoding="utf-8",
            newline="\n",
        )
        temporary.replace(path)
        now = self._clock()
        os.utime(path, (now, now))

    def consume(self, session_id: str) -> SessionContinuation | None:
        path = self._path(session_id)
        try:
            raw = json.loads(path.read_text("utf-8"))
            if self._clock() - path.stat().st_mtime > self._max_age_seconds:
                return None
            context_raw = raw["context"]
            return SessionContinuation(
                context=CodexTurnContext(
                    active=context_raw["active"],
                    route=CodexTurnRoute(context_raw["route"]),
                    reference_paths=tuple(context_raw.get("reference_paths", ())),
                ),
                interaction=PendingInteraction(raw["interaction"]),
                phase=WorkflowPhase(raw["phase"]),
                approval_transition=ApprovalTransition(raw["approval_transition"]),
            )
        except (OSError, KeyError, TypeError, ValueError, json.JSONDecodeError):
            return None
        finally:
            path.unlink(missing_ok=True)

    def cancel(self, session_id: str) -> None:
        self._path(session_id).unlink(missing_ok=True)


class WorkflowCheckpointManager:
    def __init__(
        self,
        plugin_data: Path,
        *,
        clock: Callable[[], float] = time.time,
    ) -> None:
        self._root = plugin_data / "control-plane"
        self._clock = clock

    def _path(self, session_id: str) -> Path:
        digest = hashlib.sha256(session_id.encode("utf-8", errors="replace")).hexdigest()
        return self._root / f"workflow-{digest}.json"

    def save(self, session_id: str, checkpoint: WorkflowCheckpoint) -> None:
        path = self._path(session_id)
        path.parent.mkdir(parents=True, exist_ok=True)
        temporary = path.with_suffix(".tmp")
        temporary.write_text(
            checkpoint.model_dump_json() + "\n",
            encoding="utf-8",
            newline="\n",
        )
        temporary.replace(path)
        now = self._clock()
        os.utime(path, (now, now))

    def load(self, session_id: str) -> WorkflowCheckpoint | None:
        path = self._path(session_id)
        try:
            return WorkflowCheckpoint.model_validate_json(path.read_text("utf-8"))
        except (OSError, ValueError):
            return None

    def clear(self, session_id: str) -> None:
        self._path(session_id).unlink(missing_ok=True)
