package cli

import (
	"bytes"
	"testing"
)

func TestADRCommandLifecycle(t *testing.T) {
	dir := t.TempDir()
	output := &Output{Format: FormatText, Writer: &bytes.Buffer{}}

	if err := RunInit(&InitConfig{Dir: dir, Output: output}); err != nil {
		t.Fatalf("RunInit() error = %v", err)
	}

	created, err := RunNew(&NewConfig{
		Title:  "Lifecycle decision",
		Dir:    dir,
		Status: "proposed",
		Output: output,
	})
	if err != nil {
		t.Fatalf("RunNew() error = %v", err)
	}

	listed, err := RunList(&ListConfig{Dir: dir, Output: output})
	if err != nil {
		t.Fatalf("RunList() error = %v", err)
	}
	if listed.Count != 1 || listed.ADRs[0].ADRID != created.ADRID {
		t.Fatalf("RunList() = %#v, want ADR %s", listed, created.ADRID)
	}

	shown, err := RunShow(&ShowConfig{ID: created.ADRID, Dir: dir, Output: output})
	if err != nil {
		t.Fatalf("RunShow() error = %v", err)
	}
	if shown.ADRID != created.ADRID {
		t.Errorf("RunShow() ADRID = %q, want %q", shown.ADRID, created.ADRID)
	}

	checked, err := RunCheckADR(&CheckADRConfig{Dir: dir, Strict: true, Output: output})
	if err != nil {
		t.Fatalf("RunCheckADR() error = %v", err)
	}
	if !checked.Valid || checked.Count != 1 {
		t.Errorf("RunCheckADR() = %#v, want one valid ADR", checked)
	}

	indexed, err := RunIndex(&IndexConfig{Dir: dir, Check: true, Output: output})
	if err != nil {
		t.Fatalf("RunIndex(check) error = %v", err)
	}
	if !indexed.UpToDate {
		t.Errorf("RunIndex(check) = %#v, want up-to-date index", indexed)
	}
}
