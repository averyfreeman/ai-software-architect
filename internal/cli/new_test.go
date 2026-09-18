package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/averyfreeman/adr-repo-governance/internal/scaffold"
)

func TestRunNewUsesInitializedTemplate(t *testing.T) {
	dir := t.TempDir()
	templateDir := filepath.Join(dir, "templates")
	if err := os.MkdirAll(templateDir, 0755); err != nil {
		t.Fatalf("creating template directory: %v", err)
	}

	template := `---
adr_id: ADR-NNNN
title: "Your Decision Title Here"
status: proposed
date: YYYY-MM-DD
---

# ADR-NNNN: Your Decision Title Here

## Context

Custom initialized context.
Literal template text: {{ .Language }}

## Decision

### [Chosen Option]: Adopted

**Adopted because:**
- Custom rationale.

**Adopted despite:**
- Custom trade-off.

## Alternatives Considered

### [Alternative A]: Rejected

**Rejected because:**
- Custom rejection rationale.

**Rejected despite:**
- Custom alternative strength.

## Consequences

**Positive:**
- Custom positive consequence.

**Negative:**
- Custom negative consequence.
`
	templatePath := filepath.Join(templateDir, "adr.md")
	if err := os.WriteFile(templatePath, []byte(template), 0644); err != nil {
		t.Fatalf("writing template: %v", err)
	}

	result, err := RunNew(&NewConfig{
		Title:  "Use initialized template",
		Dir:    dir,
		Tags:   []string{"go"},
		Paths:  []string{"internal/**"},
		Status: "proposed",
		Output: &Output{Format: FormatText, Writer: &bytes.Buffer{}},
	})
	if err != nil {
		t.Fatalf("RunNew() error = %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, result.File))
	if err != nil {
		t.Fatalf("reading created ADR: %v", err)
	}
	text := string(content)
	for _, want := range []string{
		"Custom initialized context.",
		"Literal template text: {{ .Language }}",
		"ADR-0001: Use initialized template",
		"tags:",
		"go",
		"internal/**",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("created ADR does not contain %q:\n%s", want, text)
		}
	}
}

func TestRunNewDefaultsToGenericWithoutLanguageContext(t *testing.T) {
	dir := t.TempDir()
	var output bytes.Buffer

	result, err := RunNew(&NewConfig{
		Title:   "Use the generic ADR profile",
		Dir:     dir,
		Status:  "proposed",
		NoIndex: true,
		Output:  &Output{Format: FormatText, Writer: &output},
	})
	if err != nil {
		t.Fatalf("RunNew() error = %v", err)
	}
	if result.Language != "generic" {
		t.Fatalf("Language = %q, want generic", result.Language)
	}

	content, err := os.ReadFile(filepath.Join(dir, result.File))
	if err != nil {
		t.Fatalf("reading created ADR: %v", err)
	}
	if strings.Contains(string(content), "## Language Context") {
		t.Fatalf("default ADR unexpectedly contains language context:\n%s", content)
	}

	for _, want := range []string{
		"Default ADR language: generic.",
		"Use --lang <language> to select a language-specific template.",
		"Available languages: c, cpp, elixir, erlang, generic, go, haskell, html, javascript, lisp, lua, php, rust, typescript.",
	} {
		if !strings.Contains(output.String(), want) {
			t.Errorf("text output does not contain %q:\n%s", want, output.String())
		}
	}
}

func TestRunNewRendersExplicitLanguageProfiles(t *testing.T) {
	tests := []struct {
		name         string
		language     string
		wantLanguage string
		wantDisplay  string
		wantBuild    string
		wantTest     string
		wantFormat   string
		wantDocument string
	}{
		{
			name:         "go",
			language:     "go",
			wantLanguage: "go",
			wantDisplay:  "Go",
			wantBuild:    "go build ./...",
			wantTest:     "go test ./...",
			wantFormat:   "gofmt -w .",
			wantDocument: "Go doc comments",
		},
		{
			name:         "rust alias",
			language:     "rs",
			wantLanguage: "rust",
			wantDisplay:  "Rust",
			wantBuild:    "cargo build",
			wantTest:     "cargo test",
			wantFormat:   "cargo fmt --check",
			wantDocument: "Rustdoc",
		},
		{
			name:         "generic",
			language:     "generic",
			wantLanguage: "generic",
			wantDisplay:  "Generic",
			wantBuild:    "not defined",
			wantTest:     "not defined",
			wantFormat:   "not defined",
			wantDocument: "Markdown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			var output bytes.Buffer
			result, err := RunNew(&NewConfig{
				Title:    "Use an explicit language profile",
				Dir:      dir,
				Status:   "proposed",
				Language: tt.language,
				NoIndex:  true,
				Output:   &Output{Format: FormatText, Writer: &output},
			})
			if err != nil {
				t.Fatalf("RunNew() error = %v", err)
			}
			if result.Language != tt.wantLanguage {
				t.Errorf("Language = %q, want %q", result.Language, tt.wantLanguage)
			}
			content, err := os.ReadFile(filepath.Join(dir, result.File))
			if err != nil {
				t.Fatalf("reading created ADR: %v", err)
			}
			text := string(content)
			for _, want := range []string{
				"## Language Context",
				tt.wantDisplay,
				tt.wantBuild,
				tt.wantTest,
				tt.wantFormat,
				tt.wantDocument,
			} {
				if !strings.Contains(text, want) {
					t.Errorf("created ADR does not contain %q:\n%s", want, text)
				}
			}
			if strings.Contains(output.String(), "Default ADR language:") {
				t.Errorf("explicit language emitted default notice:\n%s", output.String())
			}
		})
	}
}

func TestRunNewAllLanguageProfilesRender(t *testing.T) {
	for _, language := range scaffold.Languages() {
		t.Run(language, func(t *testing.T) {
			result, err := RunNew(&NewConfig{
				Title:    "Render every supported language",
				Dir:      t.TempDir(),
				Status:   "proposed",
				Language: language,
				NoIndex:  true,
				Output:   &Output{Format: FormatJSON, Writer: &bytes.Buffer{}},
			})
			if err != nil {
				t.Fatalf("RunNew(%q) error = %v", language, err)
			}
			if result.Language != language {
				t.Errorf("Language = %q, want %q", result.Language, language)
			}
		})
	}
}

func TestRunNewJSONIncludesLanguageAndNoticeStaysOut(t *testing.T) {
	dir := t.TempDir()
	var output bytes.Buffer

	result, err := RunNew(&NewConfig{
		Title:   "Create JSON output",
		Dir:     dir,
		Status:  "proposed",
		NoIndex: true,
		Format:  FormatJSON,
		Output:  &Output{Format: FormatJSON, Writer: &output},
	})
	if err != nil {
		t.Fatalf("RunNew() error = %v", err)
	}
	var got NewResult
	if err := json.Unmarshal(output.Bytes(), &got); err != nil {
		t.Fatalf("new output is not valid JSON: %v\n%s", err, output.String())
	}
	if got.Language != result.Language || got.Language != "generic" {
		t.Errorf("JSON language = %q, result language = %q", got.Language, result.Language)
	}
	if strings.Contains(output.String(), "Default ADR language:") || strings.Contains(output.String(), "Available languages:") {
		t.Fatalf("JSON output contains text notice:\n%s", output.String())
	}
}

func TestRunNewInvalidLanguageDoesNotWriteADR(t *testing.T) {
	dir := t.TempDir()
	_, err := RunNew(&NewConfig{
		Title:    "Reject an unknown language",
		Dir:      dir,
		Status:   "proposed",
		Language: "fortran",
		NoIndex:  true,
		Output:   &Output{Format: FormatText, Writer: &bytes.Buffer{}},
	})
	if err == nil || !strings.Contains(err.Error(), "unknown language") {
		t.Fatalf("RunNew() error = %v, want unknown-language error", err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("reading ADR directory: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("invalid language wrote files: %v", entries)
	}
}
