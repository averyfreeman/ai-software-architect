package architect

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func testAnswers() SetupAnswer {
	return SetupAnswer{
		Language:        "go",
		WhyBuilding:     "Provide a repeatable architecture record for agents.",
		Problem:         "Important decisions otherwise disappear into chat history.",
		Audience:        "Agents and maintainers of the project.",
		Alternatives:    "Ad hoc Markdown, tickets, and an external wiki.",
		WhyThisSolution: "A local structured record is available beside the code and can be validated.",
		AutoReason:      "no",
		Options: map[string]string{
			"include_pages": "make-default",
			"git_init":      "make-default",
			"create_remote": "no",
		},
	}
}

func TestSetupSeparatesConfigsAndPreservesExistingFiles(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "AGENTS.md"), []byte("user rules\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	result, err := SetupProject(root, testAnswers(), false)
	if err != nil {
		t.Fatal(err)
	}
	if result.Language != "go" {
		t.Fatalf("language = %q", result.Language)
	}
	agents, err := os.ReadFile(filepath.Join(root, "AGENTS.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(agents) != "user rules\n" {
		t.Fatal("existing AGENTS.md was overwritten")
	}
	scaffold, err := os.ReadFile(filepath.Join(root, ConfigFilename))
	if err != nil {
		t.Fatal(err)
	}
	git, err := os.ReadFile(filepath.Join(root, GitHabitsFilename))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(scaffold), "create_remote:") || strings.Contains(string(scaffold), "git_init:") {
		t.Fatal("Git mutation keys leaked into scaffold config")
	}
	if strings.Contains(string(git), "agents:") || strings.Contains(string(git), "implementation_plan:") {
		t.Fatal("scaffold keys leaked into Git habits config")
	}
	if _, err := os.Stat(filepath.Join(root, ".github/workflows/pages.yml")); err != nil {
		t.Fatal(err)
	}
	if len(result.Skipped) != 1 || result.Skipped[0] != "AGENTS.md" {
		t.Fatalf("skipped = %#v", result.Skipped)
	}
	second, err := SetupProject(root, SetupAnswer{Language: "go", WhyBuilding: testAnswers().WhyBuilding, Problem: testAnswers().Problem, Audience: testAnswers().Audience, Alternatives: testAnswers().Alternatives, WhyThisSolution: testAnswers().WhyThisSolution}, false)
	if err != nil {
		t.Fatal(err)
	}
	if !second.Scaffold.Features.Pages {
		t.Fatal("make-default did not persist the Pages preference")
	}
}

func TestSetupPersistsAutoReasonDefault(t *testing.T) {
	root := t.TempDir()
	answers := testAnswers()
	answers.AutoReason = "make-default"
	answers.WhyBuilding = ""
	answers.Problem = ""
	answers.Audience = ""
	answers.Alternatives = ""
	answers.WhyThisSolution = ""
	if _, err := SetupProject(root, answers, false); err != nil {
		t.Fatal(err)
	}
	second := SetupAnswer{Language: "go"}
	result, err := SetupProject(root, second, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Warnings) == 0 || !strings.Contains(result.Warnings[0], "auto-reasoned motivation") {
		t.Fatalf("warnings = %#v", result.Warnings)
	}
}

func TestNewDecisionUsesProjectMotivationAndIndexesIt(t *testing.T) {
	root := t.TempDir()
	if _, err := SetupProject(root, testAnswers(), false); err != nil {
		t.Fatal(err)
	}
	record, err := NewDecision(root, "Choose a stable tool boundary", "proposed")
	if err != nil {
		t.Fatal(err)
	}
	if record.Artifact.Decision.ID != "ADR-001" {
		t.Fatalf("id = %q", record.Artifact.Decision.ID)
	}
	checks, err := CheckDecisions(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(checks) != 1 || !checks[0].Result.Valid {
		t.Fatalf("checks = %#v", checks)
	}
	if ok, err := IndexMatches(root); err != nil || !ok {
		t.Fatalf("index freshness = %v, err = %v", ok, err)
	}
	index, err := IndexDecisions(root)
	if err != nil {
		t.Fatal(err)
	}
	if index.DecisionCount != 1 || index.Decisions[0].Date == "" {
		t.Fatalf("index = %#v", index)
	}
}

func TestValidateProjectBundleRequiresAcceptedExactDecisionSet(t *testing.T) {
	root := t.TempDir()
	if _, err := SetupProject(root, testAnswers(), false); err != nil {
		t.Fatal(err)
	}
	if _, err := NewDecision(root, "Choose a stable tool boundary", "accepted"); err != nil {
		t.Fatal(err)
	}
	contract := ArchitectureContract{
		SchemaVersion: SchemaVersion,
		Revision:      1,
		Scope:         "sample",
		DecisionIDs:   []string{"ADR-001"},
	}
	contractData, err := marshalYAML(contract)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ContractFilename), contractData, 0o644); err != nil {
		t.Fatal(err)
	}
	bundle, result, err := ValidateProjectBundle(root)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Valid || len(bundle.Decisions) != 1 {
		t.Fatalf("valid bundle = %v, result = %#v, bundle = %#v", result.Valid, result, bundle)
	}

	contract.DecisionIDs = nil
	contractData, err = marshalYAML(contract)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ContractFilename), contractData, 0o644); err != nil {
		t.Fatal(err)
	}
	_, result, err = ValidateProjectBundle(root)
	if err != nil {
		t.Fatal(err)
	}
	if result.Valid || !hasIssueCode(result.Errors, "decision_reference_mismatch") {
		t.Fatalf("mismatched bundle was accepted: %#v", result)
	}
}

func TestValidateArtifactBundleRejectsSecretsWithoutReturningValues(t *testing.T) {
	_, result := ValidateArtifactBundle(ArtifactBundleInput{
		ContractYAML:   "schema_version: 1.0.0\nrevision: 1\nscope: sample\n",
		Decisions:      []BundleDecisionInput{{Filename: "ADR-001-choice.md", Content: "---\nschema_version: 1.0.0\nrevision: 1\ndecision:\n  id: ADR-001\n  title: Choice\n  status: accepted\n  date: 2026-01-01\n  motivation:\n    why_building: Build it\n    problem: Solve it\n    audience: Agents\n    alternatives: Other\n    why_this_solution: Better\n  context: Context\n  drivers: [Speed]\n  considered_option_ids: [OPT-001, OPT-002]\n  selected_option_id: OPT-001\n  decision: Choose it\n  validation_criteria: [Test it]\n  decision_matrix:\n    - id: OPT-001\n      name: One\n      fit_score: 80\n      benefits: [Fast]\n      drawbacks: [Cost]\n      risks: [Risk]\n      evidence: [Test]\n      outcome: accepted\n    - id: OPT-002\n      name: Two\n      fit_score: 40\n      benefits: [Simple]\n      drawbacks: [Limited]\n      risks: [Risk]\n      evidence: [Test]\n      outcome: rejected\n  action_contract:\n    good_outcomes: [Works]\n    bad_outcomes: [Fails]\n    correct_example: Use it\n    incorrect_example: Bypass it\n---\n# Decision\n\napi_key = really-long-secret-value\n"}},
		ProjectContext: "Project context",
		CodingHandoff:  "Coding handoff",
	})
	if result.Valid || !hasIssueCode(result.Errors, "credential") {
		t.Fatalf("secret-containing bundle was accepted: %#v", result)
	}
	if strings.Contains(strings.Join(issueMessages(result.Errors), "\n"), "really-long-secret-value") {
		t.Fatal("secret value was returned in validation diagnostics")
	}
}

func TestValidateProjectBundleRejectsSymlinkedDecisionDirectory(t *testing.T) {
	root := t.TempDir()
	if _, err := SetupProject(root, testAnswers(), false); err != nil {
		t.Fatal(err)
	}
	record, err := NewDecision(root, "Choose a stable tool boundary", "accepted")
	if err != nil {
		t.Fatal(err)
	}
	contractData, err := marshalYAML(ArchitectureContract{
		SchemaVersion: SchemaVersion,
		Revision:      1,
		Scope:         "sample",
		DecisionIDs:   []string{record.Artifact.Decision.ID},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ContractFilename), contractData, 0o644); err != nil {
		t.Fatal(err)
	}
	decisionDirectory := filepath.Join(root, DecisionsDir)
	externalDirectory := filepath.Join(t.TempDir(), "decisions")
	if err := os.MkdirAll(externalDirectory, 0o755); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(record.Path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(externalDirectory, record.Filename), content, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(decisionDirectory); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(externalDirectory, decisionDirectory); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if _, _, err := ValidateProjectBundle(root); err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("symlinked decision directory result = %v", err)
	}
}

func hasIssueCode(issues []ValidationIssue, code string) bool {
	for _, issue := range issues {
		if issue.Code == code {
			return true
		}
	}
	return false
}

func issueMessages(issues []ValidationIssue) []string {
	messages := make([]string, 0, len(issues))
	for _, issue := range issues {
		messages = append(messages, issue.Message)
	}
	return messages
}

func TestDecisionValidationRequiresAllMotivationFields(t *testing.T) {
	record := &DecisionRecord{Filename: "ADR-001-example.md", Artifact: DecisionArtifact{SchemaVersion: SchemaVersion, Revision: 1, Decision: Decision{
		ID: "ADR-001", Title: "Example", Status: "proposed", Motivation: Motivation{WhyBuilding: "only one answer"}, Context: "Context", Drivers: []string{"Driver"}, ConsideredOptionIDs: []string{"OPT-001", "OPT-002"}, Decision: "Decision", ValidationCriteria: []string{"Check"},
	}}, Body: ""}
	result := validateDecision(record)
	if result.Valid {
		t.Fatal("decision with incomplete motivation was accepted")
	}
	missing := 0
	for _, issue := range result.Errors {
		if issue.Code == "missing_motivation" {
			missing++
		}
	}
	if missing != 4 {
		t.Fatalf("missing motivation errors = %d", missing)
	}
}

func TestSafeYAMLRejectsAliasesAndDuplicateKeys(t *testing.T) {
	if _, result := ValidateContract("schema_version: 1.0.0\nrevision: 1\nscope: x\nschema_version: 1.0.0\n"); result.Valid || len(result.Errors) == 0 {
		t.Fatal("duplicate YAML key was accepted")
	}
	if _, result := ValidateContract("base: &base\n  scope: x\nschema_version: 1.0.0\nrevision: 1\n<<: *base\n"); result.Valid || len(result.Errors) == 0 {
		t.Fatal("YAML alias was accepted")
	}
}

func TestValidateCanonicalContractIncludesNestedQuestionsAndOptionalNulls(t *testing.T) {
	content, err := os.ReadFile(filepath.Join("..", "..", "shared", "skills", "create-architecture-decisions", "assets", "architecture-contract.example.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	contract, result := ValidateContract(string(content))
	if !result.Valid {
		t.Fatalf("canonical contract rejected: %#v", result.Errors)
	}
	if len(contract.QualityAttributes) != 1 || len(contract.UnresolvedQuestions) != 1 {
		t.Fatalf("canonical contract lost nested fields: %#v", contract)
	}
	if contract.ArchitectureStyle == nil || *contract.ArchitectureStyle == "" {
		t.Fatal("architecture_style was not decoded")
	}
	if contract.DependencyRules[0].ViaInterface == nil || *contract.DependencyRules[0].ViaInterface != "deliver-notification" {
		t.Fatal("via_interface was not decoded")
	}
	if contract.UnresolvedQuestions[0].Answer != nil {
		t.Fatal("null answer should remain absent")
	}
}

func TestValidateContractEnforcesPydanticCollectionAndTextBounds(t *testing.T) {
	tooMany := strings.Builder{}
	tooMany.WriteString("schema_version: 1.0.0\nrevision: 1\nscope: sample\nrequired_practices:\n")
	for index := 0; index < 201; index++ {
		tooMany.WriteString("  - practice\n")
	}
	if _, result := ValidateContract(tooMany.String()); result.Valid || !hasIssueCode(result.Errors, "too_many_required_practices") {
		t.Fatalf("too many practices were accepted: %#v", result)
	}

	invalid := "schema_version: 1.0.0\nrevision: 1\nscope: '   '\nquality_attributes:\n  - name: ''\n    priority: 6\n    rationale: ''\n"
	if _, result := ValidateContract(invalid); result.Valid || len(result.Errors) < 3 {
		t.Fatalf("invalid contract fields were accepted: %#v", result)
	}

	duplicate := "schema_version: 1.0.0\nrevision: 1\nscope: sample\nquality_attributes:\n  - name: Security\n    priority: 1\n    rationale: Protect data.\n  - name: security\n    priority: 2\n    rationale: Protect data twice.\ncomponents:\n  - id: domain\n    responsibility: Business rules\ndependency_rules:\n  - source: Domain\n    target: domain\n    policy: deny\n    rationale: Invalid source identifier.\n"
	if _, result := ValidateContract(duplicate); result.Valid || !hasIssueCode(result.Errors, "duplicate_quality_attribute") || !hasIssueCode(result.Errors, "invalid_source") {
		t.Fatalf("duplicate or invalid references were accepted: %#v", result)
	}
}

func TestValidateDecisionKeepsLegacySchemaCompatible(t *testing.T) {
	record := &DecisionRecord{Filename: "ADR-001-legacy.md", Artifact: DecisionArtifact{SchemaVersion: "1.0.0", Revision: 1, Decision: Decision{
		ID: "ADR-001", Title: "Legacy decision", Status: "accepted", Context: "A legacy record.", Drivers: []string{"Simplicity"}, ConsideredOptionIDs: []string{"OPT-001"}, SelectedOptionID: stringPtr("OPT-001"), Decision: "Keep the existing boundary.", ValidationCriteria: []string{"Review passes"},
	}}, Body: "# Context\n\n## Decision Drivers\n\n## Decision Matrix\n\n## Decision\n\n## Alternatives Considered\n\n## Consequences\n\n## Agent Action Contract\n"}
	result := validateDecision(record)
	if !result.Valid {
		t.Fatalf("legacy 1.0.0 decision rejected: %#v", result.Errors)
	}
}

func TestValidateDecisionRequiresCurrentSchemaFields(t *testing.T) {
	record := &DecisionRecord{Filename: "ADR-001-current.md", Artifact: DecisionArtifact{SchemaVersion: "1.1.0", Revision: 1, Decision: Decision{
		ID: "ADR-001", Title: "Current decision", Status: "proposed", Context: "A current record.", Drivers: []string{"Simplicity"}, ConsideredOptionIDs: []string{"OPT-001", "OPT-002"}, Decision: "Choose an option.", ValidationCriteria: []string{"Review passes"},
	}}, Body: "# Context\n\n## Decision Drivers\n\n## Decision\n\n## Alternatives Considered\n\n## Consequences\n\n## Agent Action Contract\n"}
	result := validateDecision(record)
	if result.Valid || !hasIssueCode(result.Errors, "missing_motivation") || !hasIssueCode(result.Errors, "missing_decision_matrix") || !hasIssueCode(result.Errors, "missing_action_contract") {
		t.Fatalf("current 1.1.0 requirements were not enforced: %#v", result)
	}
}

func stringPtr(value string) *string {
	return &value
}

func TestSetupRefusesToOverwriteSymlink(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "AGENTS.md")
	external := filepath.Join(t.TempDir(), "outside.md")
	if err := os.WriteFile(external, []byte("outside\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(external, target); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if _, err := SetupProject(root, testAnswers(), true); err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("symlink overwrite result = %v", err)
	}
	content, err := os.ReadFile(external)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "outside\n" {
		t.Fatal("external symlink target changed")
	}
}

func TestGitPlanRejectsUnsafeRefs(t *testing.T) {
	root := t.TempDir()
	config := defaultGitHabits()
	config.Branch = "--delete-this"
	config.Init = true
	data, err := marshalYAML(config)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, GitHabitsFilename), data, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := PlanGitBootstrap(root); err == nil || !strings.Contains(err.Error(), "safe Git ref") {
		t.Fatalf("unsafe branch result = %v", err)
	}
}

func TestDecisionTemplateHasValidMachineFields(t *testing.T) {
	answers := testAnswers()
	motivation := Motivation{WhyBuilding: answers.WhyBuilding, Problem: answers.Problem, Audience: answers.Audience, Alternatives: answers.Alternatives, WhyThisSolution: answers.WhyThisSolution, Source: "human"}
	content := renderDecisionTemplate(motivation, "go", languageProfiles["go"])
	record, err := parseDecision(content, "ADR-001-template.md", "")
	if err != nil {
		t.Fatal(err)
	}
	result := validateDecision(record)
	if !result.Valid {
		t.Fatalf("template validation errors = %#v", result.Errors)
	}
}

func TestDependencyAnalysisIsStaticAndBounded(t *testing.T) {
	result, err := AnalyzeInlineDependencies([]SourceFile{{RelativePath: "cmd/main.go", Content: "package main\nimport (\n \"fmt\"\n \"example/internal/app\"\n)\n"}}, []string{"go"})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Edges) != 2 || result.Edges[0].Evidence != "cmd/main.go:3" {
		t.Fatalf("dependencies = %#v", result.Edges)
	}
	if _, err := AnalyzeInlineDependencies([]SourceFile{{RelativePath: ".env", Content: "secret"}}, []string{"go"}); err == nil {
		t.Fatal("protected inline path was accepted")
	}
}

func TestSecretScannerDoesNotReturnSecretValues(t *testing.T) {
	result := ScanArtifact("api_key = really-long-secret-value\n", "context")
	if result.SafeToWrite || len(result.Findings) != 1 || result.Findings[0].Category != "credential" {
		t.Fatalf("scan = %#v", result)
	}
}

func TestSecretScannerReportsIndependentFindingCategories(t *testing.T) {
	content := "-----BEGIN PRIVATE KEY----- sk-12345678901234567890 api_key = real-secret-value-1234\n"
	result := ScanArtifact(content, "context")
	if result.SafeToWrite || len(result.Findings) != 3 {
		t.Fatalf("scan = %#v", result)
	}
	if result.Findings[0].Category != "private-key" || result.Findings[1].Category != "token" || result.Findings[2].Category != "credential" {
		t.Fatalf("finding categories = %#v", result.Findings)
	}
	for _, finding := range result.Findings {
		if finding.Line != 1 {
			t.Fatalf("finding line = %#v", result.Findings)
		}
	}
}
