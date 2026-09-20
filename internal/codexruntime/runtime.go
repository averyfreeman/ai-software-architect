package codexruntime

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func parseEvent(payload map[string]any, expected string) (hookEvent, error) {
	eventName, ok := payload["hook_event_name"].(string)
	if !ok || eventName == "" {
		return hookEvent{}, errors.New("hook_event_name is unavailable")
	}
	if expected != "" && eventName != expected {
		return hookEvent{}, errors.New("hook event does not match the invoked hook entry point")
	}
	sessionID, ok := payload["session_id"].(string)
	if !ok || sessionID == "" {
		return hookEvent{}, errors.New("hook session_id is unavailable")
	}
	turnID := ""
	if value, present := payload["turn_id"]; present && value != nil {
		var valid bool
		turnID, valid = value.(string)
		if !valid {
			return hookEvent{}, errors.New("hook turn_id is invalid")
		}
	}
	if eventName != "PostCompact" && turnID == "" {
		return hookEvent{}, errors.New("hook turn_id is unavailable")
	}
	cwd := ""
	if value, present := payload["cwd"]; present && value != nil {
		var valid bool
		cwd, valid = value.(string)
		if !valid {
			return hookEvent{}, errors.New("hook cwd is invalid")
		}
	}
	event := hookEvent{
		Name:      eventName,
		SessionID: sessionID,
		TurnID:    turnID,
		CWD:       cwd,
		ToolName:  payload["tool_name"],
		ToolInput: payload["tool_input"],
	}
	switch eventName {
	case "UserPromptSubmit":
		event.Prompt, _ = payload["prompt"].(string)
	case "Stop":
		event.LastAssistant, _ = payload["last_assistant_message"].(string)
		if value, present := payload["stop_hook_active"]; present && value != nil {
			var valid bool
			event.StopHookActive, valid = value.(bool)
			if !valid {
				return hookEvent{}, errors.New("hook stop_hook_active is invalid")
			}
		}
	case "PreToolUse", "PostToolUse", "PostCompact":
	default:
		return hookEvent{}, fmt.Errorf("unsupported hook event: %s", eventName)
	}
	return event, nil
}

func HandleHook(payload map[string]any, options Options, expected string) (map[string]any, error) {
	if options.PluginData == "" {
		return nil, errors.New("PLUGIN_DATA is unavailable")
	}
	event, err := parseEvent(payload, expected)
	if err != nil {
		return nil, err
	}
	store := newStateStore(options)
	switch event.Name {
	case "UserPromptSubmit":
		return handleUserPrompt(event, options, store)
	case "PreToolUse":
		return handlePreTool(event, options, store)
	case "PostToolUse":
		return handlePostTool(event, options, store)
	case "PostCompact":
		return handlePostCompact(event, store)
	case "Stop":
		return handleStop(event, store)
	default:
		return nil, fmt.Errorf("unsupported hook event: %s", event.Name)
	}
}

func handleUserPrompt(event hookEvent, options Options, store stateStore) (map[string]any, error) {
	context := withReferenceHints(classifyPrompt(event.Prompt), event.Prompt, options.PluginRoot)
	continued := false
	interaction := ""
	continuationInstruction := ""
	if !context.Active {
		if hasOtherActivation(event.Prompt) {
			store.cancelContinuation(event.SessionID)
			return map[string]any{}, nil
		}
		pending, ok := store.consumeContinuation(event.SessionID)
		if !ok {
			return map[string]any{}, nil
		}
		context = pending.Context
		continued = true
		interaction = pending.Interaction
		continuationInstruction = "Preserve the immediately preceding request and do not rediscover unrelated context."
	}
	if context.Route == MissingSkillInvocation {
		store.cancelContinuation(event.SessionID)
		store.cleanup()
		return blockResponse(missingInvocationGuidance), nil
	}
	if !continued {
		store.cancelContinuation(event.SessionID)
	}
	nextCheckpoint := checkpoint{Phase: CheckpointActive}
	if continued && interaction == "decision" {
		nextCheckpoint = checkpoint{
			Phase:               CheckpointDecision,
			ExpectedArtifacts:   []string{"adr", "contract", "context", "implementation-plan"},
			ArtifactBundleValid: false,
		}
	}
	if err := store.saveCheckpoint(event.SessionID, nextCheckpoint); err != nil {
		return nil, err
	}
	if event.TurnID == "" {
		return nil, errors.New("hook turn_id is unavailable")
	}
	if err := store.saveContext(event.SessionID, event.TurnID, context); err != nil {
		return nil, err
	}
	store.cleanup()
	additional := developerContext(context, options, continued, continuationInstruction, interaction)
	if bundled := bundledReferenceContent(context, options.PluginRoot); bundled != "" {
		additional += "\n\n<bundled-architecture-reference>\n" + bundled + "\n</bundled-architecture-reference>"
	}
	return contextResponse("UserPromptSubmit", additional), nil
}

func handlePreTool(event hookEvent, options Options, store stateStore) (map[string]any, error) {
	context := store.loadContext(event.SessionID, event.TurnID)
	if !context.Active {
		return map[string]any{}, nil
	}
	workspace := event.CWD
	reason := toolDenialReason(context, event.ToolName, event.ToolInput, workspace, options)
	if reason == "" && isPatchTool(event.ToolName) {
		if workspace == "" {
			reason = "AI Software Architect denied the architecture artifact write because Codex did not provide a trustworthy workspace root."
		} else {
			currentCheckpoint, hasCheckpoint := store.loadCheckpoint(event.SessionID)
			requireComplete := hasCheckpoint && currentCheckpoint.Phase == CheckpointDecision
			reason = artifactDenialReason(event.ToolInput, workspace, requireComplete)
		}
	}
	if reason == "" {
		return map[string]any{}, nil
	}
	return denyResponse(reason), nil
}

func handlePostTool(event hookEvent, options Options, store stateStore) (map[string]any, error) {
	context := store.loadContext(event.SessionID, event.TurnID)
	if !context.Active || !isPatchTool(event.ToolName) {
		return map[string]any{}, nil
	}
	if event.CWD == "" {
		return stopResponse("AI Software Architect stopped because Codex did not provide a trustworthy workspace root for post-write verification.", "Architecture artifact post-write verification failed."), nil
	}
	candidates, err := proposedCandidates(event.ToolInput, event.CWD)
	if err != nil || len(candidates) == 0 {
		if err == nil {
			err = errors.New("no reconstructable architecture artifact candidate")
		}
		return stopResponse("AI Software Architect stopped because post-write verification failed: "+err.Error()+". Review the repository before continuing.", "Architecture artifact post-write verification failed."), nil
	}
	for _, candidate := range candidates {
		persisted, readErr := os.ReadFile(filepath.Join(event.CWD, filepath.FromSlash(candidate.Path)))
		if readErr != nil || string(persisted) != candidate.Content {
			return stopResponse("AI Software Architect stopped because post-write verification failed: persisted content differs from validated candidate. Review the repository before continuing.", "Architecture artifact post-write verification failed."), nil
		}
	}
	if reason := artifactDenialReason(event.ToolInput, event.CWD, false); reason != "" {
		return stopResponse("AI Software Architect stopped because post-write verification failed: "+reason, "Architecture artifact post-write verification failed."), nil
	}
	if completeArtifactSet(candidates) {
		next := checkpoint{
			Phase:               CheckpointComplete,
			ExpectedArtifacts:   []string{"adr", "contract", "context", "implementation-plan"},
			ArtifactBundleValid: true,
		}
		if err := store.saveCheckpoint(event.SessionID, next); err != nil {
			return nil, err
		}
		return contextResponse("PostToolUse", "The persisted ADR, architecture contract, project context, and coding handoff exactly match the pre-write validated bundle. State that the architecture artifacts were recorded and that application source code was not authorized by this workflow."), nil
	}
	return map[string]any{}, nil
}

func handlePostCompact(event hookEvent, store stateStore) (map[string]any, error) {
	current, ok := store.loadCheckpoint(event.SessionID)
	if !ok || current.Phase == CheckpointComplete {
		return map[string]any{}, nil
	}
	expected := strings.Join(current.ExpectedArtifacts, ", ")
	if expected == "" {
		expected = "none"
	}
	return contextResponse("PostCompact", "AI Software Architect typed workflow checkpoint: phase="+current.Phase+"; expected_artifacts="+expected+"; artifact_bundle_validated=false. Preserve the current approval and read-only boundaries; do not reconstruct workflow state from memory."), nil
}

func handleStop(event hookEvent, store stateStore) (map[string]any, error) {
	context := store.loadContext(event.SessionID, event.TurnID)
	if !context.Active {
		return map[string]any{}, nil
	}
	if event.StopHookActive {
		persistPending(event, context, store)
		store.clearContext(event.SessionID, event.TurnID)
		return map[string]any{}, nil
	}
	if violations := responseViolations(event.LastAssistant); len(violations) > 0 {
		return blockResponse("The AI Software Architect response did not pass its deterministic user-facing contract. Return a complete standalone replacement response that preserves all already-valid content; do not return an addendum or only the missing sentence. Correct these issues: " + strings.Join(violations, "; ") + "."), nil
	}
	persistPending(event, context, store)
	store.clearContext(event.SessionID, event.TurnID)
	return map[string]any{}, nil
}

func persistPending(event hookEvent, context turnContext, store stateStore) {
	interaction, ok := pendingInteraction(event.LastAssistant)
	if !ok {
		store.cancelContinuation(event.SessionID)
		return
	}
	value := continuation{
		Context:            context,
		Interaction:        interaction,
		Phase:              "approve",
		ApprovalTransition: "record_and_handoff",
	}
	if interaction == "clarification" {
		value.Phase = "clarify"
		value.ApprovalTransition = "resume_design"
	}
	_ = store.saveContinuation(event.SessionID, value)
	if interaction == "decision" {
		_ = store.saveCheckpoint(event.SessionID, checkpoint{Phase: CheckpointAwaitDecision})
	} else {
		_ = store.saveCheckpoint(event.SessionID, checkpoint{Phase: CheckpointClarify})
	}
}

func pendingInteraction(message string) (string, bool) {
	lower := strings.ToLower(message)
	if strings.Contains(lower, "## your decision") || strings.Contains(lower, "## sua decisão") || strings.Contains(lower, "## tu decisión") {
		return "decision", true
	}
	if strings.Contains(message, "?") {
		return "clarification", true
	}
	return "", false
}

func responseViolations(message string) []string {
	violations := make([]string, 0)
	for _, marker := range []string{"ai-architect-outcome:", "ai-architect-decision-shape:", "ai-architect-actions:"} {
		if strings.Contains(strings.ToLower(message), marker) {
			violations = append(violations, "internal response marker "+marker+" is visible")
		}
	}
	return violations
}

func isPatchTool(value any) bool {
	return patchTools[normalizedToolName(value)]
}

func completeArtifactSet(candidates []artifactCandidate) bool {
	seen := map[string]bool{}
	for _, candidate := range candidates {
		seen[candidate.Kind] = true
	}
	return seen["adr"] && seen["contract"] && seen["context"] && seen["implementation-plan"]
}

func contextResponse(event, context string) map[string]any {
	return map[string]any{
		"hookSpecificOutput": map[string]any{
			"hookEventName":     event,
			"additionalContext": context,
		},
	}
}

func denyResponse(reason string) map[string]any {
	return map[string]any{
		"hookSpecificOutput": map[string]any{
			"hookEventName":            "PreToolUse",
			"permissionDecision":       "deny",
			"permissionDecisionReason": reason,
		},
	}
}

func blockResponse(reason string) map[string]any {
	return map[string]any{"decision": "block", "reason": reason}
}

func stopResponse(reason, systemMessage string) map[string]any {
	return map[string]any{"continue": false, "stopReason": reason, "systemMessage": systemMessage}
}

func FailureResponse(expectedEvent, diagnostic string) map[string]any {
	message := "AI Software Architect control-plane validation failed; the protected operation was not allowed." + diagnostic
	if expectedEvent == "PreToolUse" {
		response := denyResponse(message)
		response["systemMessage"] = message
		return response
	}
	if expectedEvent == "PostToolUse" {
		return stopResponse(message, message)
	}
	return map[string]any{"systemMessage": message}
}
