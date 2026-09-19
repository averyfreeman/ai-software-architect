package architect

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var semverTagPattern = regexp.MustCompile(`^v?[0-9]+\.[0-9]+\.[0-9]+$`)
var gitBranchPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._/-]*$`)
var remotePartPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

type GitPlan struct {
	Commands []string `json:"commands"`
	Reasons  []string `json:"reasons,omitempty"`
}

type GitBootstrapResult struct {
	Plan      GitPlan  `json:"plan"`
	Completed []string `json:"completed"`
}

func shellDescription(name string, args ...string) string {
	parts := append([]string{name}, args...)
	return strings.Join(parts, " ")
}

func gitIsRepository(root string) bool {
	command := exec.Command("git", "-C", root, "rev-parse", "--is-inside-work-tree")
	output, err := command.Output()
	return err == nil && strings.TrimSpace(string(output)) == "true"
}

func gitOriginExists(root string) bool {
	command := exec.Command("git", "-C", root, "remote", "get-url", "origin")
	return command.Run() == nil
}

func PlanGitBootstrap(dir string) (GitPlan, error) {
	root, err := projectRoot(dir)
	if err != nil {
		return GitPlan{}, err
	}
	config, err := LoadGitHabits(root)
	if err != nil {
		return GitPlan{}, fmt.Errorf("load %s: %w", GitHabitsFilename, err)
	}
	if config.Branch == "" {
		config.Branch = "main"
	}
	if err := validateGitConfig(config); err != nil {
		return GitPlan{}, err
	}
	plan := GitPlan{Commands: []string{}, Reasons: []string{}}
	inside := gitIsRepository(root)
	if !inside {
		if !config.Init {
			return plan, fmt.Errorf("project is not a Git repository and init is disabled in %s", GitHabitsFilename)
		}
		plan.Commands = append(plan.Commands, shellDescription("git", "init", "-b", config.Branch))
	}
	if config.CreateRemote && (!inside || !gitOriginExists(root)) {
		target := config.Remote.Name
		if config.Remote.Owner != "" {
			target = config.Remote.Owner + "/" + target
		}
		visibility := config.Remote.Visibility
		if visibility != "public" && visibility != "private" {
			return plan, fmt.Errorf("remote.visibility must be public or private")
		}
		plan.Commands = append(plan.Commands, shellDescription("gh", "repo", "create", target, "--source", ".", "--remote", "origin", "--"+visibility))
	}
	if config.Commit {
		plan.Commands = append(plan.Commands, shellDescription("git", "add", "-A"), shellDescription("git", "commit", "-m", "chore: initialize project"))
	}
	if config.Tag {
		if !semverTagPattern.MatchString(config.InitialTag) {
			return plan, fmt.Errorf("initial_tag must be a simple SemVer tag such as v0.1.0")
		}
		plan.Commands = append(plan.Commands, shellDescription("git", "tag", config.InitialTag))
	}
	if config.Push {
		if !gitOriginExists(root) && !config.CreateRemote {
			return plan, fmt.Errorf("push is enabled but no origin will be available; enable create_remote or add origin first")
		}
		plan.Commands = append(plan.Commands, shellDescription("git", "push", "--set-upstream", "origin", config.Branch))
		if config.Tag {
			plan.Commands = append(plan.Commands, shellDescription("git", "push", "origin", config.InitialTag))
		}
	}
	if len(plan.Commands) == 0 {
		plan.Reasons = append(plan.Reasons, "No Git or remote action is enabled; configuration was inspected only.")
	}
	return plan, nil
}

func validateGitConfig(config GitHabitsConfig) error {
	if !gitBranchPattern.MatchString(config.Branch) || strings.Contains(config.Branch, "..") || strings.Contains(config.Branch, "//") || strings.Contains(config.Branch, "@{") || strings.HasSuffix(config.Branch, ".") || strings.HasSuffix(config.Branch, "/") {
		return fmt.Errorf("branch must be a safe Git ref name")
	}
	if config.CreateRemote {
		if config.Remote.Name == "" {
			return fmt.Errorf("remote.name is required when create_remote is enabled")
		}
		if !remotePartPattern.MatchString(config.Remote.Name) {
			return fmt.Errorf("remote.name must contain only safe repository-name characters")
		}
		if config.Remote.Owner != "" && !remotePartPattern.MatchString(config.Remote.Owner) {
			return fmt.Errorf("remote.owner must contain only safe owner-name characters")
		}
		if config.Remote.Visibility != "public" && config.Remote.Visibility != "private" {
			return fmt.Errorf("remote.visibility must be public or private")
		}
	}
	if config.Tag && !semverTagPattern.MatchString(config.InitialTag) {
		return fmt.Errorf("initial_tag must be a simple SemVer tag such as v0.1.0")
	}
	return nil
}

func BootstrapGit(dir string, approve bool) (GitBootstrapResult, error) {
	root, err := projectRoot(dir)
	if err != nil {
		return GitBootstrapResult{}, err
	}
	plan, err := PlanGitBootstrap(root)
	if err != nil {
		return GitBootstrapResult{Plan: plan}, err
	}
	if !approve {
		return GitBootstrapResult{Plan: plan}, fmt.Errorf("explicit approval is required; rerun with --approve after reviewing the plan")
	}
	config, err := LoadGitHabits(root)
	if err != nil {
		return GitBootstrapResult{Plan: plan}, err
	}
	completed := []string{}
	run := func(name string, args ...string) error {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		command := exec.CommandContext(ctx, name, args...)
		command.Dir = root
		command.Stdout = os.Stdout
		command.Stderr = os.Stderr
		if err := command.Run(); err != nil {
			return fmt.Errorf("%s failed: %w", shellDescription(name, args...), err)
		}
		completed = append(completed, shellDescription(name, args...))
		return nil
	}
	if !gitIsRepository(root) {
		if err := run("git", "init", "-b", config.Branch); err != nil {
			return GitBootstrapResult{Plan: plan, Completed: completed}, err
		}
	}
	if config.CreateRemote && !gitOriginExists(root) {
		target := config.Remote.Name
		if config.Remote.Owner != "" {
			target = config.Remote.Owner + "/" + target
		}
		if err := run("gh", "repo", "create", target, "--source", ".", "--remote", "origin", "--"+config.Remote.Visibility); err != nil {
			return GitBootstrapResult{Plan: plan, Completed: completed}, err
		}
	}
	if config.Commit {
		status := exec.Command("git", "-C", root, "status", "--porcelain")
		output, statusErr := status.Output()
		if statusErr != nil {
			return GitBootstrapResult{Plan: plan, Completed: completed}, statusErr
		}
		if strings.TrimSpace(string(output)) != "" {
			if err := run("git", "add", "-A"); err != nil {
				return GitBootstrapResult{Plan: plan, Completed: completed}, err
			}
			if err := run("git", "commit", "-m", "chore: initialize project"); err != nil {
				return GitBootstrapResult{Plan: plan, Completed: completed}, err
			}
		}
	}
	if config.Tag {
		existing := exec.Command("git", "-C", root, "rev-parse", "-q", "--verify", ""+config.InitialTag)
		if existing.Run() != nil {
			if err := run("git", "tag", config.InitialTag); err != nil {
				return GitBootstrapResult{Plan: plan, Completed: completed}, err
			}
		}
	}
	if config.Push {
		if err := run("git", "push", "--set-upstream", "origin", config.Branch); err != nil {
			return GitBootstrapResult{Plan: plan, Completed: completed}, err
		}
		if config.Tag {
			if err := run("git", "push", "origin", config.InitialTag); err != nil {
				return GitBootstrapResult{Plan: plan, Completed: completed}, err
			}
		}
	}
	return GitBootstrapResult{Plan: plan, Completed: completed}, nil
}

func ConfigPath(dir, filename string) string {
	root, err := projectRoot(dir)
	if err != nil {
		return filepath.Join(dir, filename)
	}
	return filepath.Join(root, filename)
}
