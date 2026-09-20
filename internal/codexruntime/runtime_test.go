package codexruntime

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func hookPayload(event string) map[string]any {
	return map[string]any{
		"hook_event_name": event,
		"session_id":      "session-1",
		"turn_id":         "turn-1",
		"cwd":             "",
	}
}

func TestGoRuntimeActivatesAndPersistsMinimalState(t *testing.T) {
	data := t.TempDir()
	payload := hookPayload("UserPromptSubmit")
	payload["prompt"] = "$ai-software-architect Review this repository."
	response, err := HandleHook(payload, Options{PluginData: data}, "UserPromptSubmit")
	if err != nil {
		t.Fatal(err)
	}
	output, ok := response["hookSpecificOutput"].(map[string]any)
	if !ok || !strings.Contains(output["additionalContext"].(string), "smallest sufficient workflow") {
		t.Fatalf("activation response = %#v", response)
	}
	entries, err := os.ReadDir(filepath.Join(data, "control-plane"))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("control-plane entries = %d, want context and checkpoint", len(entries))
	}
	for _, entry := range entries {
		if strings.Contains(entry.Name(), "session-1") {
			t.Fatalf("state filename leaked identity: %s", entry.Name())
		}
	}
}

func TestGoRuntimeActivationMarkersAndPackagedReferences(t *testing.T) {
	selection := classifyPrompt("[@AI Software Architect](plugin://ai-software-architect@personal)")
	if !selection.Active || selection.Route != MissingSkillInvocation {
		t.Fatalf("selection-only prompt = %#v", selection)
	}
	request := classifyPrompt("[@AI Software Architect](plugin://ai-software-architect) explain Observer pattern")
	if !request.Active || request.Route != ArchitectureWorkflow {
		t.Fatalf("structured request = %#v", request)
	}
	if !classifyPrompt("$ai-software-architect explain Observer pattern").Active {
		t.Fatal("direct skill marker did not activate")
	}
	if !hasOtherActivation("$unrelated-skill continue") {
		t.Fatal("competing skill marker was not detected")
	}

	pluginRoot := t.TempDir()
	references := filepath.Join(pluginRoot, "skills", "ai-software-architect", "references")
	if err := os.MkdirAll(references, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(references, "gof-observer.md"), []byte("# Observer\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	paths := explicitReferencePaths("explain Observer pattern", pluginRoot)
	if len(paths) != 1 || paths[0] != "references/gof-observer.md" {
		t.Fatalf("packaged reference paths = %#v", paths)
	}
}

func TestGoRuntimeFailsClosedForShellAndApplicationWrites(t *testing.T) {
	data := t.TempDir()
	payload := hookPayload("UserPromptSubmit")
	payload["prompt"] = "$ai-software-architect Review this repository."
	if _, err := HandleHook(payload, Options{PluginData: data}, "UserPromptSubmit"); err != nil {
		t.Fatal(err)
	}

	shell := hookPayload("PreToolUse")
	shell["tool_name"] = "Bash"
	shell["tool_input"] = map[string]any{"command": "python -c \"print(1)\""}
	response, err := HandleHook(shell, Options{PluginData: data}, "PreToolUse")
	if err != nil {
		t.Fatal(err)
	}
	decision := response["hookSpecificOutput"].(map[string]any)
	if decision["permissionDecision"] != "deny" {
		t.Fatalf("unsafe shell response = %#v", response)
	}

	application := hookPayload("PreToolUse")
	application["tool_name"] = "apply_patch"
	application["tool_input"] = map[string]any{
		"file_path": "main.go",
		"content":   "package main\n",
	}
	response, err = HandleHook(application, Options{PluginData: data}, "PreToolUse")
	if err != nil {
		t.Fatal(err)
	}
	if response["hookSpecificOutput"] == nil {
		t.Fatalf("application write was allowed: %#v", response)
	}
}

func TestGoRuntimeDeniesInvalidArchitectureArtifactBeforeWrite(t *testing.T) {
	data := t.TempDir()
	workspace := t.TempDir()
	payload := hookPayload("UserPromptSubmit")
	payload["prompt"] = "$ai-software-architect Record the approved decision."
	if _, err := HandleHook(payload, Options{PluginData: data}, "UserPromptSubmit"); err != nil {
		t.Fatal(err)
	}
	write := hookPayload("PreToolUse")
	write["cwd"] = workspace
	write["tool_name"] = "Write"
	write["tool_input"] = map[string]any{
		"file_path": ".ai-architect/architecture-contract.yaml",
		"content":   "schema_version: invalid\n",
	}
	response, err := HandleHook(write, Options{PluginData: data}, "PreToolUse")
	if err != nil {
		t.Fatal(err)
	}
	decision := response["hookSpecificOutput"].(map[string]any)
	if decision["permissionDecision"] != "deny" {
		t.Fatalf("invalid contract response = %#v", response)
	}
}

func TestGoRuntimeCompactionRestoresTypedCheckpoint(t *testing.T) {
	data := t.TempDir()
	payload := hookPayload("UserPromptSubmit")
	payload["prompt"] = "$ai-software-architect Compare two options."
	if _, err := HandleHook(payload, Options{PluginData: data}, "UserPromptSubmit"); err != nil {
		t.Fatal(err)
	}
	compact := hookPayload("PostCompact")
	delete(compact, "turn_id")
	response, err := HandleHook(compact, Options{PluginData: data}, "PostCompact")
	if err != nil {
		t.Fatal(err)
	}
	context := response["hookSpecificOutput"].(map[string]any)["additionalContext"].(string)
	if !strings.Contains(context, "phase=active") {
		t.Fatalf("compaction response = %#v", response)
	}
}

func TestGoRuntimePostToolVerifiesPersistedArchitectureArtifact(t *testing.T) {
	data := t.TempDir()
	workspace := t.TempDir()
	if err := os.Mkdir(filepath.Join(workspace, ".ai-architect"), 0o755); err != nil {
		t.Fatal(err)
	}
	submit := hookPayload("UserPromptSubmit")
	submit["prompt"] = "$ai-software-architect Record the approved contract."
	if _, err := HandleHook(submit, Options{PluginData: data}, "UserPromptSubmit"); err != nil {
		t.Fatal(err)
	}
	content := "schema_version: 1.0.0\nrevision: 1\nscope: smoke-test\narchitecture_style: null\nquality_attributes: []\ncomponents: []\nexternal_boundaries: []\ndependency_rules: []\nrequired_practices: []\nprohibited_practices: []\ndecision_ids: []\nunresolved_questions: []\n"
	patch := "*** Begin Patch\n*** Add File: .ai-architect/architecture-contract.yaml\n+" + strings.Join(strings.Split(strings.TrimSuffix(content, "\n"), "\n"), "\n+") + "\n*** End Patch"
	pre := hookPayload("PreToolUse")
	pre["cwd"] = workspace
	pre["tool_name"] = "apply_patch"
	pre["tool_input"] = patch
	if response, err := HandleHook(pre, Options{PluginData: data}, "PreToolUse"); err != nil {
		t.Fatal(err)
	} else if len(response) != 0 {
		t.Fatalf("valid pre-write response = %#v", response)
	}
	if err := os.WriteFile(filepath.Join(workspace, ".ai-architect", "architecture-contract.yaml"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	post := hookPayload("PostToolUse")
	post["cwd"] = workspace
	post["tool_name"] = "apply_patch"
	post["tool_input"] = patch
	if response, err := HandleHook(post, Options{PluginData: data}, "PostToolUse"); err != nil {
		t.Fatal(err)
	} else if len(response) != 0 {
		t.Fatalf("valid post-write response = %#v", response)
	}
}

func TestGoRuntimeDecisionContinuationIsTypedAndSingleUse(t *testing.T) {
	data := t.TempDir()
	submit := hookPayload("UserPromptSubmit")
	submit["prompt"] = "$ai-software-architect Compare two options."
	if _, err := HandleHook(submit, Options{PluginData: data}, "UserPromptSubmit"); err != nil {
		t.Fatal(err)
	}
	stop := hookPayload("Stop")
	stop["stop_hook_active"] = false
	stop["last_assistant_message"] = "## Your decision\nApprove, revise, or request more information."
	if response, err := HandleHook(stop, Options{PluginData: data}, "Stop"); err != nil {
		t.Fatal(err)
	} else if len(response) != 0 {
		t.Fatalf("stop response = %#v", response)
	}
	continued := hookPayload("UserPromptSubmit")
	continued["turn_id"] = "turn-2"
	continued["prompt"] = "Approve."
	response, err := HandleHook(continued, Options{PluginData: data}, "UserPromptSubmit")
	if err != nil {
		t.Fatal(err)
	}
	context := response["hookSpecificOutput"].(map[string]any)["additionalContext"].(string)
	if !strings.Contains(context, "typed decision continuation") {
		t.Fatalf("continuation response = %#v", response)
	}
	if response, err := HandleHook(continued, Options{PluginData: data}, "UserPromptSubmit"); err != nil {
		t.Fatal(err)
	} else if len(response) != 0 {
		t.Fatalf("single-use continuation response = %#v", response)
	}
}

func TestGoRuntimeSnapshotIsBoundedAndExcludesSensitivePaths(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("# safe\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".env"), []byte("secret=true\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	data, err := RunSnapshot(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) > MaxSerializedSnapshot {
		t.Fatalf("snapshot bytes = %d", len(data))
	}
	var snapshot map[string]any
	if err := json.Unmarshal(data, &snapshot); err != nil {
		t.Fatal(err)
	}
	files := snapshot["files"].([]any)
	if len(files) != 1 || files[0].(map[string]any)["path"] != "README.md" {
		t.Fatalf("snapshot files = %#v", files)
	}
}

func TestGoRuntimeFailureResponseDeniesPreToolUse(t *testing.T) {
	response := FailureResponse("PreToolUse", "")
	output := response["hookSpecificOutput"].(map[string]any)
	if output["permissionDecision"] != "deny" {
		t.Fatalf("failure response = %#v", response)
	}
}
