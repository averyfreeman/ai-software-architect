package architect

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	maxDecisionFileBytes = 500_000
	maxDecisions         = 200
)

var (
	decisionIDPattern       = regexp.MustCompile(`^ADR-[0-9]{3}$`)
	decisionFilenamePattern = regexp.MustCompile(`^ADR-[0-9]{3}(?:-[a-z0-9]+(?:-[a-z0-9]+)*)?\.md$`)
	optionIDPattern         = regexp.MustCompile(`^OPT-[0-9]{3}$`)
	datePattern             = regexp.MustCompile(`^[0-9]{4}-[0-9]{2}-[0-9]{2}$`)
	versionPattern          = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+$`)
)

func validStatus(status string) bool {
	switch strings.TrimSpace(status) {
	case "proposed", "accepted", "rejected", "deprecated", "superseded":
		return true
	default:
		return false
	}
}

func validateDecision(record *DecisionRecord) ValidationResult {
	result := ValidationResult{Valid: true}
	addError := func(path, message, code string) {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationIssue{Path: path, Message: message, Severity: "error", Code: code})
	}
	addWarning := func(path, message, code string) {
		result.Warnings = append(result.Warnings, ValidationIssue{Path: path, Message: message, Severity: "warning", Code: code})
	}
	d := record.Artifact.Decision
	status := strings.TrimSpace(d.Status)
	schemaVersion := strings.TrimSpace(record.Artifact.SchemaVersion)
	if !versionPattern.MatchString(record.Artifact.SchemaVersion) {
		addError("schema_version", "must match major.minor.patch", "invalid_schema_version")
	}
	if record.Artifact.Revision < 1 {
		addError("revision", "must be at least 1", "invalid_revision")
	}
	if !decisionIDPattern.MatchString(d.ID) {
		addError("decision.id", "must match ADR-NNN", "invalid_decision_id")
	}
	if !validText(d.Title, 1, 500) {
		addError("decision.title", "must be between 1 and 500 characters", "invalid_title")
	}
	if d.Date != "" && !datePattern.MatchString(strings.TrimSpace(d.Date)) {
		addError("decision.date", "must match YYYY-MM-DD", "invalid_date")
	}
	if !validStatus(d.Status) {
		addError("decision.status", "must be proposed, accepted, rejected, deprecated, or superseded", "invalid_status")
	}
	if record.Filename != "" {
		if !decisionFilenamePattern.MatchString(record.Filename) {
			addError("filename", "must match ADR-NNN[-slug].md", "invalid_filename")
		} else if !strings.HasPrefix(record.Filename, d.ID) {
			addError("filename", "filename and decision.id disagree", "filename_id_mismatch")
		}
	}
	if d.Motivation.Source != "" && d.Motivation.Source != "human" && d.Motivation.Source != "auto-reasoned" {
		addError("decision.motivation.source", "must be human or auto-reasoned", "invalid_motivation_source")
	}
	if d.Motivation.Source == "auto-reasoned" {
		addWarning("decision.motivation.source", "motivation was supplied by auto-reason and must be human-reviewed", "auto_reasoned_motivation")
	}
	if schemaVersion == "1.1.0" {
		if d.Date == "" {
			addError("decision.date", "schema version 1.1.0 decisions require date", "missing_date")
		}
		for field, value := range d.Motivation.Fields() {
			if !validText(value, 1, 2000) {
				addError("decision.motivation."+field, "mandatory project-motivation answer is missing or too long", "missing_motivation")
			}
			if strings.Contains(strings.ToLower(value), "pending agent reasoning") || strings.Contains(strings.ToLower(value), "human review required") {
				if status == "accepted" {
					addError("decision.motivation."+field, "accepted decisions cannot retain provisional motivation placeholders", "unresolved_motivation")
				}
			}
		}
		if len(d.DecisionMatrix) < 2 {
			addError("decision.decision_matrix", "schema version 1.1.0 decisions require at least two matrix options", "missing_decision_matrix")
		}
		if d.ActionContract.CorrectExample == "" || d.ActionContract.IncorrectExample == "" || len(nonEmpty(d.ActionContract.GoodOutcomes)) == 0 || len(nonEmpty(d.ActionContract.BadOutcomes)) == 0 {
			addError("decision.action_contract", "schema version 1.1.0 decisions require an action contract", "missing_action_contract")
		}
	}
	if !validText(d.Context, 1, 20000) {
		addError("decision.context", "must be between 1 and 20000 characters", "missing_context")
	}
	if len(d.Drivers) == 0 || len(d.Drivers) > 30 {
		addError("decision.drivers", "at least one decision driver is required", "missing_drivers")
	}
	validateTextList(addError, "decision.drivers", d.Drivers, 30, 1, 2000)
	if len(d.ConsideredOptionIDs) == 0 || len(d.ConsideredOptionIDs) > 5 {
		addError("decision.considered_option_ids", "must contain between 1 and 5 options", "invalid_option_count")
	}
	seenOptions := map[string]struct{}{}
	for _, optionID := range d.ConsideredOptionIDs {
		if !optionIDPattern.MatchString(optionID) {
			addError("decision.considered_option_ids", optionID+" must match OPT-NNN", "invalid_option_id")
		}
		if _, exists := seenOptions[optionID]; exists {
			addError("decision.considered_option_ids", "option ids must be unique", "duplicate_option_id")
		}
		seenOptions[optionID] = struct{}{}
	}
	if d.SelectedOptionID != nil {
		if _, exists := seenOptions[*d.SelectedOptionID]; !exists {
			addError("decision.selected_option_id", "must reference a considered option", "unknown_selected_option")
		}
	}
	if status == "accepted" && d.SelectedOptionID == nil {
		addError("decision.selected_option_id", "accepted decisions require a selected option", "missing_selected_option")
	}
	matrixIDs := map[string]struct{}{}
	for _, option := range d.DecisionMatrix {
		if !validText(option.Name, 1, 500) {
			addError("decision.decision_matrix."+option.ID+".name", "must be between 1 and 500 characters", "invalid_matrix_option_name")
		}
		validateTextList(addError, "decision.decision_matrix."+option.ID+".benefits", option.Benefits, 20, 1, 2000)
		validateTextList(addError, "decision.decision_matrix."+option.ID+".drawbacks", option.Drawbacks, 20, 1, 2000)
		validateTextList(addError, "decision.decision_matrix."+option.ID+".risks", option.Risks, 20, 1, 2000)
		validateTextList(addError, "decision.decision_matrix."+option.ID+".evidence", option.Evidence, 20, 1, 2000)
		if option.Outcome != "proposed" && option.Outcome != "accepted" && option.Outcome != "rejected" {
			addError("decision.decision_matrix."+option.ID+".outcome", "must be proposed, accepted, or rejected", "invalid_matrix_outcome")
		}
		matrixIDs[option.ID] = struct{}{}
	}
	if len(d.DecisionMatrix) > 0 {
		for _, optionID := range d.ConsideredOptionIDs {
			if _, exists := matrixIDs[optionID]; !exists {
				addError("decision.decision_matrix", "matrix must include every considered option", "matrix_mismatch")
			}
		}
	}
	if !validText(d.Decision, 1, 20000) {
		addError("decision.decision", "must be between 1 and 20000 characters", "missing_decision")
	}
	if len(d.ValidationCriteria) == 0 || len(d.ValidationCriteria) > 30 {
		addError("decision.validation_criteria", "must contain between 1 and 30 criteria", "missing_validation_criteria")
	}
	validateTextList(addError, "decision.validation_criteria", d.ValidationCriteria, 30, 1, 2000)
	validateTextList(addError, "decision.positive_consequences", d.PositiveConsequences, 30, 1, 2000)
	validateTextList(addError, "decision.negative_consequences", d.NegativeConsequences, 30, 1, 2000)
	validateTextList(addError, "decision.assumptions", d.Assumptions, 30, 1, 2000)
	if len(d.Supersedes) > 100 {
		addError("decision.supersedes", "must contain at most 100 decisions", "too_many_supersedes")
	}
	for index, supersededID := range d.Supersedes {
		if !decisionIDPattern.MatchString(supersededID) {
			addError(fmt.Sprintf("decision.supersedes[%d]", index), "must match ADR-NNN", "invalid_superseded_id")
		}
		if supersededID == d.ID {
			addError(fmt.Sprintf("decision.supersedes[%d]", index), "a decision cannot supersede itself", "self_supersede")
		}
	}
	if len(d.DecisionMatrix) > 0 {
		if len(d.DecisionMatrix) > 5 {
			addError("decision.decision_matrix", "must contain at most 5 options", "too_many_matrix_options")
		}
		seenMatrixIDs := map[string]struct{}{}
		for _, option := range d.DecisionMatrix {
			if !optionIDPattern.MatchString(option.ID) {
				addError("decision.decision_matrix", option.ID+" must match OPT-NNN", "invalid_matrix_option_id")
			}
			if option.FitScore < 0 || option.FitScore > 100 {
				addError("decision.decision_matrix."+option.ID, "fit_score must be between 0 and 100", "invalid_fit_score")
			}
			if _, exists := seenMatrixIDs[option.ID]; exists {
				addError("decision.decision_matrix", "option ids must be unique", "duplicate_matrix_option_id")
			}
			seenMatrixIDs[option.ID] = struct{}{}
		}
	}
	if d.ActionContract.CorrectExample != "" || d.ActionContract.IncorrectExample != "" || len(d.ActionContract.GoodOutcomes) > 0 || len(d.ActionContract.BadOutcomes) > 0 {
		validateTextList(addError, "decision.action_contract.good_outcomes", d.ActionContract.GoodOutcomes, 20, 1, 2000)
		validateTextList(addError, "decision.action_contract.bad_outcomes", d.ActionContract.BadOutcomes, 20, 1, 2000)
		if !validText(d.ActionContract.CorrectExample, 1, 2000) {
			addError("decision.action_contract.correct_example", "must be between 1 and 2000 characters", "invalid_correct_example")
		}
		if !validText(d.ActionContract.IncorrectExample, 1, 2000) {
			addError("decision.action_contract.incorrect_example", "must be between 1 and 2000 characters", "invalid_incorrect_example")
		}
	}
	bodySections := []string{"Context", "Decision Drivers", "Decision Matrix", "Decision", "Alternatives Considered", "Consequences", "Agent Action Contract"}
	for _, section := range bodySections {
		if !hasMarkdownSection(record.Body, section) {
			addError("body", "missing required section: "+section, "missing_section")
		}
	}
	if !strings.Contains(strings.ToLower(record.Body), "accepted because:") {
		addWarning("body", "include an explicit 'Accepted because:' rationale when the decision is accepted", "missing_acceptance_rationale")
	}
	if !strings.Contains(strings.ToLower(record.Body), "accepted despite:") {
		addWarning("body", "include an explicit 'Accepted despite:' trade-off", "missing_acceptance_tradeoff")
	}
	return result
}

func nonEmpty(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			result = append(result, value)
		}
	}
	return result
}

func hasMarkdownSection(body, title string) bool {
	for _, line := range strings.Split(body, "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "#") {
			continue
		}
		heading := strings.TrimSpace(strings.TrimLeft(trimmed, "#"))
		if strings.EqualFold(heading, title) {
			return true
		}
	}
	return false
}

func parseDecision(content, filename, path string) (*DecisionRecord, error) {
	var artifact DecisionArtifact
	body, err := frontmatter(content, &artifact)
	if err != nil {
		return nil, err
	}
	return &DecisionRecord{Artifact: artifact, Body: body, Filename: filename, Path: path}, nil
}

func loadDecision(path string) (*DecisionRecord, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() > maxDecisionFileBytes {
		return nil, fmt.Errorf("decision file %s is not a bounded regular file", path)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return parseDecision(string(content), filepath.Base(path), path)
}

func listDecisionFiles(directory string) ([]string, error) {
	entries, err := os.ReadDir(directory)
	if errors.Is(err, os.ErrNotExist) {
		return []string{}, nil
	}
	if err != nil {
		return nil, err
	}
	files := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() && decisionFilenamePattern.MatchString(entry.Name()) {
			files = append(files, entry.Name())
		}
	}
	sort.Slice(files, func(i, j int) bool { return decisionNumber(files[i]) < decisionNumber(files[j]) })
	return files, nil
}

func decisionNumber(filename string) int {
	if len(filename) < 7 {
		return 0
	}
	number, _ := strconv.Atoi(filename[4:7])
	return number
}

func loadDecisions(directory string) ([]*DecisionRecord, []string, error) {
	files, err := listDecisionFiles(directory)
	if err != nil {
		return nil, nil, err
	}
	if len(files) > maxDecisions {
		files = files[:maxDecisions]
	}
	decisions := make([]*DecisionRecord, 0, len(files))
	invalid := []string{}
	for _, filename := range files {
		path := filepath.Join(directory, filename)
		record, err := loadDecision(path)
		if err != nil {
			invalid = append(invalid, filepath.ToSlash(path))
			continue
		}
		decisions = append(decisions, record)
	}
	return decisions, invalid, nil
}

func buildDecisionIndex(directory string) (*DecisionIndex, []string, error) {
	decisions, invalid, err := loadDecisions(directory)
	if err != nil {
		return nil, nil, err
	}
	entries := make([]DecisionIndexEntry, 0, len(decisions))
	for _, record := range decisions {
		decision := record.Artifact.Decision
		entries = append(entries, DecisionIndexEntry{
			ID: decision.ID, Title: decision.Title, Status: decision.Status, Date: decision.Date,
			File: filepath.ToSlash(record.Filename), Options: decision.ConsideredOptionIDs,
		})
	}
	return &DecisionIndex{SchemaVersion: SchemaVersion, GeneratedAt: time.Now().UTC(), DecisionCount: len(entries), Decisions: entries}, invalid, nil
}

func writeDecisionIndex(directory string) (*DecisionIndex, error) {
	index, invalid, err := buildDecisionIndex(directory)
	if err != nil {
		return nil, err
	}
	if len(invalid) > 0 {
		return nil, fmt.Errorf("cannot index invalid decision files: %s", strings.Join(invalid, ", "))
	}
	data, err := json.MarshalIndent(index, "", "  ")
	if err != nil {
		return nil, err
	}
	data = append(data, '\n')
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return nil, err
	}
	if err := os.WriteFile(filepath.Join(directory, "index.json"), data, 0o644); err != nil {
		return nil, err
	}
	return index, nil
}

func nextDecisionID(directory string) (string, int, error) {
	files, err := listDecisionFiles(directory)
	if err != nil {
		return "", 0, err
	}
	max := 0
	for _, file := range files {
		if number := decisionNumber(file); number > max {
			max = number
		}
	}
	number := max + 1
	if number > 999 {
		return "", 0, fmt.Errorf("decision id space exhausted at ADR-999")
	}
	return fmt.Sprintf("ADR-%03d", number), number, nil
}

func slugify(value string) string {
	value = strings.ToLower(value)
	var builder strings.Builder
	lastHyphen := false
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			builder.WriteRune(r)
			lastHyphen = false
		} else if !lastHyphen && builder.Len() > 0 {
			builder.WriteByte('-')
			lastHyphen = true
		}
	}
	return strings.Trim(builder.String(), "-")
}
