package architect

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func validContractYAML() string {
	return strings.Join([]string{
		"schema_version: 1.1.0",
		"revision: 1",
		"scope: sample",
		"components:",
		"  - id: app",
		"    responsibility: application boundary",
		"    owns_data: []",
		"    public_interfaces: []",
		"decision_ids: [ADR-001]",
	}, "\n") + "\n"
}

func validADR() string {
	return strings.Join([]string{
		"---",
		"schema_version: 1.1.0",
		"revision: 1",
		"decision:",
		"  id: ADR-001",
		"  title: Choose a boundary",
		"  status: accepted",
		"  context: A boundary is required for testability.",
		"  drivers: [Testability]",
		"  considered_option_ids: [OPT-001]",
		"  selected_option_id: OPT-001",
		"  decision: Keep one explicit application boundary.",
		"  positive_consequences: [Simple tests]",
		"  negative_consequences: []",
		"  assumptions: []",
		"  validation_criteria: [Boundary tests pass]",
		"  supersedes: []",
		"---",
		"# Decision",
		"",
	}, "\n")
}

func TestContractAcceptsOnlyCurrentSchema(t *testing.T) {
	result, contract := ValidateContractYAML(validContractYAML())
	if !result.Valid || contract == nil || result.SchemaVersion != CurrentSchemaVersion {
		t.Fatalf("current contract rejected: %#v", result)
	}
	wrongVersion := strings.Replace(validContractYAML(), "1.1.0", "1.1.1", 1)
	wrongResult, _ := ValidateContractYAML(wrongVersion)
	if wrongResult.Valid {
		t.Fatal("non-current schema version was accepted")
	}
}

func TestStrictYAMLRejectsAliasesDuplicatesAndUnknownFields(t *testing.T) {
	duplicate := strings.Replace(validContractYAML(), "scope: sample", "scope: sample\nscope: duplicate", 1)
	if result, _ := ValidateContractYAML(duplicate); result.Valid || len(result.Errors) == 0 {
		t.Fatalf("duplicate key was accepted: %#v", result)
	}
	alias := strings.Replace(validContractYAML(), "scope: sample", "scope: &scope sample\narchitecture_style: *scope", 1)
	if result, _ := ValidateContractYAML(alias); result.Valid || len(result.Errors) == 0 {
		t.Fatalf("alias was accepted: %#v", result)
	}
	unknown := validContractYAML() + "unexpected: true\n"
	if result, _ := ValidateContractYAML(unknown); result.Valid || len(result.Errors) == 0 {
		t.Fatalf("unknown field was accepted: %#v", result)
	}
}

func TestBundleValidationRequiresMatchingAcceptedADR(t *testing.T) {
	candidates := []ArtifactCandidate{
		{Path: ".ai-architect/architecture-contract.yaml", Content: validContractYAML(), Kind: ArtifactContract, Operation: "Add"},
		{Path: ".ai-architect/decisions/ADR-001-boundary.md", Content: validADR(), Kind: ArtifactADR, Operation: "Add"},
		{Path: ".ai-architect/project-context.md", Content: "# Context\n", Kind: ArtifactContext, Operation: "Add"},
		{Path: ".ai-architect/implementation-plan.md", Content: "# Handoff\n", Kind: ArtifactImplementationPlan, Operation: "Add"},
	}
	bundle, err := ValidateArtifactBundleCandidates(candidates, true)
	if err != nil || bundle == nil || bundle.Contract.DecisionIDs[0] != "ADR-001" {
		t.Fatalf("valid bundle rejected: bundle=%#v err=%v", bundle, err)
	}
	candidates[1].Content = strings.Replace(validADR(), "ADR-001", "ADR-002", 1)
	if _, err := ValidateArtifactBundleCandidates(candidates, true); err == nil {
		t.Fatal("mismatched ADR was accepted")
	}
}

func TestArtifactPathsAndSecretScanning(t *testing.T) {
	root := t.TempDir()
	if !IsCanonicalArtifactPath(".ai-architect/architecture-contract.yaml", root) {
		t.Fatal("canonical contract path rejected")
	}
	if !IsCanonicalArtifactPath(`.ai-architect\architecture-contract.yaml`, root) {
		t.Fatal("Windows-style canonical contract path rejected")
	}
	if IsCanonicalArtifactPath("../architecture-contract.yaml", root) {
		t.Fatal("traversal artifact path accepted")
	}
	if IsCanonicalArtifactPath(filepath.Join(root, "application.go"), root) {
		t.Fatal("application source path accepted")
	}
	safe := ScanGeneratedArtifact("password: " + "$" + "{PASSWORD}\n")
	if !safe.SafeToWrite {
		t.Fatalf("placeholder was marked secret: %#v", safe)
	}
	unsafe := ScanGeneratedArtifact("token: sk-123456789012345678901234\n")
	if unsafe.SafeToWrite || len(unsafe.Findings) == 0 {
		t.Fatalf("secret was not detected: %#v", unsafe)
	}
}

func TestWorkspaceAndDependencyBoundaries(t *testing.T) {
	root := t.TempDir()
	if _, err := ValidateInlinePythonPath("../outside.py"); err == nil {
		t.Fatal("path traversal accepted")
	}
	if !IsProtectedPath(".env.local") || !IsProtectedPath(".ssh/id_rsa") {
		t.Fatal("protected path not recognized")
	}
	if err := os.WriteFile(filepath.Join(root, "app.py"), []byte("import package\nfrom local.module import value\nimportlib.import_module(name)\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	reader, err := NewWorkspaceReader(root)
	if err != nil {
		t.Fatal(err)
	}
	graph, err := AnalyzeRepositoryDependencies(reader, RepositoryAnalysisInput{})
	if err != nil || len(graph.Edges) != 2 || len(graph.Warnings) == 0 {
		t.Fatalf("dependency analysis mismatch: %#v err=%v", graph, err)
	}
}

func TestInlineAndMultilineDependencyEvidencePreservesBoundaries(t *testing.T) {
	sources := []SourceFileInput{
		{RelativePath: "domain/service.py", Content: "import json\nfrom adapter import vendor\n"},
	}
	graph, err := AnalyzeRepositoryDependencies(nil, RepositoryAnalysisInput{SourceFiles: sources})
	if err != nil {
		t.Fatal(err)
	}
	if graph.FilesExamined != 1 || len(graph.Edges) != 2 || len(graph.Warnings) != 1 {
		t.Fatalf("inline dependency analysis mismatch: %#v", graph)
	}
	if graph.Edges[1].Evidence != "domain/service.py:2" {
		t.Fatalf("inline evidence line changed: %#v", graph.Edges[1])
	}

	statements := []DependencyStatementInput{
		{RelativePath: "budget_book.py", StartLine: 12, Statement: "import pandas as pd"},
		{RelativePath: "budget_book.py", StartLine: 20, Statement: "from matplotlib import (\n    pyplot,\n)"},
	}
	graph, err = AnalyzeDependencyStatements(statements)
	if err != nil {
		t.Fatal(err)
	}
	if graph.FilesExamined != 1 || len(graph.Edges) != 2 || graph.Edges[1].Evidence != "budget_book.py:20" {
		t.Fatalf("multiline dependency analysis mismatch: %#v", graph)
	}
}

func TestSnapshotIsBoundedASCIIJSON(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("café\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	snapshot, err := CreateRepositorySnapshot(root)
	if err != nil {
		t.Fatal(err)
	}
	data, err := SerializeRepositorySnapshot(snapshot)
	if err != nil || len(data) > MaxSnapshotSerializedBytes || strings.Contains(string(data), "é") {
		t.Fatalf("snapshot serialization mismatch: err=%v size=%d", err, len(data))
	}
}
