package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/averyfreeman/git-bbq/internal/codexruntime"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	switch os.Args[1] {
	case "--codex-hook":
		if err := runHook(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
	case "--repository-snapshot":
		if err := runSnapshot(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case "--version":
		fmt.Println("ai-architect-runtime 0.1.0-dev")
	default:
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "use --codex-hook --event <event> or --repository-snapshot --root <path>")
}

func runHook(args []string) error {
	flags := flag.NewFlagSet("codex-hook", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	event := flags.String("event", "", "expected Codex lifecycle event")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 || !isRequiredEvent(*event) {
		return fmt.Errorf("event must be one of UserPromptSubmit, PreToolUse, PostToolUse, PostCompact, Stop")
	}
	raw, err := io.ReadAll(io.LimitReader(os.Stdin, codexruntime.MaxHookInputBytes+1))
	if err != nil {
		return writeJSON(codexruntime.FailureResponse(*event, " Diagnostic: unable to read hook input"))
	}
	if len(raw) > codexruntime.MaxHookInputBytes {
		return writeJSON(codexruntime.FailureResponse(*event, debugDiagnostic("hook input exceeds the bounded input size")))
	}
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil || payload == nil {
		return writeJSON(codexruntime.FailureResponse(*event, debugDiagnostic("hook input must be a JSON object")))
	}
	pluginData := os.Getenv("PLUGIN_DATA")
	pluginRoot := os.Getenv("PLUGIN_ROOT")
	executable, _ := os.Executable()
	response, err := codexruntime.HandleHook(payload, codexruntime.Options{
		PluginData: pluginData,
		PluginRoot: pluginRoot,
		Executable: executable,
	}, *event)
	if err != nil {
		response = codexruntime.FailureResponse(*event, debugDiagnostic(err.Error()))
	}
	return writeJSON(response)
}

func runSnapshot(args []string) error {
	flags := flag.NewFlagSet("repository-snapshot", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	root := flags.String("root", ".", "bounded snapshot root")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected snapshot arguments")
	}
	data, err := codexruntime.RunSnapshot(*root)
	if err != nil {
		return err
	}
	_, err = os.Stdout.Write(data)
	return err
}

func writeJSON(value any) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetEscapeHTML(false)
	return encoder.Encode(value)
}

func isRequiredEvent(value string) bool {
	for _, event := range codexruntime.RequiredEvents {
		if value == event {
			return true
		}
	}
	return false
}

func debugDiagnostic(message string) string {
	if os.Getenv("AI_ARCHITECT_HOOK_DEBUG") != "1" {
		return ""
	}
	return " Diagnostic: " + message
}
