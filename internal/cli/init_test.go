package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/averyfreeman/adr-repo-governance/internal/index"
)

func TestRunInitJSON(t *testing.T) {
	dir := t.TempDir()
	var output bytes.Buffer

	err := RunInit(&InitConfig{
		Dir: dir,
		Output: &Output{
			Format: FormatJSON,
			Writer: &output,
		},
	})
	if err != nil {
		t.Fatalf("RunInit() error = %v", err)
	}

	var result InitResult
	if err := json.Unmarshal(output.Bytes(), &result); err != nil {
		t.Fatalf("init output is not valid JSON: %v\n%s", err, output.String())
	}

	if result.ADRDir != dir {
		t.Errorf("ADRDir = %q, want %q", result.ADRDir, dir)
	}

	for name, path := range map[string]string{
		"template": result.Template,
		"index":    result.Index,
	} {
		if _, err := os.Stat(path); err != nil {
			t.Errorf("%s path %q is not present: %v", name, path, err)
		}
	}

	if want := filepath.Join(dir, "templates", "adr.md"); result.Template != want {
		t.Errorf("Template = %q, want %q", result.Template, want)
	}
}

func TestRunInitRegeneratesExistingIndex(t *testing.T) {
	dir := t.TempDir()
	output := &Output{Format: FormatText, Writer: &bytes.Buffer{}}

	created, err := RunNew(&NewConfig{
		Title:   "Existing decision",
		Dir:     dir,
		Status:  "proposed",
		NoIndex: true,
		Output:  output,
	})
	if err != nil {
		t.Fatalf("RunNew() error = %v", err)
	}

	if err := RunInit(&InitConfig{Dir: dir, Output: output}); err != nil {
		t.Fatalf("RunInit() error = %v", err)
	}

	idx, err := index.Load(filepath.Join(dir, index.IndexFilename))
	if err != nil {
		t.Fatalf("loading regenerated index: %v", err)
	}
	if idx.ADRCount != 1 {
		t.Errorf("ADRCount = %d, want 1", idx.ADRCount)
	}
	if len(idx.ADRs) != 1 || idx.ADRs[0].ADRID != created.ADRID {
		t.Errorf("index ADRs = %#v, want ADR %s", idx.ADRs, created.ADRID)
	}
}
