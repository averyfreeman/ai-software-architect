package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/averyfreeman/adr-repo-governance/internal/architect"
	"github.com/averyfreeman/adr-repo-governance/internal/codex"
)

var version = "dev"

var hookEvents = map[string]bool{
	"UserPromptSubmit": true,
	"PreToolUse":       true,
	"PostToolUse":      true,
	"PostCompact":      true,
	"Stop":             true,
}

func usage() string {
	return "AI Software Architect runtime " + version + ": use --codex-hook --event <event> or --repository-snapshot --root .; this runtime does not start a persistent server"
}

func runSnapshot() int {
	snapshot, err := architect.CreateRepositorySnapshot(".")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	data, err := architect.SerializeRepositorySnapshot(snapshot)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if _, err := os.Stdout.Write(data); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}

func readHookPayload() (map[string]any, error) {
	data, err := io.ReadAll(io.LimitReader(os.Stdin, codex.MaxHookInputBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > codex.MaxHookInputBytes {
		return nil, fmt.Errorf("hook input exceeds the bounded input size")
	}
	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, err
	}
	if payload == nil {
		return nil, fmt.Errorf("hook input must be a JSON object")
	}
	return payload, nil
}

func runHook(event string) int {
	payload, err := readHookPayload()
	pluginData := os.Getenv("PLUGIN_DATA")
	if pluginData == "" {
		err = fmt.Errorf("PLUGIN_DATA is unavailable")
	}
	pluginRoot := os.Getenv("PLUGIN_ROOT")
	var response map[string]any
	if err == nil {
		response, err = codex.HandleHook(payload, pluginData, pluginRoot, event)
	}
	if err != nil {
		diagnostic := ""
		if os.Getenv("AI_ARCHITECT_HOOK_DEBUG") == "1" {
			diagnostic = fmt.Sprintf(" Diagnostic: %T: %v", err, err)
		}
		response = codex.RuntimeFailureResponse(event, diagnostic)
	}
	data, marshalErr := json.Marshal(response)
	if marshalErr != nil {
		fmt.Fprintln(os.Stderr, marshalErr)
		return 1
	}
	if _, writeErr := os.Stdout.Write(data); writeErr != nil {
		fmt.Fprintln(os.Stderr, writeErr)
		return 1
	}
	return 0
}

func main() {
	arguments := os.Args[1:]
	if len(arguments) == 3 && arguments[0] == "--repository-snapshot" && arguments[1] == "--root" && arguments[2] == "." {
		os.Exit(runSnapshot())
	}
	if len(arguments) == 3 && arguments[0] == "--codex-hook" && arguments[1] == "--event" && hookEvents[arguments[2]] {
		os.Exit(runHook(arguments[2]))
	}
	fmt.Fprintln(os.Stderr, usage())
	os.Exit(2)
}
