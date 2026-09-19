package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/averyfreeman/git-bbq/internal/gitbbq"
	"gopkg.in/yaml.v3"
)

func TestRunGithabitsExecutePersistsConfiguredRemote(t *testing.T) {
	root := t.TempDir()
	config, err := gitbbq.GitHabitsForProfile("autonomous")
	if err != nil {
		t.Fatal(err)
	}
	config.Remote.URL = "https://github.com/example/project.git"
	data, err := yaml.Marshal(config)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, gitbbq.GitHabitsFilename), data, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := exec.Command("git", "-C", root, "init", "-q").Run(); err != nil {
		t.Fatal(err)
	}

	if err := runGithabitsExecute([]string{"--approve", "--action", "remote", root, "--json"}); err != nil {
		t.Fatal(err)
	}

	persisted, err := gitbbq.ReadGitHabits(root)
	if err != nil {
		t.Fatal(err)
	}
	if persisted.Remote.Status != "configured" {
		t.Fatalf("remote status = %q", persisted.Remote.Status)
	}
}
