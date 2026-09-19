package architect

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"
)

const (
	maxBundleDecisions      = 200
	maxBundleArtifactBytes  = 500_000
	maxBundleNarrativeChars = 20_000
	maxBundleTotalBytes     = 5_000_000
)

// ValidateArtifactBundle validates the complete record-and-handoff unit before
// persistence. It intentionally keeps procedure, history, and handoff content
// separate while enforcing the references between the structured artifacts.
func ValidateArtifactBundle(input ArtifactBundleInput) (ArtifactBundle, ValidationResult) {
	bundle := ArtifactBundle{
		Decisions:      make([]DecisionArtifact, 0, len(input.Decisions)),
		ProjectContext: input.ProjectContext,
		CodingHandoff:  input.CodingHandoff,
	}
	result := ValidationResult{Valid: true}
	totalBytes := 0

	addError := func(path, message, code string) {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationIssue{
			Path: path, Message: message, Severity: "error", Code: code,
		})
	}
	addPrefixed := func(prefix string, child ValidationResult) {
		if !child.Valid {
			result.Valid = false
		}
		for _, issue := range child.Errors {
			issue.Path = prefixedPath(prefix, issue.Path)
			result.Errors = append(result.Errors, issue)
		}
		for _, issue := range child.Warnings {
			issue.Path = prefixedPath(prefix, issue.Path)
			result.Warnings = append(result.Warnings, issue)
		}
	}
	trackBytes := func(path, content string) bool {
		bytes := len(content)
		if bytes > maxBundleArtifactBytes {
			addError(path, fmt.Sprintf("artifact exceeds %d-byte limit", maxBundleArtifactBytes), "artifact_too_large")
			return false
		}
		totalBytes += bytes
		if totalBytes > maxBundleTotalBytes {
			addError("bundle", fmt.Sprintf("bundle exceeds %d-byte total limit", maxBundleTotalBytes), "bundle_too_large")
			return false
		}
		return true
	}
	scan := func(path, content string) {
		scanResult := scanSecrets(content)
		if scanResult.Truncated {
			result.Truncated = true
		}
		for _, finding := range scanResult.Findings {
			result.Valid = false
			result.Errors = append(result.Errors, ValidationIssue{
				Path:     fmt.Sprintf("%s.line:%d", path, finding.Line),
				Message:  "secret-like content is not allowed in generated artifacts",
				Severity: "error",
				Code:     finding.Category,
			})
		}
	}
	validateNarrative := func(path, content string) {
		trimmed := strings.TrimSpace(content)
		if trimmed == "" {
			addError(path, "required narrative content is missing", "missing_narrative")
		}
		if utf8.RuneCountInString(content) > maxBundleNarrativeChars {
			addError(path, fmt.Sprintf("narrative exceeds %d characters", maxBundleNarrativeChars), "narrative_too_large")
		}
		if trackBytes(path, content) {
			scan(path, content)
		}
	}

	if trackBytes("contract", input.ContractYAML) {
		scan("contract", input.ContractYAML)
	}
	contract, contractResult := ValidateContract(input.ContractYAML)
	bundle.Contract = contract
	addPrefixed("contract", contractResult)

	validateNarrative("project_context", input.ProjectContext)
	validateNarrative("coding_handoff", input.CodingHandoff)

	if len(input.Decisions) == 0 {
		addError("decisions", "at least one decision is required", "missing_decisions")
	}
	if len(input.Decisions) > maxBundleDecisions {
		addError("decisions", fmt.Sprintf("at most %d decisions are allowed", maxBundleDecisions), "too_many_decisions")
	}

	decisionIDs := make([]string, 0, len(input.Decisions))
	seenDecisionIDs := map[string]struct{}{}
	for index, candidate := range input.Decisions {
		path := fmt.Sprintf("decisions[%d]", index)
		filename := strings.TrimSpace(candidate.Filename)
		if !decisionFilenamePattern.MatchString(filename) {
			addError(path+".filename", "must match ADR-NNN[-slug].md", "invalid_filename")
		}
		if !trackBytes(path, candidate.Content) {
			continue
		}
		scan(path, candidate.Content)
		record, err := parseDecision(candidate.Content, filename, "")
		if err != nil {
			addError(path, "could not parse decision: "+err.Error(), "invalid_decision")
			continue
		}
		decisionResult := validateDecision(record)
		addPrefixed(path, decisionResult)
		decision := record.Artifact.Decision
		decisionIDs = append(decisionIDs, decision.ID)
		if _, exists := seenDecisionIDs[decision.ID]; exists {
			addError(path+".decision.id", "bundle decision ids must be unique", "duplicate_decision_id")
		}
		seenDecisionIDs[decision.ID] = struct{}{}
		if decision.Status != "accepted" {
			addError(path+".decision.status", "record-and-handoff bundles require accepted decisions", "decision_not_accepted")
		}
		bundle.Decisions = append(bundle.Decisions, record.Artifact)
	}

	if !sameStringSet(contract.DecisionIDs, decisionIDs) {
		addError("decisions", "bundle decisions must exactly match architecture-contract decision_ids", "decision_reference_mismatch")
	}
	return bundle, result
}

// ValidateProjectBundle loads only the four canonical artifact locations and
// applies ValidateArtifactBundle. Missing files become validation errors rather
// than opaque filesystem failures so an agent can repair the bundle safely.
func ValidateProjectBundle(dir string) (ArtifactBundle, ValidationResult, error) {
	root, err := projectRoot(dir)
	if err != nil {
		return ArtifactBundle{}, ValidationResult{}, err
	}
	read := func(relative string) (string, error) {
		safe, pathErr := relativeSafe(root, relative)
		if pathErr != nil {
			return "", pathErr
		}
		if pathErr := ensureReadableTarget(root, filepath.Join(root, filepath.FromSlash(safe))); pathErr != nil {
			return "", pathErr
		}
		content, readErr := readProjectText(root, relative, nil)
		if errors.Is(readErr, os.ErrNotExist) {
			return "", nil
		}
		return content, readErr
	}
	contract, err := read(ContractFilename)
	if err != nil {
		return ArtifactBundle{}, ValidationResult{}, err
	}
	context, err := read(ContextFilename)
	if err != nil {
		return ArtifactBundle{}, ValidationResult{}, err
	}
	handoff, err := read(HandoffFilename)
	if err != nil {
		return ArtifactBundle{}, ValidationResult{}, err
	}
	filenames, err := listDecisionFiles(filepath.Join(root, filepath.FromSlash(DecisionsDir)))
	if err != nil {
		return ArtifactBundle{}, ValidationResult{}, err
	}
	decisions := make([]BundleDecisionInput, 0, len(filenames))
	for _, filename := range filenames {
		content, readErr := read(filepath.ToSlash(filepath.Join(DecisionsDir, filename)))
		if readErr != nil {
			return ArtifactBundle{}, ValidationResult{}, readErr
		}
		decisions = append(decisions, BundleDecisionInput{Filename: filename, Content: content})
	}
	bundle, result := ValidateArtifactBundle(ArtifactBundleInput{
		ContractYAML:   contract,
		Decisions:      decisions,
		ProjectContext: context,
		CodingHandoff:  handoff,
	})
	return bundle, result, nil
}

func SummarizeArtifactBundle(bundle ArtifactBundle, result ValidationResult) ArtifactBundleSummary {
	decisionIDs := make([]string, 0, len(bundle.Decisions))
	for _, decision := range bundle.Decisions {
		decisionIDs = append(decisionIDs, decision.Decision.ID)
	}
	sort.Strings(decisionIDs)
	contractIDs := append([]string{}, bundle.Contract.DecisionIDs...)
	sort.Strings(contractIDs)
	return ArtifactBundleSummary{
		Result:              result,
		ContractDecisionIDs: contractIDs,
		DecisionIDs:         decisionIDs,
		DecisionCount:       len(bundle.Decisions),
	}
}

func prefixedPath(prefix, path string) string {
	if path == "" {
		return prefix
	}
	return prefix + "." + path
}

func sameStringSet(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	leftSet := make(map[string]struct{}, len(left))
	rightSet := make(map[string]struct{}, len(right))
	for _, value := range left {
		leftSet[value] = struct{}{}
	}
	for _, value := range right {
		rightSet[value] = struct{}{}
	}
	if len(leftSet) != len(rightSet) {
		return false
	}
	for value := range leftSet {
		if _, ok := rightSet[value]; !ok {
			return false
		}
	}
	return true
}
