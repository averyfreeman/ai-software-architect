package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

func TestRunGithabitsPlanRejectsOlderObservedTag(t *testing.T) {
	root := t.TempDir()
	config, err := gitbbq.GitHabitsForProfile("autonomous")
	if err != nil {
		t.Fatal(err)
	}
	data, err := yaml.Marshal(config)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, gitbbq.GitHabitsFilename), data, 0o644); err != nil {
		t.Fatal(err)
	}

	if err := runGithabitsPlan([]string{
		"--action", "tag",
		"--tag", "v0.1.0",
		"--existing-tag", "v0.1.1",
		root,
		"--json",
	}); err == nil || !strings.Contains(err.Error(), "not newer") {
		t.Fatalf("older observed tag error = %v", err)
	}
}

func TestRunGithabitsPlanBuildsRemoteProvisionPlan(t *testing.T) {
	root := t.TempDir()
	config, err := gitbbq.GitHabitsForProfile("autonomous")
	if err != nil {
		t.Fatal(err)
	}
	config.Remote.Owner = "averyfreeman"
	config.Remote.Name = "example-project"
	data, err := yaml.Marshal(config)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, gitbbq.GitHabitsFilename), data, 0o644); err != nil {
		t.Fatal(err)
	}

	if err := runGithabitsPlan([]string{"--action", "remote", "--provision-remote", root, "--json"}); err != nil {
		t.Fatal(err)
	}
}
