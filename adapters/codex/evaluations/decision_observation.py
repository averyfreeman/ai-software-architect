# SPDX-FileCopyrightText: 2026 Leonardo Muffato (AUTOSOFT Engineering - www.autosoft-engineering.de)
# SPDX-License-Identifier: MIT

"""Extract bounded comparison outcomes without retaining free-form response text."""

from __future__ import annotations

import hashlib
import re

from adapters.codex.evaluations.models import DecisionObservation

_OPTION_CELL = re.compile(
	r"^\[(?P<category>GoF|Architecture|Presentation|Dependency|Data|Integration|"
	r"Resilience|Modernization|No pattern)\]\s+"
	r"(?:\[(?P<linked>[^\]]+)\]\((?P<link>[^)]+)\)|(?P<plain>.+))$"
)
_HEADINGS = {
	"alternatives": ("## Alternatives", "## Alternativen", "## Alternativas"),
	"recommendation": ("## Recommendation", "## Empfehlung", "## Recomendação"),
	"supporting": ("## Supporting patterns", "## Unterstützende Patterns", "## Padrões de apoio", "## Patrones complementarios"),
}


def _normalized_assumption(value: str) -> str:
    """Normalize presentation-only differences before deriving a private fingerprint."""

    return " ".join(value.casefold().split())


def _section(text: str, headings: tuple[str, ...], next_headings: tuple[str, ...]) -> str:
    starts = [text.find(heading) for heading in headings if text.find(heading) >= 0]
    if not starts:
        return ""
    start = min(starts)
    ends = [
        text.find(heading, start + 1)
        for heading in next_headings
        if text.find(heading, start + 1) >= 0
    ]
    end = min(ends) if ends else len(text)
    return text[start:end]


def _selected_row(response: str) -> tuple[str, str, str]:
    alternatives = _section(response, _HEADINGS["alternatives"], _HEADINGS["recommendation"])
    recommendation = _section(response, _HEADINGS["recommendation"], _HEADINGS["supporting"])
    rows: list[tuple[str, str, str, str]] = []
    for line in alternatives.splitlines():
        cells = [cell.strip() for cell in line.strip().strip("|").split("|")]
        if len(cells) != 6 or not re.fullmatch(r"(?:100|[1-9]?[0-9])/100", cells[1]):
            continue
        match = _OPTION_CELL.fullmatch(cells[0])
        if match is None:
            continue
        category = match.group("category")
        name = match.group("linked") or match.group("plain") or ""
        rows.append((cells[0], category, name, cells[5]))
    for option_cell, category, name, assumption in rows:
        if option_cell in recommendation:
            return category, name, assumption
    raise ValueError("validated comparison did not repeat a selected option cell")


def extract_decision_observation(response: str) -> DecisionObservation:
    """Extract one validated selection and hash, rather than store, its assumption."""

    selected_category, selected_name, selected_assumption = _selected_row(response)
    assumption = _normalized_assumption(selected_assumption)
    if not assumption:
        raise ValueError("selected alternative has no material assumption")

    # Catalog-backed names are public product metadata. A free-form no-pattern label
    # may contain project details, so reduce it to the stable category name.
    selected_name = "No pattern" if selected_category == "No pattern" else selected_name
    return DecisionObservation(
        selected_category=selected_category,
        selected_name=selected_name,
        material_assumption_sha256=hashlib.sha256(assumption.encode("utf-8")).hexdigest(),
        material_assumption_word_count=len(re.findall(r"\b\w+\b", assumption)),
        visible_response_word_count=len(re.findall(r"\b[\w-]+\b", response)),
    )
