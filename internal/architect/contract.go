package architect

import (
	"fmt"
	"regexp"
	"strings"
)

const maxValidationErrors = 100

var (
	semanticVersionPattern = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+$`)
	componentIDPattern     = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)
	questionIDPattern      = regexp.MustCompile(`^Q-[0-9]{3}$`)
	secretPatterns         = []*regexp.Regexp{
		regexp.MustCompile(`-----BEGIN (?:[A-Z0-9 ]+ )?PRIVATE KEY-----`),
		regexp.MustCompile(`\bsk-[A-Za-z0-9_-]{20,}\b`),
		regexp.MustCompile(`\bgh[pousr]_[A-Za-z0-9]{20,}\b`),
		regexp.MustCompile(`\bAKIA[A-Z0-9]{16}\b`),
	}
	credentialPattern = regexp.MustCompile(`(?i)\b(?:password|passwd|api[_-]?key|client[_-]?secret|access[_-]?token)\b\s*[:=]\s*["']?([^\s"'#,;]{12,})`)
)

var safeCredentialMarkers = []string{"example", "placeholder", "changeme", "not-a-real", "${", "{{", "<"}

func validateContractYAML(content string) (ArchitectureContract, ValidationResult) {
	var contract ArchitectureContract
	result := ValidationResult{Valid: true}
	addError := func(path, message, code string) {
		if len(result.Errors) >= maxValidationErrors {
			result.Truncated = true
			return
		}
		result.Valid = false
		result.Errors = append(result.Errors, ValidationIssue{Path: path, Message: message, Severity: "error", Code: code})
	}
	if strings.TrimSpace(content) == "" {
		addError("document", "architecture contract is empty", "empty_document")
		return contract, result
	}
	if err := unmarshalSafe([]byte(content), &contract); err != nil {
		addError("document", err.Error(), "invalid_yaml")
		return contract, result
	}
	if !semanticVersionPattern.MatchString(contract.SchemaVersion) {
		addError("schema_version", "must match major.minor.patch", "invalid_schema_version")
	}
	if contract.Revision < 1 {
		addError("revision", "must be at least 1", "invalid_revision")
	}
	if !validText(contract.Scope, 1, 500) {
		addError("scope", "required field is missing", "missing_scope")
	}
	if len(contract.QualityAttributes) > 20 {
		addError("quality_attributes", "must contain at most 20 items", "too_many_quality_attributes")
	}
	if len(contract.Components) > 200 {
		addError("components", "must contain at most 200 items", "too_many_components")
	}
	if len(contract.ExternalBoundaries) > 100 {
		addError("external_boundaries", "must contain at most 100 items", "too_many_external_boundaries")
	}
	if len(contract.DependencyRules) > 500 {
		addError("dependency_rules", "must contain at most 500 items", "too_many_dependency_rules")
	}
	if len(contract.RequiredPractices) > 200 {
		addError("required_practices", "must contain at most 200 items", "too_many_required_practices")
	}
	if len(contract.ProhibitedPractices) > 200 {
		addError("prohibited_practices", "must contain at most 200 items", "too_many_prohibited_practices")
	}
	if len(contract.DecisionIDs) > 200 {
		addError("decision_ids", "must contain at most 200 items", "too_many_decision_ids")
	}
	if len(contract.UnresolvedQuestions) > 50 {
		addError("unresolved_questions", "must contain at most 50 items", "too_many_unresolved_questions")
	}
	if contract.ArchitectureStyle != nil && !validText(*contract.ArchitectureStyle, 1, 500) {
		addError("architecture_style", "must be between 1 and 500 characters", "invalid_architecture_style")
	}
	nodes := make(map[string]struct{}, len(contract.Components)+len(contract.ExternalBoundaries))
	for index, component := range contract.Components {
		path := fmt.Sprintf("components[%d]", index)
		if !componentIDPattern.MatchString(component.ID) {
			addError(path+".id", "must be a lowercase component identifier", "invalid_component_id")
		}
		if !validText(component.Responsibility, 1, 2000) {
			addError(path+".responsibility", "must be between 1 and 2000 characters", "invalid_responsibility")
		}
		validateTextList(addError, path+".owns_data", component.OwnsData, 100, 1, 500)
		validateTextList(addError, path+".public_interfaces", component.PublicInterfaces, 100, 1, 500)
		if len(component.OwnsData) > 100 {
			addError(path+".owns_data", "must contain at most 100 items", "too_many_owns_data")
		}
		if len(component.PublicInterfaces) > 100 {
			addError(path+".public_interfaces", "must contain at most 100 items", "too_many_public_interfaces")
		}
		if _, exists := nodes[component.ID]; exists {
			addError(path+".id", "component and boundary ids must be unique", "duplicate_node_id")
		}
		nodes[component.ID] = struct{}{}
	}
	for index, boundary := range contract.ExternalBoundaries {
		path := fmt.Sprintf("external_boundaries[%d]", index)
		if !componentIDPattern.MatchString(boundary.ID) {
			addError(path+".id", "must be a lowercase boundary identifier", "invalid_boundary_id")
		}
		if !validText(boundary.Responsibility, 1, 2000) {
			addError(path+".responsibility", "must be between 1 and 2000 characters", "invalid_responsibility")
		}
		if _, exists := nodes[boundary.ID]; exists {
			addError(path+".id", "component and boundary ids must be unique", "duplicate_node_id")
		}
		nodes[boundary.ID] = struct{}{}
	}
	for index, attribute := range contract.QualityAttributes {
		path := fmt.Sprintf("quality_attributes[%d]", index)
		if !validText(attribute.Name, 1, 500) {
			addError(path+".name", "must be between 1 and 500 characters", "invalid_quality_attribute_name")
		}
		if !validText(attribute.Rationale, 1, 2000) {
			addError(path+".rationale", "must be between 1 and 2000 characters", "invalid_quality_attribute_rationale")
		}
		if attribute.Priority < 1 || attribute.Priority > 5 {
			addError(path+".priority", "must be between 1 and 5", "invalid_priority")
		}
		if attribute.MeasurableSignal != nil && !validText(*attribute.MeasurableSignal, 1, 2000) {
			addError(path+".measurable_signal", "must be between 1 and 2000 characters", "invalid_measurable_signal")
		}
	}
	qualityNames := make(map[string]struct{}, len(contract.QualityAttributes))
	for index, attribute := range contract.QualityAttributes {
		name := strings.ToLower(strings.TrimSpace(attribute.Name))
		if _, exists := qualityNames[name]; exists {
			addError(fmt.Sprintf("quality_attributes[%d].name", index), "quality-attribute names must be unique", "duplicate_quality_attribute")
		}
		qualityNames[name] = struct{}{}
	}
	decisionIDs := make(map[string]struct{}, len(contract.DecisionIDs))
	for _, decisionID := range contract.DecisionIDs {
		if !decisionIDPattern.MatchString(decisionID) {
			addError("decision_ids", decisionID+" must match ADR-NNN", "invalid_decision_id")
		}
		if _, exists := decisionIDs[decisionID]; exists {
			addError("decision_ids", "decision ids must be unique", "duplicate_decision_id")
		}
		decisionIDs[decisionID] = struct{}{}
	}
	validateTextList(addError, "required_practices", contract.RequiredPractices, 200, 1, 500)
	validateTextList(addError, "prohibited_practices", contract.ProhibitedPractices, 200, 1, 500)
	for index, question := range contract.UnresolvedQuestions {
		path := fmt.Sprintf("unresolved_questions[%d]", index)
		if !questionIDPattern.MatchString(question.ID) {
			addError(path+".id", "must match Q-NNN", "invalid_question_id")
		}
		if !validText(question.Question, 1, 2000) {
			addError(path+".question", "must be between 1 and 2000 characters", "invalid_question")
		}
		if !validText(question.DecisionImpact, 1, 2000) {
			addError(path+".decision_impact", "must be between 1 and 2000 characters", "invalid_decision_impact")
		}
		if question.Answer != nil && !validText(*question.Answer, 1, 20000) {
			addError(path+".answer", "must be between 1 and 20000 characters", "invalid_answer")
		}
	}
	questionIDs := make(map[string]struct{}, len(contract.UnresolvedQuestions))
	for index, question := range contract.UnresolvedQuestions {
		if _, exists := questionIDs[question.ID]; exists {
			addError(fmt.Sprintf("unresolved_questions[%d].id", index), "unresolved question ids must be unique", "duplicate_question_id")
		}
		questionIDs[question.ID] = struct{}{}
	}
	for index, rule := range contract.DependencyRules {
		path := fmt.Sprintf("dependency_rules[%d]", index)
		if !componentIDPattern.MatchString(rule.Source) {
			addError(path+".source", "must be a lowercase component identifier", "invalid_source")
		}
		if !componentIDPattern.MatchString(rule.Target) {
			addError(path+".target", "must be a lowercase component identifier", "invalid_target")
		}
		if _, exists := nodes[rule.Source]; !exists {
			addError(path+".source", "must reference a declared component or boundary", "unknown_source")
		}
		if _, exists := nodes[rule.Target]; !exists {
			addError(path+".target", "must reference a declared component or boundary", "unknown_target")
		}
		if rule.Policy != "allow" && rule.Policy != "deny" && rule.Policy != "allow-via-interface" {
			addError(path+".policy", "must be allow, deny, or allow-via-interface", "invalid_policy")
		}
		if rule.Policy == "allow-via-interface" && (rule.ViaInterface == nil || !validText(*rule.ViaInterface, 1, 500)) {
			addError(path+".via_interface", "is required for allow-via-interface", "missing_interface")
		}
		if rule.Policy != "allow-via-interface" && rule.ViaInterface != nil {
			addError(path+".via_interface", "must be omitted unless policy is allow-via-interface", "unexpected_interface")
		}
		if !validText(rule.Rationale, 1, 2000) {
			addError(path+".rationale", "must be between 1 and 2000 characters", "missing_rationale")
		}
	}
	return contract, result
}

func validText(value string, minLength, maxLength int) bool {
	trimmed := strings.TrimSpace(value)
	length := len([]rune(trimmed))
	return length >= minLength && length <= maxLength
}

func validateTextList(addError func(string, string, string), path string, values []string, maxItems, minLength, maxLength int) {
	if len(values) > maxItems {
		addError(path, fmt.Sprintf("must contain at most %d items", maxItems), "too_many_items")
	}
	for index, value := range values {
		if !validText(value, minLength, maxLength) {
			addError(fmt.Sprintf("%s[%d]", path, index), fmt.Sprintf("must be between %d and %d characters", minLength, maxLength), "invalid_text")
		}
	}
}

func scanSecrets(content string) SecretScanResult {
	findings := make([]SecretFinding, 0)
	for lineNumber, line := range strings.Split(content, "\n") {
		if secretPatterns[0].MatchString(line) {
			findings = append(findings, SecretFinding{Category: "private-key", Line: lineNumber + 1})
		}
		for _, pattern := range secretPatterns[1:] {
			if pattern.MatchString(line) {
				findings = append(findings, SecretFinding{Category: "token", Line: lineNumber + 1})
				break
			}
		}
		if match := credentialPattern.FindStringSubmatch(line); len(match) == 2 {
			value := strings.ToLower(match[1])
			safe := false
			for _, marker := range safeCredentialMarkers {
				if strings.Contains(value, marker) {
					safe = true
					break
				}
			}
			if !safe {
				findings = append(findings, SecretFinding{Category: "credential", Line: lineNumber + 1})
			}
		}
		if len(findings) >= 100 {
			return SecretScanResult{SafeToWrite: false, Findings: findings, Truncated: true}
		}
	}
	return SecretScanResult{SafeToWrite: len(findings) == 0, Findings: findings}
}

func validateArtifact(content, kind string) ValidationResult {
	scan := scanSecrets(content)
	result := ValidationResult{Valid: scan.SafeToWrite}
	for _, finding := range scan.Findings {
		result.Errors = append(result.Errors, ValidationIssue{
			Path: fmt.Sprintf("line:%d", finding.Line), Message: "secret-like content is not allowed in generated artifacts", Severity: "error", Code: finding.Category,
		})
	}
	if kind == "adr" {
		record, err := parseDecision(content, "artifact.md", "artifact.md")
		if err != nil {
			result.Valid = false
			result.Errors = append(result.Errors, ValidationIssue{Path: "frontmatter", Message: err.Error(), Severity: "error", Code: "invalid_adr"})
		} else {
			decisionResult := validateDecision(record)
			result.Errors = append(result.Errors, decisionResult.Errors...)
			result.Warnings = append(result.Warnings, decisionResult.Warnings...)
			if !decisionResult.Valid {
				result.Valid = false
			}
		}
	} else if kind == "contract" {
		_, contractResult := validateContractYAML(content)
		result.Errors = append(result.Errors, contractResult.Errors...)
		result.Warnings = append(result.Warnings, contractResult.Warnings...)
		if !contractResult.Valid {
			result.Valid = false
		}
	} else if kind != "context" && kind != "implementation-plan" {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationIssue{Path: "artifact_kind", Message: "must be adr, contract, context, or implementation-plan", Severity: "error", Code: "invalid_artifact_kind"})
	}
	result.Truncated = scan.Truncated
	return result
}
