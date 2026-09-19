package gitbbq

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAssessMigrationIsReadOnlyAndMapsLegacyFiles(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, ".adr-scaffold.yaml"), []byte("version: 1\nlanguage: go\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, ".ai-architect", "decisions"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".ai-architect", "project-context.md"), []byte("---\nlanguage: go\nmotivation:\n  problem: Keep decisions durable.\n---\n\n# Context\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	before, err := SnapshotFiles(root)
	if err != nil {
		t.Fatal(err)
	}
	assessment, err := AssessMigration(root)
	if err != nil {
		t.Fatal(err)
	}
	if !assessment.ReadOnly || !assessment.Detected {
		t.Fatalf("assessment = %#v", assessment)
	}
	if !containsPath(assessment.Legacy, ".adr-scaffold.yaml") || !containsPath(assessment.Legacy, ".ai-architect/project-context.md") {
		t.Fatalf("legacy paths = %#v", assessment.Legacy)
	}
	if !containsPath(assessment.Proposed, ManifestFilename) || !containsPath(assessment.Proposed, ContextFilename) {
		t.Fatalf("proposed paths = %#v", assessment.Proposed)
	}
	if !strings.Contains(strings.Join(assessment.Warnings, "\n"), "approval") {
		t.Fatalf("warnings = %#v", assessment.Warnings)
	}
	after, err := SnapshotFiles(root)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(before, "\n") != strings.Join(after, "\n") {
		t.Fatal("migration assessment changed the repository")
	}
}

func TestAssessMigrationReportsMissingLegacyScaffold(t *testing.T) {
	assessment, err := AssessMigration(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if assessment.Detected || len(assessment.Legacy) != 0 {
		t.Fatalf("unexpected legacy detection: %#v", assessment)
	}
}

func containsPath(paths []string, expected string) bool {
	for _, path := range paths {
		if path == expected {
			return true
		}
	}
	return false
}
