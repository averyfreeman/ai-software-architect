package architect

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

func RenderArchitectureContract(contract Contract) (string, error) {
	normalizeContract(&contract)
	data, err := yaml.Marshal(contract)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

type ComparisonLocale struct {
	Code          string
	Headings      [6]string
	TableHeaders  [6]string
	FitDisclosure string
	ChoosePrefix  string
	NoSupporting  string
}

var comparisonLocales = []ComparisonLocale{
	{Code: "en", Headings: [6]string{"## Decision scope and criteria", "## Evidence and assumptions", "## Alternatives", "## Recommendation", "## Supporting patterns", "## Your decision"}, TableHeaders: [6]string{"Option", "Fit", "Rationale", "Main benefit", "Main liability", "Material assumption"}, FitDisclosure: "Fit is an ordinal `NN/100` score for this decision, not a probability or measured percentage.", ChoosePrefix: "Choose", NoSupporting: "No supporting patterns are required."},
	{Code: "de", Headings: [6]string{"## Entscheidungsumfang und Kriterien", "## Evidenz und Annahmen", "## Alternativen", "## Empfehlung", "## Unterstützende Patterns", "## Deine Entscheidung"}, TableHeaders: [6]string{"Option", "Fit", "Begründung", "Hauptvorteil", "Hauptnachteil", "Wesentliche Annahme"}, FitDisclosure: "Fit ist ein ordinaler `NN/100`-Wert für diese Entscheidung, keine Wahrscheinlichkeit und kein gemessener Prozentsatz.", ChoosePrefix: "Wähle", NoSupporting: "Es sind keine unterstützenden Patterns erforderlich."},
	{Code: "pt-BR", Headings: [6]string{"## Escopo da decisão e critérios", "## Evidências e premissas", "## Alternativas", "## Recomendação", "## Padrões de apoio", "## Sua decisão"}, TableHeaders: [6]string{"Opção", "Adequação", "Justificativa", "Principal benefício", "Principal desvantagem", "Premissa relevante"}, FitDisclosure: "A adequação é uma pontuação ordinal `NN/100` para esta decisão, não uma probabilidade nem uma porcentagem medida.", ChoosePrefix: "Escolha", NoSupporting: "Nenhum padrão de apoio é necessário."},
	{Code: "es", Headings: [6]string{"## Alcance y criterios de la decisión", "## Evidencia y supuestos", "## Alternativas", "## Recomendación", "## Patrones complementarios", "## Tu decisión"}, TableHeaders: [6]string{"Opción", "Adecuación", "Justificación", "Beneficio principal", "Desventaja principal", "Supuesto relevante"}, FitDisclosure: "La adecuación es una puntuación ordinal `NN/100` para esta decisión, no una probabilidad ni un porcentaje medido.", ChoosePrefix: "Elige", NoSupporting: "No se requieren patrones complementarios."},
}

func ComparisonLocaleFor(code string) (ComparisonLocale, error) {
	for _, locale := range comparisonLocales {
		if strings.EqualFold(locale.Code, code) {
			return locale, nil
		}
	}
	return ComparisonLocale{}, fmt.Errorf("unsupported comparison response locale: %s", code)
}

func MatchingComparisonLocale(message string) (ComparisonLocale, error) {
	matches := []ComparisonLocale{}
	for _, locale := range comparisonLocales {
		positions := make([]int, 6)
		ok := true
		for index, heading := range locale.Headings {
			positions[index] = strings.Index(message, heading)
			if positions[index] < 0 || (index > 0 && positions[index] < positions[index-1]) {
				ok = false
			}
		}
		if ok {
			matches = append(matches, locale)
		}
	}
	if len(matches) != 1 {
		return ComparisonLocale{}, fmt.Errorf("comparison must use exactly one complete localized heading set")
	}
	return matches[0], nil
}

func RenderOptionComparison(comparison ArchitectureOptionComparison, localeCode string) (string, error) {
	locale, err := ComparisonLocaleFor(localeCode)
	if err != nil {
		return "", err
	}
	byID := map[string]ComparedArchitectureOption{}
	for _, option := range comparison.Alternatives {
		byID[option.ID] = option
	}
	selected, ok := byID[comparison.RecommendedOptionID]
	if !ok {
		return "", fmt.Errorf("recommended option is not present")
	}
	optionCell := func(option ComparedArchitectureOption) string {
		if option.CanonicalReference == nil {
			return fmt.Sprintf("[%s] %s", option.Category, option.Name)
		}
		return fmt.Sprintf("[%s] [%s](%s)", option.Category, option.Name, *option.CanonicalReference)
	}
	lines := []string{locale.Headings[0], "", comparison.DecisionScope, "", locale.FitDisclosure, ""}
	for _, criterion := range comparison.ScoringCriteria {
		lines = append(lines, "- "+criterion)
	}
	lines = append(lines, "", locale.Headings[1], "")
	for _, claim := range comparison.EvidenceAndAssumptions {
		lines = append(lines, fmt.Sprintf("- **%s:** %s", claim.Kind, claim.Claim))
	}
	lines = append(lines, "", locale.Headings[2], "", "| "+strings.Join(locale.TableHeaders[:], " | ")+" |", "| --- | ---: | --- | --- | --- | --- |")
	for _, option := range comparison.Alternatives {
		lines = append(lines, fmt.Sprintf("| %s | %d/100 | %s | %s | %s | %s |", optionCell(option), option.FitScore, option.FitRationale, option.MainBenefit, option.MainLiability, option.MaterialAssumption))
	}
	lines = append(lines, "", locale.Headings[3], "", locale.ChoosePrefix+" **"+optionCell(selected)+"**.", "", comparison.RecommendationRationale, "", locale.Headings[4], "")
	if len(comparison.SupportingPatterns) == 0 {
		lines = append(lines, locale.NoSupporting)
	} else {
		for _, pattern := range comparison.SupportingPatterns {
			lines = append(lines, fmt.Sprintf("- [%s] [%s](%s) — %s", pattern.Category, pattern.Name, pattern.CanonicalReference, pattern.Role))
		}
	}
	lines = append(lines, "", locale.Headings[5], "", comparison.UserDecisionPrompt, "")
	return strings.Join(lines, "\n"), nil
}
