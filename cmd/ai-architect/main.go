package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/averyfreeman/git-bbq/internal/architect"
)

type stringList []string

func (values *stringList) String() string         { return strings.Join(*values, ",") }
func (values *stringList) Set(value string) error { *values = append(*values, value); return nil }

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	var err error
	switch os.Args[1] {
	case "questions":
		err = runQuestions(os.Args[2:])
	case "setup":
		err = runSetup(os.Args[2:])
	case "decision":
		err = runDecision(os.Args[2:])
	case "validate-contract":
		err = runValidateContract(os.Args[2:])
	case "scan-artifact":
		err = runScanArtifact(os.Args[2:])
	case "validate-bundle":
		err = runValidateBundle(os.Args[2:])
	case "analyze-dependencies":
		err = runAnalyzeDependencies(os.Args[2:])
	case "git":
		err = runGit(os.Args[2:])
	case "version":
		fmt.Printf("ai-architect %s\n", architect.ToolVersion)
	default:
		usage()
		err = fmt.Errorf("unknown command %q", os.Args[1])
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Println(`ai-architect - agent-facing architecture and project scaffold

Usage:
  ai-architect questions [--format json]
  ai-architect setup --answers answers.json [--dir .] [--force] [--format json]
  ai-architect decision <init|new|index|list|show|check> ...
  ai-architect validate-contract [--dir .] contract.yaml
  ai-architect scan-artifact [--dir .] --kind <adr|contract|context|implementation-plan> artifact
  ai-architect validate-bundle [--dir .] [--strict] [--format json|text]
  ai-architect analyze-dependencies --dir . [--root path] [--language go]
  ai-architect git bootstrap [--dry-run|--approve]

The setup command is intended to be called by an agent after asking the generated
questionnaire. It never implies permission for Git, network, or publication actions.`)
}

func runQuestions(args []string) error {
	fs := flag.NewFlagSet("questions", flag.ContinueOnError)
	format := fs.String("format", "json", "output format (json|text)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *format == "json" {
		return writeJSON(architect.Questions())
	}
	for _, question := range architect.Questions() {
		defaultText := ""
		if question.Default != "" {
			defaultText = " [default: " + question.Default + "]"
		}
		fmt.Printf("%s: %s%s\n", question.ID, question.Prompt, defaultText)
	}
	return nil
}

func runSetup(args []string) error {
	fs := flag.NewFlagSet("setup", flag.ContinueOnError)
	dir := fs.String("dir", ".", "project directory")
	answersPath := fs.String("answers", "", "JSON answers file, or - for stdin")
	force := fs.Bool("force", false, "overwrite generated files (otherwise preserve existing files)")
	format := fs.String("format", "text", "output format (json|text)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *answersPath == "" {
		return errors.New("--answers is required; run `ai-architect questions --format json` first")
	}
	answers, err := readAnswers(*answersPath)
	if err != nil {
		return err
	}
	result, err := architect.SetupProject(*dir, answers, *force)
	if err != nil {
		return err
	}
	if *format == "json" {
		return writeJSON(result)
	}
	fmt.Printf("Configured %s project scaffold.\n", result.Language)
	fmt.Printf("Created %d files; preserved %d existing files.\n", len(result.Created), len(result.Skipped))
	for _, warning := range result.Warnings {
		fmt.Printf("Warning: %s\n", warning)
	}
	return nil
}

func readAnswers(path string) (architect.SetupAnswer, error) {
	var reader io.Reader
	if path == "-" {
		reader = os.Stdin
	} else {
		file, err := os.Open(filepath.Clean(path))
		if err != nil {
			return architect.SetupAnswer{}, err
		}
		defer file.Close()
		reader = file
	}
	decoder := json.NewDecoder(reader)
	decoder.DisallowUnknownFields()
	var answers architect.SetupAnswer
	if err := decoder.Decode(&answers); err != nil {
		return architect.SetupAnswer{}, fmt.Errorf("parse setup answers: %w", err)
	}
	return answers, nil
}

func runDecision(args []string) error {
	if len(args) == 0 {
		return errors.New("decision subcommand is required: init, new, index, list, show, or check")
	}
	switch args[0] {
	case "init":
		return runDecisionInit(args[1:])
	case "new":
		return runDecisionNew(args[1:])
	case "index":
		return runDecisionIndex(args[1:])
	case "list":
		return runDecisionList(args[1:])
	case "show":
		return runDecisionShow(args[1:])
	case "check":
		return runDecisionCheck(args[1:])
	default:
		return fmt.Errorf("unknown decision subcommand %q", args[0])
	}
}

func runDecisionInit(args []string) error {
	fs := flag.NewFlagSet("decision init", flag.ContinueOnError)
	dir := fs.String("dir", ".", "project directory")
	if err := fs.Parse(args); err != nil {
		return err
	}
	root := architect.ConfigPath(*dir, architect.DecisionsDir)
	if err := os.MkdirAll(filepath.Join(root, "templates"), 0o755); err != nil {
		return err
	}
	index := filepath.Join(root, "index.json")
	if _, err := os.Stat(index); os.IsNotExist(err) {
		content := fmt.Sprintf("{\n  \"schema_version\": \"%s\",\n  \"generated_at\": \"not-generated\",\n  \"decision_count\": 0,\n  \"decisions\": []\n}\n", architect.SchemaVersion)
		if err := os.WriteFile(index, []byte(content), 0o644); err != nil {
			return err
		}
	}
	fmt.Println("Decision directory ready:", root)
	return nil
}

func runDecisionNew(args []string) error {
	fs := flag.NewFlagSet("decision new", flag.ContinueOnError)
	dir := fs.String("dir", ".", "project directory")
	status := fs.String("status", "proposed", "decision status")
	format := fs.String("format", "text", "output format (json|text)")
	titleFlag := fs.String("title", "", "decision title")
	if err := fs.Parse(args); err != nil {
		return err
	}
	title := strings.TrimSpace(*titleFlag)
	if title == "" && fs.NArg() > 0 {
		title = strings.Join(fs.Args(), " ")
	}
	if title == "" {
		return errors.New("decision title is required")
	}
	record, err := architect.NewDecision(*dir, title, *status)
	if err != nil {
		return err
	}
	if *format == "json" {
		return writeJSON(record)
	}
	fmt.Printf("Created %s\n", record.Path)
	fmt.Printf("Decision %s is %s and requires agent/human review before acceptance.\n", record.Artifact.Decision.ID, record.Artifact.Decision.Status)
	return nil
}

func runDecisionIndex(args []string) error {
	fs := flag.NewFlagSet("decision index", flag.ContinueOnError)
	dir := fs.String("dir", ".", "project directory")
	check := fs.Bool("check", false, "check freshness without writing")
	format := fs.String("format", "text", "output format (json|text)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *check {
		fresh, err := architect.IndexMatches(*dir)
		if err != nil {
			return err
		}
		if *format == "json" {
			return writeJSON(map[string]any{"up_to_date": fresh})
		}
		if !fresh {
			return errors.New("decision index is stale or missing")
		}
		fmt.Println("Decision index is up to date.")
		return nil
	}
	index, err := architect.IndexDecisions(*dir)
	if err != nil {
		return err
	}
	if *format == "json" {
		return writeJSON(index)
	}
	fmt.Printf("Indexed %d decisions.\n", index.DecisionCount)
	return nil
}

func runDecisionList(args []string) error {
	fs := flag.NewFlagSet("decision list", flag.ContinueOnError)
	dir := fs.String("dir", ".", "project directory")
	status := fs.String("status", "", "filter by status")
	format := fs.String("format", "text", "output format (json|text)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	decisions, invalid, err := architect.ListDecisions(*dir, *status)
	if err != nil {
		return err
	}
	if *format == "json" {
		items := make([]any, 0, len(decisions))
		for _, record := range decisions {
			items = append(items, record.Artifact.Decision)
		}
		return writeJSON(map[string]any{"decisions": items, "invalid_files": invalid, "files_examined": len(decisions) + len(invalid)})
	}
	for _, record := range decisions {
		d := record.Artifact.Decision
		fmt.Printf("%s\t%s\t%s\t%s\n", d.ID, d.Status, d.Title, record.Filename)
	}
	for _, file := range invalid {
		fmt.Printf("invalid\t%s\n", file)
	}
	return nil
}

func runDecisionShow(args []string) error {
	fs := flag.NewFlagSet("decision show", flag.ContinueOnError)
	dir := fs.String("dir", ".", "project directory")
	format := fs.String("format", "text", "output format (json|text)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() == 0 {
		return errors.New("decision identifier is required")
	}
	record, err := architect.FindDecision(*dir, fs.Arg(0))
	if err != nil {
		return err
	}
	if *format == "json" {
		return writeJSON(record)
	}
	fmt.Print(record.Body)
	return nil
}

func runDecisionCheck(args []string) error {
	fs := flag.NewFlagSet("decision check", flag.ContinueOnError)
	dir := fs.String("dir", ".", "project directory")
	strict := fs.Bool("strict", false, "treat warnings as failures")
	format := fs.String("format", "text", "output format (json|text)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	checks, err := architect.CheckDecisions(*dir)
	if err != nil {
		return err
	}
	invalid := false
	if *format == "json" {
		if err := writeJSON(checks); err != nil {
			return err
		}
	}
	for _, check := range checks {
		if !check.Result.Valid || (*strict && len(check.Result.Warnings) > 0) {
			invalid = true
		}
		if *format != "json" {
			for _, issue := range check.Result.Errors {
				fmt.Printf("error %s %s: %s\n", check.File, issue.Path, issue.Message)
			}
			for _, issue := range check.Result.Warnings {
				fmt.Printf("warning %s %s: %s\n", check.File, issue.Path, issue.Message)
			}
			if check.Error != "" {
				fmt.Printf("error %s: %s\n", check.File, check.Error)
			}
		}
	}
	if invalid {
		return errors.New("one or more decisions failed validation")
	}
	return nil
}

func runValidateContract(args []string) error {
	fs := flag.NewFlagSet("validate-contract", flag.ContinueOnError)
	dir := fs.String("dir", ".", "project directory for a relative artifact path")
	format := fs.String("format", "text", "output format (json|text)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() == 0 {
		return errors.New("contract path is required")
	}
	data, err := os.ReadFile(resolveInputPath(*dir, fs.Arg(0)))
	if err != nil {
		return err
	}
	_, result := architect.ValidateContract(string(data))
	if *format == "json" {
		if err := writeJSON(result); err != nil {
			return err
		}
	}
	if !result.Valid {
		for _, issue := range result.Errors {
			fmt.Printf("error %s: %s\n", issue.Path, issue.Message)
		}
		return errors.New("contract is invalid")
	}
	if *format != "json" {
		fmt.Println("Contract is valid.")
	}
	return nil
}

func runScanArtifact(args []string) error {
	fs := flag.NewFlagSet("scan-artifact", flag.ContinueOnError)
	dir := fs.String("dir", ".", "project directory for a relative artifact path")
	kind := fs.String("kind", "", "artifact kind")
	format := fs.String("format", "text", "output format (json|text)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() == 0 || *kind == "" {
		return errors.New("artifact path and --kind are required")
	}
	data, err := os.ReadFile(resolveInputPath(*dir, fs.Arg(0)))
	if err != nil {
		return err
	}
	result := architect.ScanArtifact(string(data), *kind)
	if *format == "json" {
		return writeJSON(result)
	}
	if result.SafeToWrite {
		fmt.Println("Artifact is safe to write.")
		return nil
	}
	for _, finding := range result.Findings {
		fmt.Printf("%s at line %d\n", finding.Category, finding.Line)
	}
	return errors.New("artifact contains secret-like content")
}

func runValidateBundle(args []string) error {
	fs := flag.NewFlagSet("validate-bundle", flag.ContinueOnError)
	dir := fs.String("dir", ".", "project directory")
	strict := fs.Bool("strict", false, "treat warnings as failures")
	format := fs.String("format", "text", "output format (json|text)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	bundle, result, err := architect.ValidateProjectBundle(*dir)
	if err != nil {
		return err
	}
	summary := architect.SummarizeArtifactBundle(bundle, result)
	if *format == "json" {
		if err := writeJSON(summary); err != nil {
			return err
		}
	} else {
		for _, issue := range result.Errors {
			fmt.Printf("error %s: %s\n", issue.Path, issue.Message)
		}
		for _, issue := range result.Warnings {
			fmt.Printf("warning %s: %s\n", issue.Path, issue.Message)
		}
		if result.Valid && (!*strict || len(result.Warnings) == 0) {
			fmt.Printf("Artifact bundle is valid (%d decision(s)).\n", summary.DecisionCount)
		}
	}
	if !result.Valid || (*strict && len(result.Warnings) > 0) {
		return errors.New("architecture artifact bundle is invalid")
	}
	return nil
}

func resolveInputPath(dir, input string) string {
	if filepath.IsAbs(input) {
		return filepath.Clean(input)
	}
	return filepath.Join(dir, filepath.FromSlash(input))
}

func runAnalyzeDependencies(args []string) error {
	fs := flag.NewFlagSet("analyze-dependencies", flag.ContinueOnError)
	dir := fs.String("dir", ".", "project directory")
	format := fs.String("format", "text", "output format (json|text)")
	var roots stringList
	var languages stringList
	fs.Var(&roots, "root", "relative source root (repeatable)")
	fs.Var(&languages, "language", "language profile (repeatable)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	result, err := architect.AnalyzeDependencies(*dir, roots, languages)
	if err != nil {
		return err
	}
	if *format == "json" {
		return writeJSON(result)
	}
	for _, edge := range result.Edges {
		fmt.Printf("%s -> %s (%s)\n", edge.Source, edge.Target, edge.Evidence)
	}
	for _, warning := range result.Warnings {
		fmt.Printf("warning: %s\n", warning)
	}
	return nil
}

func runGit(args []string) error {
	if len(args) == 0 || args[0] != "bootstrap" {
		return errors.New("git subcommand must be bootstrap")
	}
	fs := flag.NewFlagSet("git bootstrap", flag.ContinueOnError)
	dir := fs.String("dir", ".", "project directory")
	dryRun := fs.Bool("dry-run", false, "show planned commands without executing")
	approve := fs.Bool("approve", false, "explicitly approve the planned commands")
	format := fs.String("format", "text", "output format (json|text)")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	plan, err := architect.PlanGitBootstrap(*dir)
	if err != nil {
		return err
	}
	if *dryRun || !*approve {
		if *format == "json" {
			return writeJSON(plan)
		}
		for _, command := range plan.Commands {
			fmt.Println(command)
		}
		if !*dryRun {
			return errors.New("no Git action was executed; rerun with --approve after reviewing the plan")
		}
		return nil
	}
	result, err := architect.BootstrapGit(*dir, true)
	if *format == "json" {
		if writeErr := writeJSON(result); writeErr != nil {
			return writeErr
		}
	} else {
		for _, command := range result.Completed {
			fmt.Println("completed:", command)
		}
	}
	return err
}

func writeJSON(value any) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}
