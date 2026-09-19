# SPDX-FileCopyrightText: 2026 Leonardo Muffo (AUTOSOFT Engineering - www.autosoft-engineering.de)
# SPDX-License-Identifier: MIT

from __future__ import annotations

from pathlib import Path

from ai_architect_schemas import ArchitectureContract

from adapters.codex.activation_policy import CodexTurnContext, CodexTurnRoute
from adapters.codex.continuation import (
    ApprovalTransition,
    CheckpointPhase,
    PendingInteraction,
    SessionContinuation,
    WorkflowCheckpoint,
    WorkflowPhase,
)
from adapters.codex.hook_lifecycle import (
    BlockingOutcome,
    CodexResponseAdapter,
    ContextInjectionOutcome,
    DenialOutcome,
    HookEvent,
    HookLifecycle,
    InMemoryStateStore,
    NoOpOutcome,
    PostCompactEvent,
    PostToolUseEvent,
    PreToolUseEvent,
    StopEvent,
    StopOutcome,
    UserPromptSubmitEvent,
)
from adapters.codex.renderers import render_architecture_contract
from adapters.codex.state_store import FileStateStore


class MutableClock:
    def __init__(self, value: float = 1_000.0) -> None:
        self.value = value

    def __call__(self) -> float:
        return self.value

    def advance(self, seconds: float) -> None:
        self.value += seconds


def _payload(event: str, **extra: object) -> dict[str, object]:
    return {
        "hook_event_name": event,
        "session_id": "session-1",
        "turn_id": "turn-1",
        **extra,
    }


def _active_submit(
    lifecycle: HookLifecycle,
    *,
    session_id: str = "session-1",
    turn_id: str = "turn-1",
) -> ContextInjectionOutcome:
    outcome = lifecycle.handle(
        UserPromptSubmitEvent(
            session_id=session_id,
            turn_id=turn_id,
            cwd=None,
            prompt="$ai-software-architect Review this project.",
        )
    )
    assert isinstance(outcome, ContextInjectionOutcome)
    return outcome


def _decision_message() -> str:
    return "## Your decision\nApprove, revise, or request more information."


def _artifact_patch(contract: str, adr: str) -> str:
    files = {
        ".ai-architect/project-context.md": "# Context\n",
        ".ai-architect/architecture-contract.yaml": contract,
        ".ai-architect/implementation-plan.md": "# Coding handoff\n",
        ".ai-architect/decisions/ADR-001-boundary.md": adr,
    }
    lines = ["*** Begin Patch"]
    for path, content in files.items():
        lines.append(f"*** Add File: {path}")
        lines.extend(f"+{line}" for line in content.rstrip("\n").splitlines())
    lines.append("*** End Patch")
    return "\n".join(lines) + "\n"


def _valid_bundle() -> tuple[str, str]:
    contract = render_architecture_contract(
        ArchitectureContract.model_validate(
            {
                "schema_version": "1.0.0",
                "revision": 1,
                "scope": "sample",
                "decision_ids": ["ADR-001"],
            }
        )
    )
    adr = """---
schema_version: 1.0.0
revision: 1
decision:
  id: ADR-001
  title: Choose a boundary
  status: accepted
  context: A boundary is required.
  drivers: [Testability]
  considered_option_ids: [OPT-001]
  selected_option_id: OPT-001
  decision: Use one explicit boundary.
  validation_criteria: [Boundary tests pass.]
---
# Decision
"""
    return contract, adr


def test_hook_event_normalization_covers_all_five_events() -> None:
    event_cases = (
        ("UserPromptSubmit", UserPromptSubmitEvent),
        ("PreToolUse", PreToolUseEvent),
        ("PostToolUse", PostToolUseEvent),
        ("PostCompact", PostCompactEvent),
        ("Stop", StopEvent),
    )
    for event_name, event_type in event_cases:
        payload = _payload(event_name)
        if event_name == "UserPromptSubmit":
            payload["prompt"] = "$ai-software-architect Review this project."
        if event_name in {"PreToolUse", "PostToolUse"}:
            payload.update({"tool_name": "bash", "tool_input": {"command": "git status"}})
        if event_name == "Stop":
            payload["last_assistant_message"] = "Done."
        assert isinstance(HookEvent.from_payload(payload), event_type)


def test_activation_inactive_and_missing_invocation_outcomes() -> None:
    store = InMemoryStateStore()
    lifecycle = HookLifecycle(store)

    active = _active_submit(lifecycle)
    assert "model-selected workflow" in active.additional_context

    inactive = lifecycle.handle(
        HookEvent.from_payload(
            _payload("UserPromptSubmit", prompt="Please review this code.")
        )
    )
    assert isinstance(inactive, NoOpOutcome)

    missing = lifecycle.handle(
        HookEvent.from_payload(
            _payload(
                "UserPromptSubmit",
                prompt="[@AI Software Architect](plugin://ai-software-architect@personal)",
            )
        )
    )
    assert isinstance(missing, BlockingOutcome)
    assert "without a request" in missing.reason


def test_clarification_continuation_is_bounded_and_single_use() -> None:
    store = InMemoryStateStore()
    lifecycle = HookLifecycle(store)
    _active_submit(lifecycle)

    lifecycle.handle(
        StopEvent(
            session_id="session-1",
            turn_id="turn-1",
            cwd=None,
            last_assistant_message="Which platform should this support?",
            stop_hook_active=False,
        )
    )
    continued = lifecycle.handle(
        UserPromptSubmitEvent(
            session_id="session-1",
            turn_id="turn-2",
            cwd=None,
            prompt="A desktop application.",
        )
    )
    assert isinstance(continued, ContextInjectionOutcome)
    assert "typed clarification continuation" in continued.additional_context

    consumed_again = lifecycle.handle(
        UserPromptSubmitEvent(
            session_id="session-1",
            turn_id="turn-3",
            cwd=None,
            prompt="Another answer.",
        )
    )
    assert isinstance(consumed_again, NoOpOutcome)


def test_decision_continuation_updates_checkpoint_and_competing_activation_cancels() -> None:
    store = InMemoryStateStore()
    lifecycle = HookLifecycle(store)
    _active_submit(lifecycle)
    lifecycle.handle(
        StopEvent(
            session_id="session-1",
            turn_id="turn-1",
            cwd=None,
            last_assistant_message=_decision_message(),
            stop_hook_active=False,
        )
    )
    continued = lifecycle.handle(
        UserPromptSubmitEvent(
            session_id="session-1",
            turn_id="turn-2",
            cwd=None,
            prompt="Approve it.",
        )
    )
    assert isinstance(continued, ContextInjectionOutcome)
    assert "typed decision continuation" in continued.additional_context
    assert lifecycle.state.checkpoint is not None
    assert lifecycle.state.checkpoint.phase == CheckpointPhase.DECISION_RESPONSE

    store.open_continuation(
        "session-1",
        SessionContinuation(
            context=CodexTurnContext(True, CodexTurnRoute.ARCHITECTURE_WORKFLOW),
            interaction=PendingInteraction.CLARIFICATION,
            phase=WorkflowPhase.CLARIFY,
            approval_transition=ApprovalTransition.RESUME_DESIGN,
        ),
    )
    competing = lifecycle.handle(
        UserPromptSubmitEvent(
            session_id="session-1",
            turn_id="turn-3",
            cwd=None,
            prompt="$other-skill Do something else.",
        )
    )
    assert isinstance(competing, NoOpOutcome)
    assert store.consume_continuation("session-1") is None


def test_expired_continuation_and_checkpoint_restoration_use_injected_clock() -> None:
    clock = MutableClock()
    store = InMemoryStateStore(clock=clock)
    lifecycle = HookLifecycle(store, clock=clock)
    _active_submit(lifecycle)
    lifecycle.handle(
        StopEvent(
            session_id="session-1",
            turn_id="turn-1",
            cwd=None,
            last_assistant_message="Which option?",
            stop_hook_active=False,
        )
    )
    clock.advance(3_601)
    expired = lifecycle.handle(
        UserPromptSubmitEvent(
            session_id="session-1",
            turn_id="turn-2",
            cwd=None,
            prompt="The first option.",
        )
    )
    assert isinstance(expired, NoOpOutcome)

    store.save_checkpoint(
        "session-1",
        WorkflowCheckpoint(
            phase=CheckpointPhase.AWAIT_DECISION,
            expected_artifacts=["adr", "contract", "context", "implementation-plan"],
        ),
    )
    restored = lifecycle.handle(PostCompactEvent("session-1", "turn-3", None))
    assert isinstance(restored, ContextInjectionOutcome)
    assert "phase=await_decision" in restored.additional_context


def test_tool_policy_allows_static_reads_and_denies_unsafe_surfaces() -> None:
    store = InMemoryStateStore()
    lifecycle = HookLifecycle(store)
    _active_submit(lifecycle)

    allowed = lifecycle.handle(
        PreToolUseEvent(
            "session-1",
            "turn-1",
            None,
            "bash",
            {"command": "git status"},
        )
    )
    assert isinstance(allowed, NoOpOutcome)

    denied_shell = lifecycle.handle(
        PreToolUseEvent(
            "session-1",
            "turn-1",
            None,
            "bash",
            {"command": "python -c 'print(1)'"},
        )
    )
    assert isinstance(denied_shell, DenialOutcome)

    denied_patch = lifecycle.handle(
        PreToolUseEvent(
            "session-1",
            "turn-1",
            "/workspace",
            "apply_patch",
            "*** Begin Patch\n*** Update File: app.py\n*** End Patch\n",
        )
    )
    assert isinstance(denied_patch, DenialOutcome)


def test_artifact_write_and_post_write_failure_are_fail_closed(tmp_path: Path) -> None:
    store = InMemoryStateStore()
    lifecycle = HookLifecycle(store)
    _active_submit(lifecycle)
    lifecycle.handle(
        StopEvent(
            "session-1",
            "turn-1",
            None,
            _decision_message(),
            False,
        )
    )
    lifecycle.handle(
        UserPromptSubmitEvent("session-1", "turn-2", str(tmp_path), "Approve it.")
    )
    contract, adr = _valid_bundle()
    patch = _artifact_patch(contract, adr)
    allowed = lifecycle.handle(
        PreToolUseEvent("session-1", "turn-2", str(tmp_path), "apply_patch", patch)
    )
    assert isinstance(allowed, NoOpOutcome)

    for path, content in {
        ".ai-architect/project-context.md": "# Context\n",
        ".ai-architect/architecture-contract.yaml": contract,
        ".ai-architect/implementation-plan.md": "# Coding handoff\n",
        ".ai-architect/decisions/ADR-001-boundary.md": adr,
    }.items():
        target = tmp_path / path
        target.parent.mkdir(parents=True, exist_ok=True)
        target.write_text(content, encoding="utf-8")
    verified = lifecycle.handle(
        PostToolUseEvent("session-1", "turn-2", str(tmp_path), "apply_patch", patch)
    )
    assert isinstance(verified, ContextInjectionOutcome)
    assert lifecycle.state.checkpoint is not None
    assert lifecycle.state.checkpoint.phase == CheckpointPhase.COMPLETE

    mismatch = lifecycle.handle(
        PostToolUseEvent(
            "session-1",
            "turn-2",
            str(tmp_path),
            "write",
            {"path": ".ai-architect/project-context.md", "content": "changed\n"},
        )
    )
    assert isinstance(mismatch, StopOutcome)
    assert "post-write verification failed" in mismatch.stop_reason


def test_file_and_memory_state_adapters_produce_equivalent_transitions(
    tmp_path: Path,
) -> None:
    clock = MutableClock()
    stores = (
        InMemoryStateStore(clock=clock),
        FileStateStore(tmp_path, clock=clock),
    )
    rendered: list[dict[str, object]] = []
    for store in stores:
        lifecycle = HookLifecycle(store, clock=clock)
        _active_submit(lifecycle)
        stopped = lifecycle.handle(
            StopEvent(
                "session-1",
                "turn-1",
                None,
                "Which platform?",
                False,
            )
        )
        rendered.append(CodexResponseAdapter.render(stopped))
        assert isinstance(stopped, NoOpOutcome)
        assert store.consume_continuation("session-1") is not None
        assert store.consume_continuation("session-1") is None
    assert rendered == [{}, {}]


def test_file_state_cleanup_prunes_stale_files_and_preserves_hashed_names(
    tmp_path: Path,
) -> None:
    clock = MutableClock()
    store = FileStateStore(tmp_path, clock=clock, max_state_age_seconds=10)
    store.save_context(
        "session-1",
        "turn-1",
        CodexTurnContext(True, CodexTurnRoute.ARCHITECTURE_WORKFLOW),
    )
    state_path = next((tmp_path / "control-plane").glob("turn-*.json"))
    assert state_path.name.startswith("turn-")
    clock.advance(11)
    store.cleanup()
    assert not state_path.exists()
