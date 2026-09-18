package codex

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/averyfreeman/adr-repo-governance/internal/architect"
)

type comparisonLocale struct {
	Code         string
	Headings     [6]string
	TableHeaders [6]string
}

var comparisonLocales = []comparisonLocale{
	{Code: "en", Headings: [6]string{"## Decision scope and criteria", "## Evidence and assumptions", "## Alternatives", "## Recommendation", "## Supporting patterns", "## Your decision"}, TableHeaders: [6]string{"Option", "Fit", "Rationale", "Main benefit", "Main liability", "Material assumption"}},
	{Code: "de", Headings: [6]string{"## Entscheidungsumfang und Kriterien", "## Evidenz und Annahmen", "## Alternativen", "## Empfehlung", "## Unterstützende Patterns", "## Deine Entscheidung"}, TableHeaders: [6]string{"Option", "Fit", "Begründung", "Hauptvorteil", "Hauptnachteil", "Wesentliche Annahme"}},
	{Code: "pt-BR", Headings: [6]string{"## Escopo da decisão e critérios", "## Evidências e premissas", "## Alternativas", "## Recomendação", "## Padrões de apoio", "## Sua decisão"}, TableHeaders: [6]string{"Opção", "Adequação", "Justificativa", "Principal benefício", "Principal desvantagem", "Premissa relevante"}},
	{Code: "es", Headings: [6]string{"## Alcance y criterios de la decisión", "## Evidencia y supuestos", "## Alternativas", "## Recomendación", "## Patrones complementarios", "## Tu decisión"}, TableHeaders: [6]string{"Opción", "Adecuación", "Justificación", "Beneficio principal", "Desventaja principal", "Supuesto relevante"}},
}

func matchingComparisonLocale(message string) (comparisonLocale, error) {
	matches := []comparisonLocale{}
	for _, locale := range comparisonLocales {
		previous := -1
		ok := true
		for _, heading := range locale.Headings {
			position := strings.Index(message, heading)
			if position < 0 || position < previous {
				ok = false
				break
			}
			previous = position
		}
		if ok {
			matches = append(matches, locale)
		}
	}
	if len(matches) != 1 {
		return comparisonLocale{}, fmt.Errorf("comparison must use exactly one complete localized heading set")
	}
	return matches[0], nil
}

func localeContainingDecisionHeading(message string) (comparisonLocale, error) {
	matches := []comparisonLocale{}
	for _, locale := range comparisonLocales {
		if strings.Contains(message, locale.Headings[5]) {
			matches = append(matches, locale)
		}
	}
	if len(matches) != 1 {
		return comparisonLocale{}, fmt.Errorf("response must use one unambiguous localized decision heading")
	}
	return matches[0], nil
}

type parsedOptionComparison struct {
	DecisionScopeAndCriteria string
	EvidenceAndAssumptions   string
	Alternatives             []architect.ComparedArchitectureOption
	RecommendedOptionID      string
	Recommendation           string
	SupportingPatterns       string
	UserDecisionPrompt       string
}

var optionCellPattern = regexp.MustCompile("^\\[(GoF|Architecture|Presentation|Dependency|Data|Integration|Resilience|Modernization|No pattern)\\]\\s+(?:\\[([^\\]]+)\\]\\(([^)]+)\\)|(.+))$")

func splitTableRow(line string) []string {
	value := strings.TrimSpace(line)
	if strings.HasPrefix(value, "|") {
		value = value[1:]
	}
	if strings.HasSuffix(value, "|") {
		value = value[:len(value)-1]
	}
	parts := strings.Split(value, "|")
	for index := range parts {
		parts[index] = strings.TrimSpace(parts[index])
	}
	return parts
}

func fitScore(value string) (int, bool) {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, "**") && strings.HasSuffix(value, "**") {
		value = strings.TrimSuffix(strings.TrimPrefix(value, "**"), "**")
	}
	if strings.HasPrefix(value, "__") && strings.HasSuffix(value, "__") {
		value = strings.TrimSuffix(strings.TrimPrefix(value, "__"), "__")
	}
	if !regexp.MustCompile("^(?:100|[1-9]?[0-9])/100$").MatchString(value) {
		return 0, false
	}
	number, err := strconv.Atoi(strings.TrimSuffix(value, "/100"))
	return number, err == nil && number >= 0 && number <= 100
}

func parseOptionComparisonMarkdown(message string) (parsedOptionComparison, error) {
	if hiddenHTMLCommentPattern.MatchString(message) {
		return parsedOptionComparison{}, fmt.Errorf("internal control markers and HTML comments must not be rendered")
	}
	locale, err := matchingComparisonLocale(message)
	if err != nil {
		return parsedOptionComparison{}, err
	}
	alternativesText := sectionText(message, locale.Headings[2], locale.Headings[3])
	headerFound := false
	for _, line := range strings.Split(alternativesText, "\n") {
		if equalStrings(splitTableRow(line), locale.TableHeaders[:]) {
			headerFound = true
			break
		}
	}
	if !headerFound {
		return parsedOptionComparison{}, fmt.Errorf("comparison must use the selected locale's exact table header")
	}
	rows := []architect.ComparedArchitectureOption{}
	optionNames := map[string]string{}
	optionCells := map[string]string{}
	for _, line := range strings.Split(alternativesText, "\n") {
		cells := splitTableRow(line)
		if len(cells) != 6 {
			continue
		}
		score, ok := fitScore(cells[1])
		if !ok {
			continue
		}
		match := optionCellPattern.FindStringSubmatch(cells[0])
		if len(match) == 0 {
			continue
		}
		category := strings.TrimSpace(match[1])
		name := strings.TrimSpace(match[2])
		link := strings.TrimSpace(match[3])
		if name == "" {
			name = strings.TrimSpace(match[4])
		}
		if category == "No pattern" && link != "" {
			return parsedOptionComparison{}, fmt.Errorf("a No pattern alternative must use plain option text without a link")
		}
		if err := validateCanonicalReference(category, name, link); err != nil {
			return parsedOptionComparison{}, err
		}
		if name == "" || cells[2] == "" || cells[3] == "" || cells[4] == "" || cells[5] == "" {
			return parsedOptionComparison{}, fmt.Errorf("comparison rows must contain visible content")
		}
		optionID := fmt.Sprintf("OPT-%03d", len(rows)+1)
		var canonical *string
		if link != "" {
			canonical = &link
		}
		rows = append(rows, architect.ComparedArchitectureOption{
			ID: optionID, Category: category, Name: name, CanonicalReference: canonical,
			FitScore: score, FitRationale: cells[2], MainBenefit: cells[3],
			MainLiability: cells[4], MaterialAssumption: cells[5],
		})
		optionNames[strings.ToLower(name)] = optionID
		optionCells[optionID] = cells[0]
	}
	if len(rows) < 2 || len(rows) > 5 {
		return parsedOptionComparison{}, fmt.Errorf("comparison must contain two to five valid alternative rows")
	}
	recommendation := sectionText(message, locale.Headings[3], locale.Headings[4])
	loweredRecommendation := strings.ToLower(recommendation)
	mentioned := []struct {
		position int
		id       string
	}{}
	for name, id := range optionNames {
		if position := strings.Index(loweredRecommendation, name); position >= 0 {
			mentioned = append(mentioned, struct {
				position int
				id       string
			}{position: position, id: id})
		}
	}
	if len(mentioned) == 0 {
		return parsedOptionComparison{}, fmt.Errorf("recommendation must name a compared alternative")
	}
	sort.Slice(mentioned, func(i, j int) bool { return mentioned[i].position < mentioned[j].position })
	recommendedID := mentioned[0].id
	if !strings.Contains(recommendation, optionCells[recommendedID]) {
		return parsedOptionComparison{}, fmt.Errorf("recommendation must repeat the selected Option cell exactly, including its category and canonical link")
	}
	decisionScope := sectionText(message, locale.Headings[0], locale.Headings[1])
	if !strings.Contains(strings.ToLower(decisionScope), "ordinal") {
		return parsedOptionComparison{}, fmt.Errorf("decision criteria must describe Fit as an ordinal score")
	}
	supporting := sectionText(message, locale.Headings[4], locale.Headings[5])
	if err := validateSupportingPatterns(supporting); err != nil {
		return parsedOptionComparison{}, err
	}
	decisionPrompt := strings.TrimSpace(hiddenHTMLCommentPattern.ReplaceAllString(sectionText(message, locale.Headings[5], ""), ""))
	if decisionPrompt == "" {
		return parsedOptionComparison{}, fmt.Errorf("user decision prompt must contain visible guidance")
	}
	if regexp.MustCompile("(?m)^#{1,6}\\s+").MatchString(decisionPrompt) {
		return parsedOptionComparison{}, fmt.Errorf("user decision section must contain visible guidance only")
	}
	evidence := sectionText(message, locale.Headings[1], locale.Headings[2])
	if evidence == "" || recommendation == "" || supporting == "" {
		return parsedOptionComparison{}, fmt.Errorf("comparison sections must contain visible content")
	}
	return parsedOptionComparison{
		DecisionScopeAndCriteria: decisionScope,
		EvidenceAndAssumptions:   evidence,
		Alternatives:             rows,
		RecommendedOptionID:      recommendedID,
		Recommendation:           recommendation,
		SupportingPatterns:       supporting,
		UserDecisionPrompt:       decisionPrompt,
	}, nil
}

func equalStrings(actual, expected []string) bool {
	if len(actual) != len(expected) {
		return false
	}
	for index := range actual {
		if actual[index] != expected[index] {
			return false
		}
	}
	return true
}

func validateSupportingPatterns(text string) error {
	linkPattern := regexp.MustCompile("\\[([^\\]]+)\\]\\(([^)]+)\\)")
	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "-") && !strings.HasPrefix(trimmed, "*") {
			continue
		}
		for _, match := range linkPattern.FindAllStringSubmatch(line, -1) {
			spec := referenceSpec(match[1])
			if spec == nil {
				continue
			}
			if !strings.Contains(line, "["+spec.Category+"]") || match[2] != CanonicalReferenceBase+spec.Filename {
				return fmt.Errorf("the first supporting-pattern mention of %s must use [%s] and its canonical public reference", spec.Name, spec.Category)
			}
		}
		for _, spec := range referenceCatalog {
			if !strings.Contains(strings.ToLower(line), strings.ToLower(spec.Name)) {
				continue
			}
			if !strings.Contains(line, "["+spec.Category+"]") || !strings.Contains(line, CanonicalReferenceBase+spec.Filename) {
				return fmt.Errorf("the first supporting-pattern mention of %s must use [%s] and its canonical public reference", spec.Name, spec.Category)
			}
		}
	}
	return nil
}
