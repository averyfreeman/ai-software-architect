package architect

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"
)

const (
	CurrentSchemaVersion = "1.1.0"
	MaxErrors            = 100
	MaxNestingDepth      = 50

	MaxInlineSourceFiles       = 500
	MaxInlineSourceBytes       = 5_000_000
	MaxInlineSourceFileBytes   = 500_000
	MaxDependencyStatements    = 5_000
	MaxDependencyStatementSize = 20_000
	MaxDependencyTotalBytes    = 500_000
	MaxDependencyEdges         = 5_000
	MaxWorkspaceFiles          = 500
	MaxWorkspaceBytes          = 5_000_000
	MaxWorkspaceFileBytes      = 500_000
)

type QualityAttribute struct {
	Name             string  `yaml:"name" json:"name"`
	Priority         int     `yaml:"priority" json:"priority"`
	Rationale        string  `yaml:"rationale" json:"rationale"`
	MeasurableSignal *string `yaml:"measurable_signal,omitempty" json:"measurable_signal,omitempty"`
}

type ClarificationQuestion struct {
	ID             string  `yaml:"id" json:"id"`
	Question       string  `yaml:"question" json:"question"`
	DecisionImpact string  `yaml:"decision_impact" json:"decision_impact"`
	Critical       bool    `yaml:"critical,omitempty" json:"critical,omitempty"`
	Answer         *string `yaml:"answer,omitempty" json:"answer,omitempty"`
}

type EvidenceClaim struct {
	Kind     string   `yaml:"kind" json:"kind"`
	Claim    string   `yaml:"claim" json:"claim"`
	Evidence []string `yaml:"evidence" json:"evidence"`
}

type ArchitectureOption struct {
	ID           string   `yaml:"id" json:"id"`
	Name         string   `yaml:"name" json:"name"`
	Summary      string   `yaml:"summary" json:"summary"`
	Benefits     []string `yaml:"benefits" json:"benefits"`
	Drawbacks    []string `yaml:"drawbacks" json:"drawbacks"`
	Risks        []string `yaml:"risks" json:"risks"`
	FitScore     int      `yaml:"fit_score" json:"fit_score"`
	FitRationale string   `yaml:"fit_rationale" json:"fit_rationale"`
}

type ComparedArchitectureOption struct {
	ID                 string  `yaml:"id" json:"id"`
	Category           string  `yaml:"category" json:"category"`
	Name               string  `yaml:"name" json:"name"`
	CanonicalReference *string `yaml:"canonical_reference,omitempty" json:"canonical_reference,omitempty"`
	FitScore           int     `yaml:"fit_score" json:"fit_score"`
	FitRationale       string  `yaml:"fit_rationale" json:"fit_rationale"`
	MainBenefit        string  `yaml:"main_benefit" json:"main_benefit"`
	MainLiability      string  `yaml:"main_liability" json:"main_liability"`
	MaterialAssumption string  `yaml:"material_assumption" json:"material_assumption"`
}

type SupportingPattern struct {
	Category           string `yaml:"category" json:"category"`
	Name               string `yaml:"name" json:"name"`
	CanonicalReference string `yaml:"canonical_reference" json:"canonical_reference"`
	Role               string `yaml:"role" json:"role"`
}

type ArchitectureOptionComparison struct {
	DecisionScope           string                       `yaml:"decision_scope" json:"decision_scope"`
	ScoringCriteria         []string                     `yaml:"scoring_criteria" json:"scoring_criteria"`
	EvidenceAndAssumptions  []EvidenceClaim              `yaml:"evidence_and_assumptions" json:"evidence_and_assumptions"`
	Alternatives            []ComparedArchitectureOption `yaml:"alternatives" json:"alternatives"`
	FewerThanThreeRationale *string                      `yaml:"fewer_than_three_rationale,omitempty" json:"fewer_than_three_rationale,omitempty"`
	RecommendedOptionID     string                       `yaml:"recommended_option_id" json:"recommended_option_id"`
	RecommendationRationale string                       `yaml:"recommendation_rationale" json:"recommendation_rationale"`
	SupportingPatterns      []SupportingPattern          `yaml:"supporting_patterns" json:"supporting_patterns"`
	UserDecisionPrompt      string                       `yaml:"user_decision_prompt" json:"user_decision_prompt"`
	OfferedActions          []string                     `yaml:"offered_actions" json:"offered_actions"`
}

type Decision struct {
	ID                   string   `yaml:"id" json:"id"`
	Title                string   `yaml:"title" json:"title"`
	Status               string   `yaml:"status" json:"status"`
	Context              string   `yaml:"context" json:"context"`
	Drivers              []string `yaml:"drivers" json:"drivers"`
	ConsideredOptionIDs  []string `yaml:"considered_option_ids" json:"considered_option_ids"`
	SelectedOptionID     *string  `yaml:"selected_option_id,omitempty" json:"selected_option_id,omitempty"`
	Decision             string   `yaml:"decision" json:"decision"`
	PositiveConsequences []string `yaml:"positive_consequences" json:"positive_consequences"`
	NegativeConsequences []string `yaml:"negative_consequences" json:"negative_consequences"`
	Assumptions          []string `yaml:"assumptions" json:"assumptions"`
	ValidationCriteria   []string `yaml:"validation_criteria" json:"validation_criteria"`
	Supersedes           []string `yaml:"supersedes" json:"supersedes"`
}

type DecisionArtifact struct {
	SchemaVersion string   `yaml:"schema_version" json:"schema_version"`
	Revision      int      `yaml:"revision" json:"revision"`
	Decision      Decision `yaml:"decision" json:"decision"`
}

type Component struct {
	ID               string   `yaml:"id" json:"id"`
	Responsibility   string   `yaml:"responsibility" json:"responsibility"`
	OwnsData         []string `yaml:"owns_data" json:"owns_data"`
	PublicInterfaces []string `yaml:"public_interfaces" json:"public_interfaces"`
}

type ExternalBoundary struct {
	ID             string `yaml:"id" json:"id"`
	Responsibility string `yaml:"responsibility" json:"responsibility"`
}

type DependencyRule struct {
	Source       string  `yaml:"source" json:"source"`
	Target       string  `yaml:"target" json:"target"`
	Policy       string  `yaml:"policy" json:"policy"`
	ViaInterface *string `yaml:"via_interface,omitempty" json:"via_interface,omitempty"`
	Rationale    string  `yaml:"rationale" json:"rationale"`
}

type Contract struct {
	SchemaVersion       string                  `yaml:"schema_version" json:"schema_version"`
	Revision            int                     `yaml:"revision" json:"revision"`
	Scope               string                  `yaml:"scope" json:"scope"`
	ArchitectureStyle   *string                 `yaml:"architecture_style,omitempty" json:"architecture_style,omitempty"`
	QualityAttributes   []QualityAttribute      `yaml:"quality_attributes" json:"quality_attributes"`
	Components          []Component             `yaml:"components" json:"components"`
	ExternalBoundaries  []ExternalBoundary      `yaml:"external_boundaries" json:"external_boundaries"`
	DependencyRules     []DependencyRule        `yaml:"dependency_rules" json:"dependency_rules"`
	RequiredPractices   []string                `yaml:"required_practices" json:"required_practices"`
	ProhibitedPractices []string                `yaml:"prohibited_practices" json:"prohibited_practices"`
	DecisionIDs         []string                `yaml:"decision_ids" json:"decision_ids"`
	UnresolvedQuestions []ClarificationQuestion `yaml:"unresolved_questions" json:"unresolved_questions"`
}

type ArtifactBundle struct {
	Contract       Contract           `yaml:"contract" json:"contract"`
	Decisions      []DecisionArtifact `yaml:"decisions" json:"decisions"`
	ProjectContext string             `yaml:"project_context" json:"project_context"`
	CodingHandoff  string             `yaml:"coding_handoff" json:"coding_handoff"`
}

type ConformanceFinding struct {
	ID             string   `yaml:"id" json:"id"`
	Classification string   `yaml:"classification" json:"classification"`
	Severity       string   `yaml:"severity" json:"severity"`
	Confidence     string   `yaml:"confidence" json:"confidence"`
	DecisionID     *string  `yaml:"decision_id,omitempty" json:"decision_id,omitempty"`
	Rule           string   `yaml:"rule" json:"rule"`
	Evidence       []string `yaml:"evidence" json:"evidence"`
	Recommendation string   `yaml:"recommendation" json:"recommendation"`
}

type ConformanceReport struct {
	Scope         string               `yaml:"scope" json:"scope"`
	Findings      []ConformanceFinding `yaml:"findings" json:"findings"`
	FilesExamined int                  `yaml:"files_examined" json:"files_examined"`
	FilesSkipped  int                  `yaml:"files_skipped" json:"files_skipped"`
	Claims        []EvidenceClaim      `yaml:"claims" json:"claims"`
	Warnings      []string             `yaml:"warnings" json:"warnings"`
	Truncated     bool                 `yaml:"truncated" json:"truncated"`
}

type ContractValidationResult struct {
	Valid         bool     `json:"valid"`
	SchemaVersion string   `json:"schema_version,omitempty"`
	Errors        []string `json:"errors,omitempty"`
	Warnings      []string `json:"warnings,omitempty"`
	Truncated     bool     `json:"truncated,omitempty"`
}

type SecretFinding struct {
	Category string `json:"category"`
	Line     int    `json:"line"`
}

type ArtifactSecretScanResult struct {
	SafeToWrite bool            `json:"safe_to_write"`
	Findings    []SecretFinding `json:"findings,omitempty"`
	Truncated   bool            `json:"truncated,omitempty"`
}

var (
	adrIDPattern       = regexp.MustCompile(`^ADR-[0-9]{3}$`)
	optionIDPattern    = regexp.MustCompile(`^OPT-[0-9]{3}$`)
	componentIDPattern = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)
	questionIDPattern  = regexp.MustCompile(`^Q-[0-9]{3}$`)
)

func ValidateContractYAML(content string) (ContractValidationResult, *Contract) {
	var contract Contract
	if err := decodeStrictYAML(content, &contract); err != nil {
		return invalidContractResult([]string{err.Error()}), nil
	}
	normalizeContract(&contract)
	errors := validateContract(&contract)
	if len(errors) > MaxErrors {
		return ContractValidationResult{Valid: false, Errors: errors[:MaxErrors], Truncated: true}, nil
	}
	if len(errors) > 0 {
		return ContractValidationResult{Valid: false, Errors: errors}, nil
	}
	return ContractValidationResult{Valid: true, SchemaVersion: CurrentSchemaVersion}, &contract
}

func ParseContractYAML(content string) (Contract, error) {
	result, contract := ValidateContractYAML(content)
	if !result.Valid || contract == nil {
		return Contract{}, errorsAsText(result.Errors)
	}
	return *contract, nil
}

func ParseDecisionArtifactYAML(content string) (DecisionArtifact, error) {
	var artifact DecisionArtifact
	if err := decodeStrictYAML(content, &artifact); err != nil {
		return DecisionArtifact{}, err
	}
	normalizeDecisionArtifact(&artifact)
	errors := validateDecisionArtifact(&artifact)
	if len(errors) > 0 {
		return DecisionArtifact{}, errorsAsText(errors)
	}
	return artifact, nil
}

func ParseADRFrontmatter(content string) (DecisionArtifact, error) {
	lines := strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return DecisionArtifact{}, fmt.Errorf("ADR must begin with YAML frontmatter")
	}
	end := -1
	for index := 1; index < len(lines); index++ {
		if strings.TrimSpace(lines[index]) == "---" {
			end = index
			break
		}
	}
	if end < 0 {
		return DecisionArtifact{}, fmt.Errorf("ADR YAML frontmatter is not terminated")
	}
	return ParseDecisionArtifactYAML(strings.Join(lines[1:end], "\n"))
}

func ValidateBundle(bundle ArtifactBundle) error {
	if err := validateContract(&bundle.Contract); err != nil {
		return errorsAsText(err)
	}
	if len(bundle.Decisions) < 1 || len(bundle.Decisions) > 200 {
		return fmt.Errorf("decisions must contain between 1 and 200 items")
	}
	seen := make(map[string]bool, len(bundle.Decisions))
	for index := range bundle.Decisions {
		if err := validateDecisionArtifact(&bundle.Decisions[index]); err != nil {
			return errorsAsText(err)
		}
		id := bundle.Decisions[index].Decision.ID
		if seen[id] {
			return fmt.Errorf("bundle decision ids must be unique")
		}
		seen[id] = true
		if bundle.Decisions[index].Decision.Status != "accepted" {
			return fmt.Errorf("record-and-handoff bundles require accepted decisions")
		}
	}
	expected := make(map[string]bool, len(bundle.Contract.DecisionIDs))
	for _, id := range bundle.Contract.DecisionIDs {
		if expected[id] {
			return fmt.Errorf("decision_ids must be unique")
		}
		expected[id] = true
	}
	if len(expected) != len(seen) {
		return fmt.Errorf("bundle decisions must exactly match architecture-contract decision_ids")
	}
	for id := range expected {
		if !seen[id] {
			return fmt.Errorf("bundle decisions must exactly match architecture-contract decision_ids")
		}
	}
	if !validNarrative(bundle.ProjectContext, 20_000) || !validNarrative(bundle.CodingHandoff, 20_000) {
		return fmt.Errorf("project_context and coding_handoff must be non-empty and at most 20000 characters")
	}
	return nil
}

func ValidateDecisionArtifact(artifact DecisionArtifact) error {
	errors := validateDecisionArtifact(&artifact)
	if len(errors) > 0 {
		return errorsAsText(errors)
	}
	return nil
}

func validateContract(contract *Contract) []string {
	var errors []string
	if contract.SchemaVersion != CurrentSchemaVersion {
		errors = append(errors, fmt.Sprintf("schema_version: must be exactly %s", CurrentSchemaVersion))
	}
	if contract.Revision < 1 {
		errors = append(errors, "revision: must be at least 1")
	}
	if !validText(contract.Scope, 1, 500) {
		errors = append(errors, "scope: must contain 1 to 500 characters")
	}
	if contract.ArchitectureStyle != nil && !validText(*contract.ArchitectureStyle, 1, 500) {
		errors = append(errors, "architecture_style: must contain 1 to 500 characters")
	}
	if len(contract.QualityAttributes) > 20 {
		errors = append(errors, "quality_attributes: must contain at most 20 items")
	}
	qualityNames := map[string]bool{}
	for index, item := range contract.QualityAttributes {
		prefix := fmt.Sprintf("quality_attributes[%d]", index)
		if !validText(item.Name, 1, 500) {
			errors = append(errors, prefix+".name: invalid text")
		}
		if item.Priority < 1 || item.Priority > 5 {
			errors = append(errors, prefix+".priority: must be between 1 and 5")
		}
		if !validText(item.Rationale, 1, 2_000) {
			errors = append(errors, prefix+".rationale: invalid text")
		}
		if item.MeasurableSignal != nil && !validText(*item.MeasurableSignal, 1, 2_000) {
			errors = append(errors, prefix+".measurable_signal: invalid text")
		}
		key := strings.ToLower(strings.TrimSpace(item.Name))
		if qualityNames[key] {
			errors = append(errors, "quality_attributes: names must be unique")
		}
		qualityNames[key] = true
	}
	if len(contract.Components) > 200 {
		errors = append(errors, "components: must contain at most 200 items")
	}
	if len(contract.ExternalBoundaries) > 100 {
		errors = append(errors, "external_boundaries: must contain at most 100 items")
	}
	nodeIDs := map[string]bool{}
	for index, item := range contract.Components {
		prefix := fmt.Sprintf("components[%d]", index)
		if !componentIDPattern.MatchString(item.ID) {
			errors = append(errors, prefix+".id: invalid component id")
		}
		if !validText(item.Responsibility, 1, 2_000) {
			errors = append(errors, prefix+".responsibility: invalid text")
		}
		validateTextList(&errors, prefix+".owns_data", item.OwnsData, 0, 100, 500)
		validateTextList(&errors, prefix+".public_interfaces", item.PublicInterfaces, 0, 100, 500)
		if nodeIDs[item.ID] {
			errors = append(errors, "component and external-boundary ids must be unique")
		}
		nodeIDs[item.ID] = true
	}
	for index, item := range contract.ExternalBoundaries {
		prefix := fmt.Sprintf("external_boundaries[%d]", index)
		if !componentIDPattern.MatchString(item.ID) {
			errors = append(errors, prefix+".id: invalid boundary id")
		}
		if !validText(item.Responsibility, 1, 2_000) {
			errors = append(errors, prefix+".responsibility: invalid text")
		}
		if nodeIDs[item.ID] {
			errors = append(errors, "component and external-boundary ids must be unique")
		}
		nodeIDs[item.ID] = true
	}
	if len(contract.DependencyRules) > 500 {
		errors = append(errors, "dependency_rules: must contain at most 500 items")
	}
	for index, item := range contract.DependencyRules {
		prefix := fmt.Sprintf("dependency_rules[%d]", index)
		if !componentIDPattern.MatchString(item.Source) {
			errors = append(errors, prefix+".source: invalid id")
		}
		if !componentIDPattern.MatchString(item.Target) {
			errors = append(errors, prefix+".target: invalid id")
		}
		if item.Policy != "allow" && item.Policy != "deny" && item.Policy != "allow-via-interface" {
			errors = append(errors, prefix+".policy: invalid policy")
		}
		if (item.Policy == "allow-via-interface") != (item.ViaInterface != nil) {
			errors = append(errors, prefix+".via_interface: required only for allow-via-interface")
		}
		if item.ViaInterface != nil && !validText(*item.ViaInterface, 1, 500) {
			errors = append(errors, prefix+".via_interface: invalid text")
		}
		if !validText(item.Rationale, 1, 2_000) {
			errors = append(errors, prefix+".rationale: invalid text")
		}
		if !nodeIDs[item.Source] || !nodeIDs[item.Target] {
			errors = append(errors, prefix+": dependency rules must reference declared nodes")
		}
	}
	validateTextList(&errors, "required_practices", contract.RequiredPractices, 0, 200, 500)
	validateTextList(&errors, "prohibited_practices", contract.ProhibitedPractices, 0, 200, 500)
	if len(contract.DecisionIDs) > 200 {
		errors = append(errors, "decision_ids: must contain at most 200 items")
	}
	seenDecisions := map[string]bool{}
	for _, id := range contract.DecisionIDs {
		if !adrIDPattern.MatchString(id) {
			errors = append(errors, "decision_ids: invalid ADR id")
		}
		if seenDecisions[id] {
			errors = append(errors, "decision_ids must be unique")
		}
		seenDecisions[id] = true
	}
	if len(contract.UnresolvedQuestions) > 50 {
		errors = append(errors, "unresolved_questions: must contain at most 50 items")
	}
	seenQuestions := map[string]bool{}
	for index, item := range contract.UnresolvedQuestions {
		prefix := fmt.Sprintf("unresolved_questions[%d]", index)
		if !questionIDPattern.MatchString(item.ID) {
			errors = append(errors, prefix+".id: invalid question id")
		}
		if !validText(item.Question, 1, 2_000) {
			errors = append(errors, prefix+".question: invalid text")
		}
		if !validText(item.DecisionImpact, 1, 2_000) {
			errors = append(errors, prefix+".decision_impact: invalid text")
		}
		if item.Answer != nil && !validText(*item.Answer, 1, 20_000) {
			errors = append(errors, prefix+".answer: invalid text")
		}
		if seenQuestions[item.ID] {
			errors = append(errors, "unresolved question ids must be unique")
		}
		seenQuestions[item.ID] = true
	}
	return errors
}

func validateDecisionArtifact(artifact *DecisionArtifact) []string {
	var errors []string
	if artifact.SchemaVersion != CurrentSchemaVersion {
		errors = append(errors, fmt.Sprintf("schema_version: must be exactly %s", CurrentSchemaVersion))
	}
	if artifact.Revision < 1 {
		errors = append(errors, "revision: must be at least 1")
	}
	decision := &artifact.Decision
	if !adrIDPattern.MatchString(decision.ID) {
		errors = append(errors, "decision.id: invalid ADR id")
	}
	if !validText(decision.Title, 1, 500) {
		errors = append(errors, "decision.title: invalid text")
	}
	validStatuses := map[string]bool{"proposed": true, "accepted": true, "rejected": true, "superseded": true}
	if !validStatuses[decision.Status] {
		errors = append(errors, "decision.status: invalid status")
	}
	if !validNarrative(decision.Context, 20_000) {
		errors = append(errors, "decision.context: invalid text")
	}
	validateTextList(&errors, "decision.drivers", decision.Drivers, 1, 30, 2_000)
	validateIDList(&errors, "decision.considered_option_ids", decision.ConsideredOptionIDs, 1, 5, optionIDPattern, true)
	if decision.SelectedOptionID != nil {
		if !optionIDPattern.MatchString(*decision.SelectedOptionID) || !contains(decision.ConsideredOptionIDs, *decision.SelectedOptionID) {
			errors = append(errors, "decision.selected_option_id: must reference a considered option")
		}
	}
	if decision.Status == "accepted" && decision.SelectedOptionID == nil {
		errors = append(errors, "accepted decisions require selected_option_id")
	}
	if !validNarrative(decision.Decision, 20_000) {
		errors = append(errors, "decision.decision: invalid text")
	}
	validateTextList(&errors, "decision.positive_consequences", decision.PositiveConsequences, 0, 30, 2_000)
	validateTextList(&errors, "decision.negative_consequences", decision.NegativeConsequences, 0, 30, 2_000)
	validateTextList(&errors, "decision.assumptions", decision.Assumptions, 0, 30, 2_000)
	validateTextList(&errors, "decision.validation_criteria", decision.ValidationCriteria, 1, 30, 2_000)
	validateIDList(&errors, "decision.supersedes", decision.Supersedes, 0, 100, adrIDPattern, false)
	if contains(decision.Supersedes, decision.ID) {
		errors = append(errors, "a decision cannot supersede itself")
	}
	return errors
}

func validateTextList(errors *[]string, field string, values []string, min, max, maxLength int) {
	if len(values) < min || len(values) > max {
		*errors = append(*errors, fmt.Sprintf("%s: must contain between %d and %d items", field, min, max))
	}
	for index, value := range values {
		if !validText(value, 1, maxLength) {
			*errors = append(*errors, fmt.Sprintf("%s[%d]: invalid text", field, index))
		}
	}
}

func validateIDList(errors *[]string, field string, values []string, min, max int, pattern *regexp.Regexp, unique bool) {
	if len(values) < min || len(values) > max {
		*errors = append(*errors, fmt.Sprintf("%s: must contain between %d and %d items", field, min, max))
	}
	seen := map[string]bool{}
	for _, value := range values {
		if !pattern.MatchString(value) {
			*errors = append(*errors, fmt.Sprintf("%s: invalid id %q", field, value))
		}
		if unique && seen[value] {
			*errors = append(*errors, fmt.Sprintf("%s: ids must be unique", field))
		}
		seen[value] = true
	}
}

func validText(value string, min, max int) bool {
	value = strings.TrimSpace(value)
	length := utf8.RuneCountInString(value)
	return length >= min && length <= max
}

func validNarrative(value string, max int) bool { return validText(value, 1, max) }

func contains(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}

func errorsAsText(errors []string) error {
	if len(errors) == 0 {
		return nil
	}
	return fmt.Errorf("%s", strings.Join(errors, "; "))
}

func invalidContractResult(errors []string) ContractValidationResult {
	return ContractValidationResult{Valid: false, Errors: errors}
}

func normalizeContract(contract *Contract) {
	if contract.QualityAttributes == nil {
		contract.QualityAttributes = []QualityAttribute{}
	}
	if contract.Components == nil {
		contract.Components = []Component{}
	}
	if contract.ExternalBoundaries == nil {
		contract.ExternalBoundaries = []ExternalBoundary{}
	}
	if contract.DependencyRules == nil {
		contract.DependencyRules = []DependencyRule{}
	}
	if contract.RequiredPractices == nil {
		contract.RequiredPractices = []string{}
	}
	if contract.ProhibitedPractices == nil {
		contract.ProhibitedPractices = []string{}
	}
	if contract.DecisionIDs == nil {
		contract.DecisionIDs = []string{}
	}
	if contract.UnresolvedQuestions == nil {
		contract.UnresolvedQuestions = []ClarificationQuestion{}
	}
}

func normalizeDecisionArtifact(artifact *DecisionArtifact) {
	if artifact.Decision.Drivers == nil {
		artifact.Decision.Drivers = []string{}
	}
	if artifact.Decision.ConsideredOptionIDs == nil {
		artifact.Decision.ConsideredOptionIDs = []string{}
	}
	if artifact.Decision.PositiveConsequences == nil {
		artifact.Decision.PositiveConsequences = []string{}
	}
	if artifact.Decision.NegativeConsequences == nil {
		artifact.Decision.NegativeConsequences = []string{}
	}
	if artifact.Decision.Assumptions == nil {
		artifact.Decision.Assumptions = []string{}
	}
	if artifact.Decision.ValidationCriteria == nil {
		artifact.Decision.ValidationCriteria = []string{}
	}
	if artifact.Decision.Supersedes == nil {
		artifact.Decision.Supersedes = []string{}
	}
}

func SortedStringSet(values []string) []string {
	result := append([]string(nil), values...)
	sort.Strings(result)
	return result
}
