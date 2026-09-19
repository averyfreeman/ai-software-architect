package architect

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"time"
)

// Questions returns the stable setup questionnaire consumed by host agents.
func Questions() []QuestionSpec {
	return append([]QuestionSpec(nil), questionSpecs()...)
}

// SetupProject creates only the files selected by the submitted setup answers.
// Existing files are preserved unless force is true.
func SetupProject(dir string, answers SetupAnswer, force bool) (SetupResult, error) {
	root, err := projectRoot(dir)
	if err != nil {
		return SetupResult{}, err
	}
	return setupProject(root, answers, force)
}

func LoadScaffoldConfig(dir string) (ScaffoldConfig, error) {
	root, err := projectRoot(dir)
	if err != nil {
		return ScaffoldConfig{}, err
	}
	data, err := os.ReadFile(filepath.Join(root, ConfigFilename))
	if err != nil {
		return ScaffoldConfig{}, err
	}
	var config ScaffoldConfig
	if err := unmarshalSafe(data, &config); err != nil {
		return ScaffoldConfig{}, fmt.Errorf("parse %s: %w", ConfigFilename, err)
	}
	return config, nil
}

func LoadGitHabits(dir string) (GitHabitsConfig, error) {
	root, err := projectRoot(dir)
	if err != nil {
		return GitHabitsConfig{}, err
	}
	data, err := os.ReadFile(filepath.Join(root, GitHabitsFilename))
	if err != nil {
		return GitHabitsConfig{}, err
	}
	var config GitHabitsConfig
	if err := unmarshalSafe(data, &config); err != nil {
		return GitHabitsConfig{}, fmt.Errorf("parse %s: %w", GitHabitsFilename, err)
	}
	return config, nil
}

func LoadProjectContext(dir string) (ProjectContext, error) {
	root, err := projectRoot(dir)
	if err != nil {
		return ProjectContext{}, err
	}
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(ContextFilename)))
	if err != nil {
		return ProjectContext{}, err
	}
	var context ProjectContext
	if _, err := frontmatter(string(data), &context); err != nil {
		return ProjectContext{}, fmt.Errorf("parse %s: %w", ContextFilename, err)
	}
	return context, nil
}

func ValidateContract(content string) (ArchitectureContract, ValidationResult) {
	return validateContractYAML(content)
}

func ScanArtifact(content, kind string) SecretScanResult {
	return scanSecrets(content)
}

func ValidateArtifact(content, kind string) ValidationResult {
	return validateArtifact(content, kind)
}

func AnalyzeDependencies(dir string, roots, languages []string) (DependencyResult, error) {
	root, err := projectRoot(dir)
	if err != nil {
		return DependencyResult{}, err
	}
	files, skipped, err := collectSourceFiles(root, roots, languageExtensions(languages))
	if err != nil {
		return DependencyResult{}, err
	}
	truncated := len(files) >= maxWorkspaceFiles
	return analyzeDependencies(files, languages, skipped, truncated), nil
}

func AnalyzeInlineDependencies(files []SourceFile, languages []string) (DependencyResult, error) {
	extensions := languageExtensions(languages)
	seen := map[string]struct{}{}
	totalBytes := 0
	converted := make([]sourceFile, len(files))
	for index, file := range files {
		relative, err := relativeSafe(".", file.RelativePath)
		if err != nil {
			return DependencyResult{}, err
		}
		for _, part := range strings.Split(relative, "/") {
			if strings.HasPrefix(part, ".") {
				return DependencyResult{}, fmt.Errorf("hidden inline source paths are not accepted: %s", file.RelativePath)
			}
		}
		if _, exists := seen[relative]; exists {
			return DependencyResult{}, fmt.Errorf("duplicate inline source path: %s", relative)
		}
		seen[relative] = struct{}{}
		if _, ok := extensions[strings.ToLower(filepath.Ext(relative))]; !ok {
			return DependencyResult{}, fmt.Errorf("unsupported inline source suffix: %s", relative)
		}
		if len(file.Content) > maxSourceBytes {
			return DependencyResult{}, fmt.Errorf("inline source exceeds the single-file budget: %s", relative)
		}
		totalBytes += len(file.Content)
		if totalBytes > maxWorkspaceBytes {
			return DependencyResult{}, fmt.Errorf("inline source exceeds the total workspace budget")
		}
		converted[index] = sourceFile{RelativePath: relative, Content: file.Content}
	}
	return analyzeDependencies(converted, languages, 0, false), nil
}

// SourceFile is the bounded source representation accepted by the MCP adapter.
type SourceFile struct {
	RelativePath string `json:"relative_path"`
	Content      string `json:"content"`
}

type DecisionCheck struct {
	File   string           `json:"file"`
	Result ValidationResult `json:"result"`
	Error  string           `json:"error,omitempty"`
}

func NewDecision(dir, title, status string) (DecisionRecord, error) {
	root, err := projectRoot(dir)
	if err != nil {
		return DecisionRecord{}, err
	}
	if strings.TrimSpace(title) == "" {
		return DecisionRecord{}, fmt.Errorf("decision title is required")
	}
	if !validStatus(status) {
		return DecisionRecord{}, fmt.Errorf("invalid status %q", status)
	}
	context, err := LoadProjectContext(root)
	if err != nil {
		return DecisionRecord{}, fmt.Errorf("load project context before creating a decision: %w", err)
	}
	decisionDirectory := filepath.Join(root, filepath.FromSlash(DecisionsDir))
	id, _, err := nextDecisionID(decisionDirectory)
	if err != nil {
		return DecisionRecord{}, err
	}
	today := time.Now().UTC().Format("2006-01-02")
	preferred := "OPT-001"
	decision := Decision{
		ID: id, Title: strings.TrimSpace(title), Date: today, Status: status, Motivation: context.Motivation,
		Context:             "Replace with the concrete context and forces that made this decision necessary.",
		Drivers:             []string{"Replace with a measurable decision driver."},
		ConsideredOptionIDs: []string{"OPT-001", "OPT-002"}, SelectedOptionID: nil,
		Decision:             "Replace with the proposed decision and its boundary.",
		PositiveConsequences: []string{"Replace with a good outcome."}, NegativeConsequences: []string{"Replace with a bad outcome and mitigation."},
		Assumptions: []string{"Replace with a bounded assumption."}, ValidationCriteria: []string{"Replace with an observable validation criterion."}, Supersedes: []string{},
		DecisionMatrix: []DecisionOption{{ID: "OPT-001", Name: "Preferred option", FitScore: 0, Benefits: []string{"Replace"}, Drawbacks: []string{"Replace"}, Risks: []string{"Replace"}, Evidence: []string{"Replace"}, Outcome: "proposed"}, {ID: "OPT-002", Name: "Alternative option", FitScore: 0, Benefits: []string{"Replace"}, Drawbacks: []string{"Replace"}, Risks: []string{"Replace"}, Evidence: []string{"Replace"}, Outcome: "rejected"}},
		ActionContract: ActionContract{GoodOutcomes: []string{"Describe what goes right when the selected option is followed."}, BadOutcomes: []string{"Describe what goes wrong when the boundary is bypassed."}, CorrectExample: "Describe one compliant action.", IncorrectExample: "Describe one non-compliant action and its outcome."},
	}
	if status == "accepted" {
		decision.SelectedOptionID = &preferred
	}
	artifact := DecisionArtifact{SchemaVersion: SchemaVersion, Revision: 1, Decision: decision}
	filename := id + "-" + slugify(title) + ".md"
	body := renderDecisionBody(artifact)
	content := "---\n" + mustYAML(artifact) + "---\n\n" + body
	validation := validateDecision(&DecisionRecord{Artifact: artifact, Body: body, Filename: filename})
	if !validation.Valid {
		return DecisionRecord{}, fmt.Errorf("generated decision is invalid: %s", validation.Errors[0].Message)
	}
	if err := os.MkdirAll(decisionDirectory, 0o755); err != nil {
		return DecisionRecord{}, err
	}
	path := filepath.Join(decisionDirectory, filename)
	if _, err := os.Stat(path); err == nil {
		return DecisionRecord{}, fmt.Errorf("decision file already exists: %s", path)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return DecisionRecord{}, err
	}
	if _, err := writeDecisionIndex(decisionDirectory); err != nil {
		return DecisionRecord{}, fmt.Errorf("write decision index: %w", err)
	}
	return DecisionRecord{Artifact: artifact, Body: body, Filename: filename, Path: path}, nil
}

func renderDecisionBody(artifact DecisionArtifact) string {
	d := artifact.Decision
	selected := "Proposed"
	if d.SelectedOptionID != nil {
		selected = "Accepted"
	}
	matrix := make([]string, 0, len(d.DecisionMatrix))
	for _, option := range d.DecisionMatrix {
		matrix = append(matrix, fmt.Sprintf("| %s — %s | %d | %s | %s | %s | %s | %s |", option.ID, option.Name, option.FitScore, strings.Join(option.Benefits, "; "), strings.Join(option.Drawbacks, "; "), strings.Join(option.Risks, "; "), strings.Join(option.Evidence, "; "), option.Outcome))
	}
	return fmt.Sprintf(`# %s: %s

## Context

%s

## Decision Drivers

%s

## Decision Matrix

| Option | Fit | Benefits | Drawbacks | Risks | Evidence | Outcome |
| --- | ---: | --- | --- | --- | --- | --- |
%s

## Decision

### Selected option: %s

%s

**Accepted because:** Tie the selected option to the decision drivers and evidence.

**Accepted despite:** Name the downside deliberately accepted.

## Alternatives Considered

The matrix above compares every credible option. Explain why each non-selected option was rejected and what legitimate strength it had.

**Rejected because:** The alternatives do not satisfy the highest-priority drivers as well.

**Rejected despite:** Each alternative's strongest benefit was considered.

## Consequences

### Good outcomes

%s

### Bad outcomes and mitigations

%s

## Agent Action Contract

- Correct example: %s
- Incorrect example: %s

## Confirmation

- Validation criterion: %s
- Motivation source: %s
`, d.ID, d.Title, d.Context, bulletList(d.Drivers), strings.Join(matrix, "\n"), selected, d.Decision, bulletList(d.PositiveConsequences), bulletList(d.NegativeConsequences), d.ActionContract.CorrectExample, d.ActionContract.IncorrectExample, strings.Join(d.ValidationCriteria, "; "), d.Motivation.Source)
}

func bulletList(values []string) string {
	if len(values) == 0 {
		return "- Replace with an evidence-backed statement."
	}
	result := make([]string, 0, len(values))
	for _, value := range values {
		result = append(result, "- "+value)
	}
	return strings.Join(result, "\n")
}

func IndexDecisions(dir string) (DecisionIndex, error) {
	root, err := projectRoot(dir)
	if err != nil {
		return DecisionIndex{}, err
	}
	index, invalid, err := buildDecisionIndex(filepath.Join(root, filepath.FromSlash(DecisionsDir)))
	if err != nil {
		return DecisionIndex{}, err
	}
	if len(invalid) > 0 {
		return DecisionIndex{}, fmt.Errorf("invalid decision files: %s", strings.Join(invalid, ", "))
	}
	if _, err := writeDecisionIndex(filepath.Join(root, filepath.FromSlash(DecisionsDir))); err != nil {
		return DecisionIndex{}, err
	}
	return *index, nil
}

func CheckDecisions(dir string) ([]DecisionCheck, error) {
	root, err := projectRoot(dir)
	if err != nil {
		return nil, err
	}
	directory := filepath.Join(root, filepath.FromSlash(DecisionsDir))
	files, err := listDecisionFiles(directory)
	if err != nil {
		return nil, err
	}
	checks := make([]DecisionCheck, 0, len(files))
	for _, filename := range files {
		path := filepath.Join(directory, filename)
		record, err := loadDecision(path)
		if err != nil {
			checks = append(checks, DecisionCheck{File: filepath.ToSlash(path), Result: ValidationResult{Valid: false}, Error: err.Error()})
			continue
		}
		checks = append(checks, DecisionCheck{File: filepath.ToSlash(path), Result: validateDecision(record)})
	}
	return checks, nil
}

func ListDecisions(dir, status string) ([]DecisionRecord, []string, error) {
	root, err := projectRoot(dir)
	if err != nil {
		return nil, nil, err
	}
	decisions, invalid, err := loadDecisions(filepath.Join(root, filepath.FromSlash(DecisionsDir)))
	if err != nil {
		return nil, nil, err
	}
	result := make([]DecisionRecord, 0, len(decisions))
	for _, record := range decisions {
		if status == "" || strings.EqualFold(record.Artifact.Decision.Status, status) {
			result = append(result, *record)
		}
	}
	return result, invalid, nil
}

func FindDecision(dir, identifier string) (DecisionRecord, error) {
	decisions, _, err := ListDecisions(dir, "")
	if err != nil {
		return DecisionRecord{}, err
	}
	needle := strings.ToUpper(strings.TrimSpace(identifier))
	for _, record := range decisions {
		if strings.ToUpper(record.Artifact.Decision.ID) == needle || strings.EqualFold(record.Filename, identifier) || strings.HasPrefix(strings.ToUpper(record.Filename), needle+"-") {
			return record, nil
		}
	}
	return DecisionRecord{}, fmt.Errorf("decision %q was not found", identifier)
}

func IndexMatches(dir string) (bool, error) {
	root, err := projectRoot(dir)
	if err != nil {
		return false, err
	}
	directory := filepath.Join(root, filepath.FromSlash(DecisionsDir))
	path := filepath.Join(directory, "index.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	var existing DecisionIndex
	if err := jsonUnmarshal(data, &existing); err != nil {
		return false, err
	}
	fresh, invalid, err := buildDecisionIndex(directory)
	if err != nil {
		return false, err
	}
	if len(invalid) > 0 {
		return false, fmt.Errorf("invalid decision files: %s", strings.Join(invalid, ", "))
	}
	existing.GeneratedAt = time.Time{}
	fresh.GeneratedAt = time.Time{}
	return deepEqualJSON(existing, *fresh), nil
}

// These small indirections keep the public API free of serialization policy.
func jsonUnmarshal(data []byte, target any) error { return json.Unmarshal(data, target) }
func deepEqualJSON(a, b any) bool                 { return reflect.DeepEqual(a, b) }

func sortedStrings(values []string) []string {
	result := append([]string(nil), values...)
	sort.Strings(result)
	return result
}
