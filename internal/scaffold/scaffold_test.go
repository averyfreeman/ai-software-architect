package scaffold

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunAllLanguages(t *testing.T) {
	for _, language := range Languages() {
		t.Run(language, func(t *testing.T) {
			dir := t.TempDir()
			result, err := Run(&Config{Language: language, Dir: dir})
			if err != nil {
				t.Fatalf("Run() error = %v", err)
			}
			if result.Language != language {
				t.Fatalf("Language = %q, want %q", result.Language, language)
			}
			if len(result.Files) != 9 {
				t.Fatalf("generated %d files, want 9: %v", len(result.Files), result.Files)
			}
			if _, err := os.Stat(filepath.Join(dir, ".git")); !os.IsNotExist(err) {
				t.Fatalf("scaffold created a Git directory: %v", err)
			}
			config, err := os.ReadFile(filepath.Join(dir, configFilename))
			if err != nil {
				t.Fatalf("read generated config: %v", err)
			}
			if !strings.Contains(string(config), "language: "+language) {
				t.Errorf("generated config does not name %q: %s", language, config)
			}
		})
	}
}

func TestRunResolvesAliases(t *testing.T) {
	tests := map[string]string{
		"ts":     "typescript",
		"c++":    "cpp",
		"clisp":  "lisp",
		"golang": "go",
	}
	for alias, want := range tests {
		t.Run(alias, func(t *testing.T) {
			result, err := Run(&Config{Language: alias, Dir: t.TempDir()})
			if err != nil {
				t.Fatalf("Run() error = %v", err)
			}
			if result.Language != want {
				t.Errorf("Language = %q, want %q", result.Language, want)
			}
		})
	}
}

func TestRunUnknownLanguage(t *testing.T) {
	_, err := Run(&Config{Language: "fortran", Dir: t.TempDir()})
	if err == nil || !strings.Contains(err.Error(), "unknown language") {
		t.Fatalf("Run() error = %v, want unknown-language error", err)
	}
}

func TestRunForceOverwritesExistingFile(t *testing.T) {
	dir := t.TempDir()
	if _, err := Run(&Config{Language: "go", Dir: dir}); err != nil {
		t.Fatalf("initial Run() error = %v", err)
	}
	agentsPath := filepath.Join(dir, "AGENTS.md")
	if err := os.WriteFile(agentsPath, []byte("old content"), 0644); err != nil {
		t.Fatalf("write existing file: %v", err)
	}
	if _, err := Run(&Config{Language: "go", Dir: dir, Force: true}); err != nil {
		t.Fatalf("forced Run() error = %v", err)
	}
	content, err := os.ReadFile(agentsPath)
	if err != nil {
		t.Fatalf("read overwritten file: %v", err)
	}
	if strings.Contains(string(content), "old content") {
		t.Fatal("--force did not overwrite existing file")
	}
}

func TestRunCollisionResolverCanRename(t *testing.T) {
	dir := t.TempDir()
	if _, err := Run(&Config{Language: "rust", Dir: dir}); err != nil {
		t.Fatalf("initial Run() error = %v", err)
	}
	var prompt bytes.Buffer
	result, err := Run(&Config{
		Language: "rust",
		Dir:      dir,
		Input:    strings.NewReader(""),
		Output:   &prompt,
		Resolve: func(path string, _ io.Reader, _ io.Writer) (Action, string, error) {
			if path == "AGENTS.md" {
				return ActionRename, "", nil
			}
			return ActionSkip, "", nil
		},
	})
	if err != nil {
		t.Fatalf("collision Run() error = %v", err)
	}
	foundRename := false
	for _, file := range result.Files {
		if strings.HasSuffix(file, "AGENTS_1.md") {
			foundRename = true
		}
	}
	if !foundRename {
		t.Fatalf("renamed file not reported: %v", result.Files)
	}
	if _, err := os.Stat(filepath.Join(dir, "AGENTS_1.md")); err != nil {
		t.Fatalf("renamed file missing: %v", err)
	}
}

func TestRunCollisionRequiresForceWhenNonInteractive(t *testing.T) {
	dir := t.TempDir()
	if _, err := Run(&Config{Language: "go", Dir: dir}); err != nil {
		t.Fatalf("initial Run() error = %v", err)
	}
	_, err := Run(&Config{
		Language: "go",
		Dir:      dir,
		Input:    strings.NewReader(""),
		Output:   &bytes.Buffer{},
	})
	if err == nil || !strings.Contains(err.Error(), "--force") {
		t.Fatalf("Run() error = %v, want actionable collision error", err)
	}
}
