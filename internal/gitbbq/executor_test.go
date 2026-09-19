package gitbbq

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

type recordingGitRunner struct {
	calls     int
	workspace string
	command   []string
}

func (runner *recordingGitRunner) Run(workspace string, command []string) (GitRunResult, error) {
	runner.calls++
	runner.workspace = workspace
	runner.command = append([]string(nil), command...)
	return GitRunResult{ExitCode: 0}, nil
}

func TestExecuteGitPlanRequiresApprovalBeforeRunner(t *testing.T) {
	config, err := GitHabitsForProfile("autonomous")
	if err != nil {
		t.Fatal(err)
	}
	plan, err := PlanGitAction(config, GitActionCommit, GitPlanRequest{Message: "feat: add executor"})
	if err != nil {
		t.Fatal(err)
	}
	runner := &recordingGitRunner{}

	if _, err := ExecuteGitPlan(config, "/workspace", plan, false, runner); err == nil {
		t.Fatal("unapproved Git plan was executed")
	}
	if runner.calls != 0 {
		t.Fatalf("runner calls = %d, want 0", runner.calls)
	}
}

func TestExecuteGitPlanRunsApprovedCommandThroughRunner(t *testing.T) {
	config, err := GitHabitsForProfile("autonomous")
	if err != nil {
		t.Fatal(err)
	}
	plan, err := PlanGitAction(config, GitActionCommit, GitPlanRequest{Message: "feat: add executor"})
	if err != nil {
		t.Fatal(err)
	}
	runner := &recordingGitRunner{}

	execution, err := ExecuteGitPlan(config, "/workspace", plan, true, runner)
	if err != nil {
		t.Fatal(err)
	}
	if !execution.Executed || execution.ExitCode != 0 {
		t.Fatalf("execution = %#v", execution)
	}
	if runner.workspace != "/workspace" {
		t.Fatalf("workspace = %q", runner.workspace)
	}
	if len(runner.command) != 4 || runner.command[0] != "git" || runner.command[1] != "commit" {
		t.Fatalf("command = %#v", runner.command)
	}
}

func TestExecuteGitPlanRejectsForcePushFromTamperedPlan(t *testing.T) {
	config, err := GitHabitsForProfile("autonomous")
	if err != nil {
		t.Fatal(err)
	}
	config.Remote.Status = "configured"
	config.Remote.URL = "https://github.com/example/project.git"
	plan := GitPlan{
		SchemaVersion:    SchemaVersion,
		Action:           GitActionPush,
		Allowed:          true,
		RequiresApproval: true,
		Command:          []string{"git", "push", "--force", "origin", "main"},
	}
	runner := &recordingGitRunner{}

	if _, err := ExecuteGitPlan(config, "/workspace", plan, true, runner); err == nil {
		t.Fatal("force push was executed")
	}
	if runner.calls != 0 {
		t.Fatalf("runner calls = %d, want 0", runner.calls)
	}
}

func TestExecuteGitPlanRechecksCurrentPolicy(t *testing.T) {
	plannedConfig, err := GitHabitsForProfile("autonomous")
	if err != nil {
		t.Fatal(err)
	}
	plan, err := PlanGitAction(plannedConfig, GitActionCommit, GitPlanRequest{Message: "feat: add executor"})
	if err != nil {
		t.Fatal(err)
	}
	runner := &recordingGitRunner{}

	if _, err := ExecuteGitPlan(DefaultGitHabits(), "/workspace", plan, true, runner); err == nil {
		t.Fatal("execution ignored the current githabits policy")
	}
	if runner.calls != 0 {
		t.Fatalf("runner calls = %d, want 0", runner.calls)
	}
}

func TestOSGitRunnerRunsGitInWorkspace(t *testing.T) {
	workspace := t.TempDir()
	runner := OSGitRunner{Timeout: time.Second}

	result, err := runner.Run(workspace, []string{"git", "init", "-q"})
	if err != nil {
		t.Fatal(err)
	}
	if result.ExitCode != 0 {
		t.Fatalf("exit code = %d", result.ExitCode)
	}
	if _, err := os.Stat(filepath.Join(workspace, ".git")); err != nil {
		t.Fatalf("git repository was not initialized: %v", err)
	}
}
