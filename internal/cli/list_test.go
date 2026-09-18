package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunListReadsCurrentADRFiles(t *testing.T) {
	dir := t.TempDir()
	output := &Output{Format: FormatText, Writer: &bytes.Buffer{}}

	created, err := RunNew(&NewConfig{
		Title:  "Original title",
		Dir:    dir,
		Tags:   []string{"database", "storage"},
		Paths:  []string{"src/db/**", "src/api/**"},
		Status: "proposed",
		Output: output,
	})
	if err != nil {
		t.Fatalf("RunNew() error = %v", err)
	}

	filePath := filepath.Join(dir, created.File)
	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("reading created ADR: %v", err)
	}
	updated := strings.Replace(string(content), "title: Original title", "title: Updated title", 1)
	if updated == string(content) {
		t.Fatalf("created ADR did not contain expected title frontmatter")
	}
	if err := os.WriteFile(filePath, []byte(updated), 0644); err != nil {
		t.Fatalf("updating ADR: %v", err)
	}

	result, err := RunList(&ListConfig{Dir: dir, Output: output})
	if err != nil {
		t.Fatalf("RunList() error = %v", err)
	}
	if result.Count != 1 || result.ADRs[0].Title != "Updated title" {
		t.Errorf("RunList() = %#v, want one updated ADR", result)
	}

	filtered, err := RunList(&ListConfig{
		Dir:    dir,
		Status: "proposed",
		Tags:   []string{"missing", "storage"},
		Paths:  []string{"missing/**", "src/db/users.go"},
		Output: output,
	})
	if err != nil {
		t.Fatalf("RunList(filtered) error = %v", err)
	}
	if filtered.Count != 1 || filtered.ADRs[0].ADRID != created.ADRID {
		t.Errorf("RunList(filtered) = %#v, want matching ADR", filtered)
	}
}
