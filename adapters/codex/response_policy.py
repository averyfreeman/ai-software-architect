# SPDX-FileCopyrightText: 2026 Leonardo Muffato (AUTOSOFT Engineering - www.autosoft-engineering.de)
# SPDX-License-Identifier: MIT

"""Pure visible-response validation and continuation policy for Codex hooks."""

from __future__ import annotations

import re
from dataclasses import dataclass
from pathlib import Path

from ai_architect_schemas import ComparedArchitectureOption
from pydantic import ValidationError

try:
    from adapters.codex.activation_policy import (
        CANONICAL_REFERENCE_BASE,
        CodexTurnContext,
        CodexTurnRoute,
    )
    from adapters.codex.continuation import (
        ApprovalTransition,
        PendingInteraction,
        SessionContinuation,
        WorkflowPhase,
    )
    from adapters.codex.reference_catalog import REFERENCE_CATALOG, ReferenceSpec
    from adapters.codex.response_locales import (
        ComparisonSection,
        comparison_contract_guidance,
        comparison_locale,
        contains_comparison_section,
        locale_containing_section,
        matching_comparison_locale,
    )
except ModuleNotFoundError as exc:
    if exc.name != "adapters":
        raise
    from activation_policy import (  # type: ignore[import-not-found, no-redef]
        CANONICAL_REFERENCE_BASE,
        CodexTurnContext,
        CodexTurnRoute,
    )
    from continuation import (  # type: ignore[import-not-found, no-redef]
        ApprovalTransition,
        PendingInteraction,
        SessionContinuation,
        WorkflowPhase,
    )
    from reference_catalog import (  # type: ignore[import-not-found, no-redef]
        REFERENCE_CATALOG,
        ReferenceSpec,
    )
    from response_locales import (  # type: ignore[import-not-found, no-redef]
        ComparisonSection,
        comparison_contract_guidance,
        comparison_locale,
        contains_comparison_section,
        locale_containing_section,
        matching_comparison_locale,
    )

REQUIRED_COMPARISON_SECTIONS = comparison_locale("en").headings
HIDDEN_HTML_COMMENT_PATTERN = re.compile(r"<!--.*?-->", flags=re.DOTALL)


@dataclass(frozen=True)
class ParsedOptionComparison:
    """The exact rendering fields the Stop hook can verify without inference."""

    decision_scope_and_criteria: str
    evidence_and_assumptions: str
    alternatives: tuple[ComparedArchitectureOption, ...]
    recommended_option_id: str
    recommendation: str
    supporting_patterns: str
    user_decision_prompt: str


def pending_continuation(
    message: str,
    context: CodexTurnContext,
) -> SessionContinuation | None:
    visible = message.rstrip()
    if contains_comparison_section(visible, ComparisonSection.USER_DECISION):
        return SessionContinuation(
            context=context,
            interaction=PendingInteraction.DECISION,
            phase=WorkflowPhase.APPROVE,
            approval_transition=ApprovalTransition.RECORD_AND_HANDOFF,
        )
    if visible.endswith("?"):
        return SessionContinuation(
            context=context,
            interaction=PendingInteraction.CLARIFICATION,
            phase=WorkflowPhase.CLARIFY,
            approval_transition=ApprovalTransition.RESUME_DESIGN,
        )
    return None


def continuation_instruction(continuation: SessionContinuation) -> str:
    if continuation.interaction == PendingInteraction.DECISION:
        return (
            "The preceding response requested a decision. Interpret the user's reply "
            "host-natively and in any language. If the user approves, do not merely "
            "acknowledge approval: transition to `record_and_handoff`, safely create "
            "and validate the approved ADR, architecture contract, context, and coding "
            "handoff when this is a project-bound material decision. Architecture "
            "artifact writes under `.ai-architect/` are authorized by approval unless "
            "the original request explicitly prohibited creating or modifying files. "
            "Approval never authorizes application-code changes. If the original turn "
            "was read-only or projectless, preserve that restriction and plainly "
            "explain why artifacts were not persisted. If the user revises or rejects "
            "the proposal, return to design and persist nothing."
        )
    return (
        "The preceding response requested clarification. Interpret this reply as the "
        "answer, retain the clarified constraints, and resume the smallest sufficient "
        "architecture workflow without requiring another skill invocation."
    )


def _section_text(message: str, heading: str, next_heading: str | None) -> str:
    start = message.find(heading)
    if start < 0:
        return ""
    start += len(heading)
    end = message.find(next_heading, start) if next_heading else len(message)
    return message[start:end].strip() if end >= 0 else ""


def _reference_spec_for_name(name: str) -> ReferenceSpec | None:
    return REFERENCE_CATALOG.named(name)


def _validate_canonical_reference(
    *,
    category: str,
    name: str,
    link: str | None,
) -> None:
    if category == "No pattern":
        return
    if link is None or not link.startswith(CANONICAL_REFERENCE_BASE):
        raise ValueError(f"{name} must link to the canonical public reference")
    expected = _reference_spec_for_name(name)
    if expected is None:
        return
    if category != expected.category:
        raise ValueError(f"{name} must use the {expected.category} category")
    if link != CANONICAL_REFERENCE_BASE + expected.filename:
        raise ValueError(f"{name} must link to {expected.filename}")


def _validate_supporting_patterns(text: str) -> None:
    linked_name = re.compile(r"\[([^\]]+)\]\(([^)]+)\)")
    for line in text.splitlines():
        if not line.lstrip().startswith(("-", "*")):
            continue
        linked_specs = tuple(
            (REFERENCE_CATALOG.named(name), link)
            for name, link in linked_name.findall(line)
        )
        for spec, link in linked_specs:
            if spec is None:
                continue
            if (
                f"[{spec.category}]" not in line
                or link != CANONICAL_REFERENCE_BASE + spec.filename
            ):
                raise ValueError(
                    f"the first supporting-pattern mention of {spec.name} must use "
                    f"[{spec.category}] and its canonical public reference"
                )
        for spec in REFERENCE_CATALOG.explicitly_named(line):
            expected_link = CANONICAL_REFERENCE_BASE + spec.filename
            if f"[{spec.category}]" not in line or expected_link not in line:
                raise ValueError(
                    f"the first supporting-pattern mention of {spec.name} must use "
                    f"[{spec.category}] and its canonical public reference"
                )


def parse_option_comparison_markdown(message: str) -> ParsedOptionComparison:
    """Parse only the user-facing fields that are deterministically represented."""

    if HIDDEN_HTML_COMMENT_PATTERN.search(message):
        raise ValueError("internal control markers and HTML comments must not be rendered")
    locale = matching_comparison_locale(message)

    alternatives_text = _section_text(
        message,
        locale.heading(ComparisonSection.ALTERNATIVES),
        locale.heading(ComparisonSection.RECOMMENDATION),
    )
    header_found = any(
        tuple(cell.strip() for cell in line.strip().strip("|").split("|"))
        == locale.table_headers
        for line in alternatives_text.splitlines()
    )
    if not header_found:
        raise ValueError("comparison must use the selected locale's exact table header")
    rows: list[ComparedArchitectureOption] = []
    option_names: dict[str, str] = {}
    option_cells: dict[str, str] = {}
    option_pattern = re.compile(
        r"^\[(?P<category>GoF|Architecture|Presentation|Dependency|Data|Integration|"
        r"Resilience|Modernization|No pattern)\]\s+"
        r"(?:\[(?P<linked_name>[^\]]+)\]\((?P<link>[^)]+)\)|(?P<plain_name>.+))$"
    )
    for line in alternatives_text.splitlines():
        cells = [cell.strip() for cell in line.strip().strip("|").split("|")]
        if len(cells) != 6:
            continue
        score_match = re.fullmatch(
            r"(?P<plain_score>(?:100|[1-9]?[0-9])/100)|"
            r"(?P<emphasis>\*\*|__)"
            r"(?P<emphasized_score>(?:100|[1-9]?[0-9])/100)"
            r"(?P=emphasis)",
            cells[1],
        )
        if score_match is None:
            continue
        score_text = score_match.group("plain_score") or score_match.group(
            "emphasized_score"
        )
        if score_text is None:
            continue
        matched = option_pattern.fullmatch(cells[0])
        if matched is None:
            continue
        category = matched.group("category")
        name = matched.group("linked_name") or matched.group("plain_name")
        option_id = f"OPT-{len(rows) + 1:03d}"
        if category == "No pattern" and matched.group("link") is not None:
            raise ValueError(
                "a No pattern alternative must use plain option text without a link"
            )
        link = None if category == "No pattern" else matched.group("link")
        _validate_canonical_reference(category=category, name=name, link=link)
        option = ComparedArchitectureOption.model_validate(
            {
                "id": option_id,
                "category": category,
                "name": name,
                "canonical_reference": link,
                "fit_score": int(score_text.removesuffix("/100")),
                "fit_rationale": cells[2],
                "main_benefit": cells[3],
                "main_liability": cells[4],
                "material_assumption": cells[5],
            }
        )
        rows.append(option)
        option_names[option.name.casefold()] = option.id
        option_cells[option.id] = cells[0]
    if not 2 <= len(rows) <= 5:
        raise ValueError("comparison must contain two to five valid alternative rows")

    recommendation = _section_text(
        message,
        locale.heading(ComparisonSection.RECOMMENDATION),
        locale.heading(ComparisonSection.SUPPORTING_PATTERNS),
    )
    lowered_recommendation = recommendation.casefold()
    mentioned = sorted(
        (lowered_recommendation.find(name), option_id)
        for name, option_id in option_names.items()
        if name in lowered_recommendation
    )
    if not mentioned:
        raise ValueError("recommendation must name a compared alternative")
    recommended_option_id = mentioned[0][1]
    if option_cells[recommended_option_id] not in recommendation:
        raise ValueError(
            "recommendation must repeat the selected Option cell exactly, including "
            "its category and canonical link"
        )

    decision_scope = _section_text(
        message,
        locale.heading(ComparisonSection.DECISION_SCOPE),
        locale.heading(ComparisonSection.EVIDENCE),
    )
    if "ordinal" not in decision_scope.casefold():
        raise ValueError("decision criteria must describe Fit as an ordinal score")

    supporting_patterns = _section_text(
        message,
        locale.heading(ComparisonSection.SUPPORTING_PATTERNS),
        locale.heading(ComparisonSection.USER_DECISION),
    )
    _validate_supporting_patterns(supporting_patterns)

    decision_prompt = _section_text(
        message,
        locale.heading(ComparisonSection.USER_DECISION),
        None,
    )
    visible_decision_prompt = re.sub(
        r"<!--.*?-->",
        "",
        decision_prompt,
        flags=re.DOTALL,
    ).strip()
    if not visible_decision_prompt:
        raise ValueError("user decision prompt must contain visible guidance")

    parsed = ParsedOptionComparison(
        decision_scope_and_criteria=decision_scope,
        evidence_and_assumptions=_section_text(
            message,
            locale.heading(ComparisonSection.EVIDENCE),
            locale.heading(ComparisonSection.ALTERNATIVES),
        ),
        alternatives=tuple(rows),
        recommended_option_id=recommended_option_id,
        recommendation=recommendation,
        supporting_patterns=supporting_patterns,
        user_decision_prompt=visible_decision_prompt,
    )
    if not all(
        (
            parsed.decision_scope_and_criteria,
            parsed.evidence_and_assumptions,
            parsed.recommendation,
            parsed.supporting_patterns,
        )
    ):
        raise ValueError("comparison sections must contain visible content")
    return parsed


def _option_comparison_violations(message: str) -> list[str]:
    try:
        parse_option_comparison_markdown(message)
    except (ValidationError, ValueError) as exc:
        return [
            "render one complete replacement in the user's language using exactly "
            "one localized comparison contract."
            + comparison_contract_guidance()
            + ". Provide two to five genuine rows (normally "
            "three to five). Allowed category labels are `GoF`, `Architecture`, "
            "`Presentation`, `Dependency`, `Data`, `Integration`, `Resilience`, "
            "`Modernization`, and `No pattern`. Example Option cells: `[No pattern] "
            "Keep the script simple`; `[GoF] "
            f"[Strategy]({CANONICAL_REFERENCE_BASE}gof-strategy.md)`. Each named "
            "option links its canonical public reference, and each Fit is ordinal "
            "`NN/100`; Decision scope and criteria must explicitly call Fit ordinal. "
            "The Recommendation must name one table option. Supporting patterns must "
            "remain separate, and the first mention of every named supporting pattern "
            "must include its category and canonical public link. Your decision must "
            "contain visible guidance offering approval, revision, or more "
            "information. Do not include internal control markers or HTML comments. "
            "Validation "
            f"detail: {exc}"
        ]
    return []


def _architecture_workflow_violations(message: str) -> list[str]:
    if HIDDEN_HTML_COMMENT_PATTERN.search(message):
        return [
            "remove internal control markers or HTML comments and return only user-facing content"
        ]
    if contains_comparison_section(message, ComparisonSection.ALTERNATIVES):
        return _option_comparison_violations(message)
    if contains_comparison_section(message, ComparisonSection.USER_DECISION):
        locale = locale_containing_section(message, ComparisonSection.USER_DECISION)
        user_decision_heading = locale.heading(ComparisonSection.USER_DECISION)
        visible_guidance = re.sub(
            r"<!--.*?-->",
            "",
            _section_text(message, user_decision_heading, None),
            flags=re.DOTALL,
        ).strip()
        if not visible_guidance:
            return ["place visible decision guidance under `## Your decision`"]
        if re.search(r"(?m)^#{1,6}\s+", visible_guidance):
            return [
                "put all recommendation headings and content before `## Your "
                "decision`; keep that final section limited to visible decision "
                "guidance"
            ]
    return []


def final_response_violations(
    context: CodexTurnContext,
    message: str,
) -> list[str]:
    if context.route == CodexTurnRoute.ARCHITECTURE_WORKFLOW:
        violations = _architecture_workflow_violations(message)
        missing_links = [
            f"{CANONICAL_REFERENCE_BASE}{Path(path).name}"
            for path in context.reference_paths
            if f"{CANONICAL_REFERENCE_BASE}{Path(path).name}" not in message
        ]
        if missing_links:
            violations.append(
                "include the canonical public link for every explicitly routed "
                "architecture reference: " + ", ".join(missing_links)
            )
        return violations
    return []
