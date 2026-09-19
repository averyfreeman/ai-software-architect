# SPDX-FileCopyrightText: 2026 Leonardo Muffato (AUTOSOFT Engineering - www.autosoft-engineering.de)
# SPDX-License-Identifier: MIT

"""Typed lifecycle for the five Codex architecture control-plane hook events."""

from __future__ import annotations

import time
from collections.abc import Mapping
from dataclasses import dataclass, field
from pathlib import Path
from typing import Any, Literal

try:
    from adapters.codex.activation_policy import (
        MISSING_INVOCATION_GUIDANCE,
        CodexTurnContext,
        CodexTurnRoute,
        artifact_authoring_resource_path,
        bundled_reference_content,
        classify_prompt,
        comparison_bundle_path,
        developer_context,
        has_other_activation,
        repository_snapshot_command,
        with_reference_hints,
    )
    from adapters.codex.artifact_guard import (
        architecture_artifact_denial_reason,
        proposed_artifact_candidates,
        validate_artifact_bundle_candidates,
    )
    from adapters.codex.continuation import (
        CheckpointPhase,
        PendingInteraction,
        SessionContinuation,
        WorkflowCheckpoint,
    )
    from adapters.codex.response_policy import (
        continuation_instruction,
        final_response_violations,
        pending_continuation,
    )
    from adapters.codex.state_store import (
        FileStateStore,
        InMemoryStateStore,
        StateStore,
    )
    from adapters.codex.tool_policy import tool_denial_reason
except ModuleNotFoundError as exc:
    if exc.name != "adapters":
        raise
    from activation_policy import (  # type: ignore[import-not-found, no-redef]
        MISSING_INVOCATION_GUIDANCE,
        CodexTurnContext,
        CodexTurnRoute,
        artifact_authoring_resource_path,
        bundled_reference_content,
        classify_prompt,
        comparison_bundle_path,
        developer_context,
        has_other_activation,
        repository_snapshot_command,
        with_reference_hints,
    )
    from artifact_guard import (  # type: ignore[import-not-found, no-redef]
        architecture_artifact_denial_reason,
        proposed_artifact_candidates,
        validate_artifact_bundle_candidates,
    )
    from continuation import (  # type: ignore[import-not-found, no-redef]
        CheckpointPhase,
        PendingInteraction,
        SessionContinuation,
        WorkflowCheckpoint,
    )
    from response_policy import (  # type: ignore[import-not-found, no-redef]
        continuation_instruction,
        final_response_violations,
        pending_continuation,
    )
    from state_store import (  # type: ignore[import-not-found, no-redef]
        FileStateStore,
        InMemoryStateStore,
        StateStore,
    )
    from tool_policy import tool_denial_reason  # type: ignore[import-not-found, no-redef]

HookEventName = Literal[
    "UserPromptSubmit",
    "PreToolUse",
    "PostToolUse",
    "PostCompact",
    "Stop",
]


def _required_session_id(payload: Mapping[str, object]) -> str:
    value = payload.get("session_id")
    if not isinstance(value, str) or not value:
        raise ValueError("hook session_id is unavailable")
    return value


def _optional_turn_id(payload: Mapping[str, object]) -> str | None:
    value = payload.get("turn_id")
    if value is None:
        return None
    if not isinstance(value, str):
        raise ValueError("hook turn_id is invalid")
    return value


def _inactive_context() -> CodexTurnContext:
    return CodexTurnContext(active=False, route=CodexTurnRoute.INACTIVE)


class HookEvent:
    """Typed normalization seam for raw, already edge-validated hook payloads."""

    @classmethod
    def from_payload(
        cls,
        payload: Mapping[str, object],
        expected_event: str | None = None,
    ) -> HookEventValue:
        event_name = payload.get("hook_event_name")
        if not isinstance(event_name, str):
            raise ValueError("hook_event_name is unavailable")
        if expected_event is not None and event_name != expected_event:
            raise ValueError("hook event does not match the invoked hook entry point")
        session_id = _required_session_id(payload)
        turn_id = _optional_turn_id(payload)
        cwd = payload.get("cwd")
        if cwd is not None and not isinstance(cwd, str):
            raise ValueError("hook cwd is invalid")
        if event_name != "PostCompact" and turn_id is None:
            raise ValueError("hook turn_id is unavailable")
        if event_name == "UserPromptSubmit":
            prompt = payload.get("prompt")
            return UserPromptSubmitEvent(
                session_id=session_id,
                turn_id=turn_id,
                cwd=cwd,
                prompt=prompt if isinstance(prompt, str) else "",
            )
        if event_name == "PreToolUse":
            return PreToolUseEvent(
                session_id=session_id,
                turn_id=turn_id,
                cwd=cwd,
                tool_name=payload.get("tool_name"),
                tool_input=payload.get("tool_input"),
            )
        if event_name == "PostToolUse":
            return PostToolUseEvent(
                session_id=session_id,
                turn_id=turn_id,
                cwd=cwd,
                tool_name=payload.get("tool_name"),
                tool_input=payload.get("tool_input"),
            )
        if event_name == "PostCompact":
            return PostCompactEvent(session_id=session_id, turn_id=turn_id, cwd=cwd)
        if event_name == "Stop":
            message = payload.get("last_assistant_message")
            return StopEvent(
                session_id=session_id,
                turn_id=turn_id,
                cwd=cwd,
                last_assistant_message=message if isinstance(message, str) else "",
                stop_hook_active=payload.get("stop_hook_active") is True,
            )
        raise ValueError(f"unsupported hook event: {event_name}")


@dataclass(frozen=True)
class UserPromptSubmitEvent(HookEvent):
    session_id: str
    turn_id: str | None
    cwd: str | None
    prompt: str


@dataclass(frozen=True)
class PreToolUseEvent(HookEvent):
    session_id: str
    turn_id: str | None
    cwd: str | None
    tool_name: object
    tool_input: object


@dataclass(frozen=True)
class PostToolUseEvent(HookEvent):
    session_id: str
    turn_id: str | None
    cwd: str | None
    tool_name: object
    tool_input: object


@dataclass(frozen=True)
class PostCompactEvent(HookEvent):
    session_id: str
    turn_id: str | None
    cwd: str | None


@dataclass(frozen=True)
class StopEvent(HookEvent):
    session_id: str
    turn_id: str | None
    cwd: str | None
    last_assistant_message: str
    stop_hook_active: bool


HookEventValue = (
    UserPromptSubmitEvent
    | PreToolUseEvent
    | PostToolUseEvent
    | PostCompactEvent
    | StopEvent
)


@dataclass(frozen=True)
class NoOpOutcome:
    pass


@dataclass(frozen=True)
class ContextInjectionOutcome:
    hook_event_name: HookEventName
    additional_context: str


@dataclass(frozen=True)
class DenialOutcome:
    hook_event_name: Literal["PreToolUse"]
    reason: str


@dataclass(frozen=True)
class BlockingOutcome:
    reason: str


@dataclass(frozen=True)
class StopOutcome:
    stop_reason: str
    system_message: str


HookOutcome = (
    NoOpOutcome
    | ContextInjectionOutcome
    | DenialOutcome
    | BlockingOutcome
    | StopOutcome
)


class CodexResponseAdapter:
    """Render typed outcomes into the stable Codex hook dictionary contract."""

    @staticmethod
    def render(outcome: HookOutcome) -> dict[str, Any]:
        if isinstance(outcome, NoOpOutcome):
            return {}
        if isinstance(outcome, ContextInjectionOutcome):
            return {
                "hookSpecificOutput": {
                    "hookEventName": outcome.hook_event_name,
                    "additionalContext": outcome.additional_context,
                }
            }
        if isinstance(outcome, DenialOutcome):
            return {
                "hookSpecificOutput": {
                    "hookEventName": outcome.hook_event_name,
                    "permissionDecision": "deny",
                    "permissionDecisionReason": outcome.reason,
                }
            }
        if isinstance(outcome, BlockingOutcome):
            return {"decision": "block", "reason": outcome.reason}
        return {
            "continue": False,
            "stopReason": outcome.stop_reason,
            "systemMessage": outcome.system_message,
        }


@dataclass
class LifecycleState:
    context: CodexTurnContext = field(
        default_factory=lambda: CodexTurnContext(
            active=False,
            route=CodexTurnRoute.INACTIVE,
        )
    )
    continuation: SessionContinuation | None = None
    checkpoint: WorkflowCheckpoint | None = None


class HookLifecycle:
    """Own event ordering, persisted effects, cleanup, and terminal behavior."""

    def __init__(
        self,
        state_store: StateStore,
        *,
        plugin_root: Path | None = None,
        clock: Any = time.time,
    ) -> None:
        self._state_store = state_store
        self._plugin_root = plugin_root
        self._clock = clock
        self.state = LifecycleState()

    def handle(self, event: HookEventValue) -> HookOutcome:
        if isinstance(event, UserPromptSubmitEvent):
            return self._handle_user_prompt_submit(event)
        if isinstance(event, PreToolUseEvent):
            return self._handle_pre_tool_use(event)
        if isinstance(event, PostToolUseEvent):
            return self._handle_post_tool_use(event)
        if isinstance(event, PostCompactEvent):
            return self._handle_post_compact(event)
        if isinstance(event, StopEvent):
            return self._handle_stop(event)
        raise TypeError(f"unsupported lifecycle event: {type(event).__name__}")

    def _set_state(
        self,
        *,
        context: CodexTurnContext | None = None,
        continuation: SessionContinuation | None = None,
        checkpoint: WorkflowCheckpoint | None = None,
    ) -> None:
        if context is not None:
            self.state.context = context
        self.state.continuation = continuation
        self.state.checkpoint = checkpoint

    def _handle_user_prompt_submit(
        self,
        event: UserPromptSubmitEvent,
    ) -> HookOutcome:
        context = with_reference_hints(classify_prompt(event.prompt), event.prompt)
        continuation: SessionContinuation | None = None
        if not context.active:
            if has_other_activation(event.prompt):
                self._state_store.cancel_continuation(event.session_id)
                self._set_state(context=context)
                return NoOpOutcome()
            continuation = self._state_store.consume_continuation(event.session_id)
            if continuation is None:
                self._set_state(context=context)
                return NoOpOutcome()
            context = continuation.context
        if context.route == CodexTurnRoute.MISSING_SKILL_INVOCATION:
            self._state_store.cancel_continuation(event.session_id)
            self._state_store.cleanup()
            self._set_state(context=context, continuation=None)
            return BlockingOutcome(reason=MISSING_INVOCATION_GUIDANCE)

        if continuation is None:
            self._state_store.cancel_continuation(event.session_id)
        checkpoint = (
            WorkflowCheckpoint(phase=CheckpointPhase.ACTIVE)
            if continuation is None or continuation.interaction != PendingInteraction.DECISION
            else WorkflowCheckpoint(
                phase=CheckpointPhase.DECISION_RESPONSE,
                expected_artifacts=["adr", "contract", "context", "implementation-plan"],
            )
        )
        self._state_store.save_checkpoint(event.session_id, checkpoint)
        if event.turn_id is None:
            raise ValueError("hook turn_id is unavailable")
        self._state_store.save_context(event.session_id, event.turn_id, context)
        self._state_store.cleanup()
        additional_context = developer_context(
            context,
            continued=continuation is not None,
            continuation_instruction=(
                continuation_instruction(continuation)
                if continuation is not None
                else ""
            ),
            continuation_interaction=(
                continuation.interaction.value if continuation is not None else None
            ),
            snapshot_command=(
                repository_snapshot_command(self._plugin_root)
                if self._plugin_root is not None
                else ""
            ),
            comparison_bundle_path=comparison_bundle_path(self._plugin_root),
            bundled_reference_content=bundled_reference_content(
                self._plugin_root,
                context,
            ),
        )
        if (
            self._plugin_root is not None
            and continuation is not None
            and continuation.interaction == PendingInteraction.DECISION
        ):
            additional_context += (
                " Exact installed record-and-handoff resource path (authoritative; read "
                "this generated bundle once with a host-native static file tool and do not "
                "search for or separately load its canonical source files): "
                + artifact_authoring_resource_path(self._plugin_root)
                + "."
            )
        self._set_state(
            context=context,
            continuation=continuation,
            checkpoint=checkpoint,
        )
        return ContextInjectionOutcome("UserPromptSubmit", additional_context)

    def _handle_pre_tool_use(self, event: PreToolUseEvent) -> HookOutcome:
        context = self._state_store.load_context(event.session_id, event.turn_id or "")
        self._set_state(context=context)
        workspace = Path(event.cwd) if event.cwd else None
        reason = tool_denial_reason(
            context,
            event.tool_name,
            event.tool_input,
            workspace=workspace,
            plugin_root=self._plugin_root,
        )
        if reason is None and context.active:
            local_name = str(event.tool_name).rsplit(".", 1)[-1].casefold()
            if local_name in {"apply_patch", "edit", "write"}:
                if workspace is None:
                    reason = (
                        "AI Software Architect denied the architecture artifact write because "
                        "Codex did not provide a trustworthy workspace root."
                    )
                else:
                    checkpoint = self._state_store.load_checkpoint(event.session_id)
                    self.state.checkpoint = checkpoint
                    reason = architecture_artifact_denial_reason(
                        event.tool_input,
                        workspace,
                        require_complete_bundle=(
                            checkpoint is not None
                            and checkpoint.phase == CheckpointPhase.DECISION_RESPONSE
                        ),
                    )
        if reason is None:
            return NoOpOutcome()
        return DenialOutcome("PreToolUse", reason)

    def _handle_post_tool_use(self, event: PostToolUseEvent) -> HookOutcome:
        context = self._state_store.load_context(event.session_id, event.turn_id or "")
        self._set_state(context=context)
        if not context.active:
            return NoOpOutcome()
        local_name = str(event.tool_name).rsplit(".", 1)[-1].casefold()
        if local_name not in {"apply_patch", "edit", "write"}:
            return NoOpOutcome()
        if not event.cwd:
            return StopOutcome(
                stop_reason=(
                    "AI Software Architect stopped because Codex did not provide a "
                    "trustworthy workspace root for post-write verification."
                ),
                system_message="Architecture artifact post-write verification failed.",
            )
        workspace = Path(event.cwd)
        try:
            candidates = proposed_artifact_candidates(event.tool_input, workspace)
            if not candidates:
                raise ValueError("no reconstructable architecture artifact candidate")
            for candidate in candidates:
                persisted = (workspace / candidate.path).read_text("utf-8")
                if persisted != candidate.content:
                    raise ValueError(
                        "persisted content differs from validated candidate: "
                        f"{candidate.path.as_posix()}"
                    )
            bundle = validate_artifact_bundle_candidates(candidates)
        except (OSError, UnicodeError, ValueError) as exc:
            return StopOutcome(
                stop_reason=(
                    "AI Software Architect stopped because post-write verification failed: "
                    f"{exc}. Review the repository before continuing."
                ),
                system_message="Architecture artifact post-write verification failed.",
            )
        if bundle is None:
            return NoOpOutcome()
        checkpoint = WorkflowCheckpoint(
            phase=CheckpointPhase.COMPLETE,
            expected_artifacts=["adr", "contract", "context", "implementation-plan"],
            artifact_bundle_validated=True,
        )
        self._state_store.save_checkpoint(event.session_id, checkpoint)
        self.state.checkpoint = checkpoint
        return ContextInjectionOutcome(
            "PostToolUse",
            "The persisted ADR, architecture contract, project context, and coding "
            "handoff exactly match the pre-write validated bundle. State that the "
            "architecture artifacts were recorded and that application source code "
            "was not authorized by this workflow.",
        )

    def _handle_post_compact(self, event: PostCompactEvent) -> HookOutcome:
        checkpoint = self._state_store.load_checkpoint(event.session_id)
        self.state.checkpoint = checkpoint
        if checkpoint is None or checkpoint.phase == CheckpointPhase.COMPLETE:
            return NoOpOutcome()
        expected = ", ".join(checkpoint.expected_artifacts) or "none"
        return ContextInjectionOutcome(
            "PostCompact",
            "AI Software Architect typed workflow checkpoint: phase="
            f"{checkpoint.phase.value}; expected_artifacts={expected}; "
            "artifact_bundle_validated=false. Preserve the current approval and "
            "read-only boundaries; do not reconstruct workflow state from memory.",
        )

    def _handle_stop(self, event: StopEvent) -> HookOutcome:
        context = self._state_store.load_context(event.session_id, event.turn_id or "")
        self._set_state(context=context)
        if not context.active:
            return NoOpOutcome()

        if event.stop_hook_active:
            self._persist_pending_or_cancel(event, context)
            self._state_store.clear_context(event.session_id, event.turn_id or "")
            self.state.context = _inactive_context()
            return NoOpOutcome()

        violations = final_response_violations(context, event.last_assistant_message)
        if violations:
            return BlockingOutcome(
                reason=(
                    "The AI Software Architect response did not pass its deterministic "
                    "user-facing contract. Return a complete standalone replacement response "
                    "that preserves all already-valid content; do not return an addendum or "
                    "only the missing sentence. Correct these issues: "
                    + "; ".join(violations)
                    + "."
                )
            )
        self._state_store.clear_context(event.session_id, event.turn_id or "")
        self._persist_pending_or_cancel(event, context)
        self.state.context = _inactive_context()
        return NoOpOutcome()

    def _persist_pending_or_cancel(
        self,
        event: StopEvent,
        context: CodexTurnContext,
    ) -> None:
        pending = pending_continuation(event.last_assistant_message, context)
        if pending is None:
            self._state_store.cancel_continuation(event.session_id)
            self.state.continuation = None
            return
        self._state_store.open_continuation(event.session_id, pending)
        checkpoint = WorkflowCheckpoint(
            phase=(
                CheckpointPhase.AWAIT_DECISION
                if pending.interaction == PendingInteraction.DECISION
                else CheckpointPhase.CLARIFY
            )
        )
        self._state_store.save_checkpoint(event.session_id, checkpoint)
        self.state.continuation = pending
        self.state.checkpoint = checkpoint


__all__ = [
    "BlockingOutcome",
    "CodexResponseAdapter",
    "ContextInjectionOutcome",
    "DenialOutcome",
    "FileStateStore",
    "HookEvent",
    "HookEventValue",
    "HookLifecycle",
    "HookOutcome",
    "InMemoryStateStore",
    "LifecycleState",
    "NoOpOutcome",
    "PostCompactEvent",
    "PostToolUseEvent",
    "PreToolUseEvent",
    "StateStore",
    "StopEvent",
    "StopOutcome",
    "UserPromptSubmitEvent",
]
