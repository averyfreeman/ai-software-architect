// Package main provides the entry point for the adr CLI.
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/averyfreeman/adr-repo-governance/internal/cli"
	"github.com/averyfreeman/adr-repo-governance/internal/scaffold"
)

// Version information, set via ldflags at build time.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

const defaultADRDir = "docs/adr"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	global, command, args, err := parseGlobalOptions(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	if command == "" {
		printUsage()
		os.Exit(1)
	}

	switch command {
	case "init":
		runInit(args, global)
	case "new":
		runNew(args, global)
	case "index":
		runIndex(args, global)
	case "list":
		runList(args, global)
	case "show":
		runShow(args, global)
	case "check":
		runCheck(args, global)
	case "review", "detect-bs":
		runDetectBS(args, global, command)
	case "scaffold":
		runScaffold(args, global)
	case "version":
		printVersion()
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`adr-rg - Git-native ADR governance

Usage:
  adr [global options] <command> [options]

Global options:
  --dir PATH       ADR directory or scaffold target override
  --format FORMAT  Output format override (text|json, or yaml for index)
  --json           Shorthand for --format json

Commands:
  init          Initialize ADR directory structure
  new           Create a new ADR
  index         Generate/update the ADR index
  list          List ADRs with optional filters
  show          Display details of an ADR
  check         Validate ADRs
  review        Find ADRs relevant to changed files
  detect-bs     Compatibility alias for review
  scaffold      Generate a language-specific project scaffold
  version       Show version information
  help          Show this help message

Run 'adr <command> -h' for more information on a command.`)
}

func printVersion() {
	fmt.Printf("adr version %s\n", version)
	fmt.Printf("  commit: %s\n", commit)
	fmt.Printf("  built:  %s\n", date)
}

func runInit(args []string, global globalOptions) {
	fs := flag.NewFlagSet("init", flag.ExitOnError)
	common := addCommonFlags(fs, defaultADRDir, global, "Output format (text|json)")
	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	outputFormat, err := common.outputFormat(false)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	cfg := &cli.InitConfig{
		Dir:    *common.Dir,
		Output: cli.NewOutput(outputFormat),
	}

	if err := cli.RunInit(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func runNew(args []string, global globalOptions) {
	fs := flag.NewFlagSet("new", flag.ExitOnError)
	common := addCommonFlags(fs, defaultADRDir, global, "Output format (text|json)")
	language := stringFlag(fs, "lang", "l", "", "Language profile or alias (optional)")
	tags := stringListFlag(fs, "tags", "t", "Tags")
	fs.Var(tags, "tag", "Alias for --tags")
	paths := stringListFlag(fs, "paths", "p", "Scope paths (globs)")
	fs.Var(paths, "path", "Alias for --paths")
	status := stringFlag(fs, "status", "s", "proposed", "Initial status")
	noIndex := boolFlag(fs, "no-index", "n", false, "Skip updating index")

	fs.Usage = func() {
		fmt.Println("Usage: adr new [options] <title>")
		fmt.Println()
		fmt.Println("Create a new ADR with the given title.")
		fmt.Println()
		fmt.Println("Options:")
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	if fs.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "error: title is required")
		fs.Usage()
		os.Exit(1)
	}

	title := strings.Join(fs.Args(), " ")

	outputFormat, err := common.outputFormat(false)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	cfg := &cli.NewConfig{
		Title:    title,
		Dir:      *common.Dir,
		Tags:     []string(*tags),
		Paths:    []string(*paths),
		Status:   *status,
		Language: *language,
		NoIndex:  *noIndex,
		Format:   outputFormat,
		Output:   cli.NewOutput(outputFormat),
	}

	if _, err := cli.RunNew(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func runIndex(args []string, global globalOptions) {
	fs := flag.NewFlagSet("index", flag.ExitOnError)
	common := addCommonFlags(fs, defaultADRDir, global, "Output format (text|json|yaml)")
	check := boolFlag(fs, "check", "c", false, "Check if index is up-to-date (don't modify)")
	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	outputFormat, err := common.outputFormat(true)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	cfg := &cli.IndexConfig{
		Dir:    *common.Dir,
		Check:  *check,
		Format: outputFormat,
		Output: cli.NewOutput(outputFormat),
	}

	_, err = cli.RunIndex(cfg)
	if err != nil {
		if *check {
			os.Exit(2) // Lint failure exit code
		}
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func runList(args []string, global globalOptions) {
	fs := flag.NewFlagSet("list", flag.ExitOnError)
	common := addCommonFlags(fs, defaultADRDir, global, "Output format (text|json)")
	status := stringFlag(fs, "status", "s", "", "Filter by status")
	tag := stringListFlag(fs, "tag", "t", "Filter by tag")
	path := stringListFlag(fs, "path", "p", "Filter by scope path match")
	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	outputFormat, err := common.outputFormat(false)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	cfg := &cli.ListConfig{
		Dir:    *common.Dir,
		Status: *status,
		Tags:   []string(*tag),
		Paths:  []string(*path),
		Format: outputFormat,
		Output: cli.NewOutput(outputFormat),
	}

	if _, err := cli.RunList(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func runShow(args []string, global globalOptions) {
	fs := flag.NewFlagSet("show", flag.ExitOnError)
	common := addCommonFlags(fs, defaultADRDir, global, "Output format (text|json)")

	fs.Usage = func() {
		fmt.Println("Usage: adr show [options] <ADR-ID|number|filename>")
		fmt.Println()
		fmt.Println("Display details of a specific ADR.")
		fmt.Println()
		fmt.Println("Options:")
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	if fs.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "error: ADR identifier is required")
		fs.Usage()
		os.Exit(1)
	}

	outputFormat, err := common.outputFormat(false)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	cfg := &cli.ShowConfig{
		ID:     fs.Arg(0),
		Dir:    *common.Dir,
		Format: outputFormat,
		Output: cli.NewOutput(outputFormat),
	}

	if _, err := cli.RunShow(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func runCheck(args []string, global globalOptions) {
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		if args[0] != "adr" {
			fmt.Fprintf(os.Stderr, "Unknown check subcommand: %s\n", args[0])
			os.Exit(1)
		}
		args = args[1:]
	}
	runCheckADR(args, global)
}

func runCheckADR(args []string, global globalOptions) {
	fs := flag.NewFlagSet("check", flag.ExitOnError)
	common := addCommonFlags(fs, defaultADRDir, global, "Output format (text|json)")
	strict := boolFlag(fs, "strict", "s", false, "Treat warnings as errors (fail on missing rationale pattern)")
	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	outputFormat, err := common.outputFormat(false)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	cfg := &cli.CheckADRConfig{
		Dir:    *common.Dir,
		Strict: *strict,
		Format: outputFormat,
		Output: cli.NewOutput(outputFormat),
	}

	result, err := cli.RunCheckADR(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	if !result.Valid {
		os.Exit(2) // Lint failure exit code
	}
}

func runDetectBS(args []string, global globalOptions, commandName string) {
	fs := flag.NewFlagSet(commandName, flag.ExitOnError)
	common := addCommonFlags(fs, defaultADRDir, global, "Output format (text|json)")
	base := stringFlag(fs, "base", "b", cli.DefaultBaseRef(), "Base ref for git diff (auto-detected when unambiguous)")

	fs.Usage = func() {
		fmt.Printf("Usage: adr %s [--base <ref>] [options]\n", commandName)
		fmt.Println()
		fmt.Println("Find ADRs whose documented scope overlaps changed files.")
		fmt.Println()
		fmt.Println("Options:")
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	if *base == "" {
		fmt.Fprintln(os.Stderr, "error: --base is required when no unambiguous Git base can be detected")
		fs.Usage()
		os.Exit(1)
	}

	outputFormat, err := common.outputFormat(false)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	cfg := &cli.DetectBSConfig{
		Dir:    *common.Dir,
		Base:   *base,
		Format: outputFormat,
		Output: cli.NewOutput(outputFormat),
	}

	if _, err := cli.RunDetectBS(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func runScaffold(args []string, global globalOptions) {
	fs := flag.NewFlagSet("scaffold", flag.ExitOnError)
	language := stringFlag(fs, "lang", "l", "", "Language template (may be positional)")
	common := addCommonFlags(fs, ".", global, "Output format (text|json)")
	force := boolFlag(fs, "force", "F", false, "Overwrite existing files without prompting")

	fs.Usage = func() {
		fmt.Println("Usage: adr scaffold [--lang <language>|<language>] [options]")
		fmt.Println()
		fmt.Println("Generate an offline, language-specific project scaffold.")
		fmt.Println()
		fmt.Println("Available languages:", strings.Join(scaffold.Languages(), ", "))
		fmt.Println()
		fmt.Println("Options:")
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	if *language == "" {
		if fs.NArg() == 1 {
			*language = fs.Arg(0)
		} else if fs.NArg() > 1 {
			fmt.Fprintln(os.Stderr, "error: scaffold accepts one positional language")
			fs.Usage()
			os.Exit(1)
		}
	}
	if *language == "" {
		fmt.Fprintln(os.Stderr, "error: language is required")
		fs.Usage()
		os.Exit(1)
	}

	outputFormat, err := common.outputFormat(false)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	result, err := scaffold.Run(&scaffold.Config{
		Language: *language,
		Dir:      *common.Dir,
		Force:    *force,
		Input:    os.Stdin,
		Output:   os.Stderr,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	output := cli.NewOutput(outputFormat)
	if output.IsStructuredFormat() {
		_ = output.PrintStructured(result)
		return
	}
	output.Success("Scaffolded %s project in %s", result.Language, result.Directory)
	for _, file := range result.Files {
		output.Println("  Created %s", file)
	}
}

type globalOptions struct {
	Dir    string
	Format string
}

type commonFlags struct {
	Dir    *string
	Format *string
	JSON   *bool
}

func addCommonFlags(fs *flag.FlagSet, defaultDir string, global globalOptions, formatUsage string) commonFlags {
	if global.Dir != "" {
		defaultDir = global.Dir
	}
	defaultFormat := global.Format
	if defaultFormat == "" {
		defaultFormat = "text"
	}

	dir := stringFlag(fs, "dir", "d", defaultDir, "Directory path")
	format := stringFlag(fs, "format", "o", defaultFormat, formatUsage)
	json := fs.Bool("json", false, "Output JSON (shorthand for --format json)")
	return commonFlags{Dir: dir, Format: format, JSON: json}
}

func (f commonFlags) outputFormat(allowYAML bool) (cli.OutputFormat, error) {
	format := *f.Format
	if *f.JSON {
		format = "json"
	}
	if allowYAML {
		return cli.ParseIndexOutputFormat(format)
	}
	return cli.ParseOutputFormat(format)
}

func parseGlobalOptions(args []string) (globalOptions, string, []string, error) {
	var global globalOptions
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--dir" || arg == "-d":
			if i+1 >= len(args) {
				return global, "", nil, fmt.Errorf("%s requires a value", arg)
			}
			global.Dir = args[i+1]
			i++
		case strings.HasPrefix(arg, "--dir="):
			global.Dir = strings.TrimPrefix(arg, "--dir=")
		case strings.HasPrefix(arg, "-d="):
			global.Dir = strings.TrimPrefix(arg, "-d=")
		case arg == "--format" || arg == "-o":
			if i+1 >= len(args) {
				return global, "", nil, fmt.Errorf("%s requires a value", arg)
			}
			global.Format = args[i+1]
			i++
		case strings.HasPrefix(arg, "--format="):
			global.Format = strings.TrimPrefix(arg, "--format=")
		case strings.HasPrefix(arg, "-o="):
			global.Format = strings.TrimPrefix(arg, "-o=")
		case arg == "--json":
			global.Format = "json"
		case arg == "--":
			if i+1 >= len(args) {
				return global, "", nil, fmt.Errorf("a command is required after --")
			}
			return global, args[i+1], args[i+2:], nil
		default:
			return global, arg, args[i+1:], nil
		}
	}
	return global, "", nil, nil
}

func stringFlag(fs *flag.FlagSet, name, short, value, usage string) *string {
	result := fs.String(name, value, usage)
	fs.StringVar(result, short, value, usage+" (short form)")
	return result
}

type stringListValues []string

func (f *stringListValues) String() string {
	return strings.Join(*f, ",")
}

func (f *stringListValues) Set(value string) error {
	for _, item := range strings.Split(value, ",") {
		item = strings.TrimSpace(item)
		if item != "" {
			*f = append(*f, item)
		}
	}
	return nil
}

func stringListFlag(fs *flag.FlagSet, name, short, usage string) *stringListValues {
	result := &stringListValues{}
	fs.Var(result, name, usage+" (repeatable; comma-separated)")
	fs.Var(result, short, usage+" (short form; repeatable)")
	return result
}

func boolFlag(fs *flag.FlagSet, name, short string, value bool, usage string) *bool {
	result := fs.Bool(name, value, usage)
	fs.BoolVar(result, short, value, usage+" (short form)")
	return result
}
