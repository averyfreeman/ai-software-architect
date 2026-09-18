package codex

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/averyfreeman/adr-repo-governance/internal/architect"
)

var validHookEvents = map[string]bool{
	"UserPromptSubmit": true,
	"PreToolUse":       true,
	"PostToolUse":      true,
	"PostCompact":      true,
	"Stop":             true,
}

func stringField(payload map[string]any, name string) string {
	value, _ := payload[name].(string)
	return value
}

func sessionID(payload map[string]any) (string, error) {
	value := stringField(payload, "session_id")
	if value == "" {
		return "", fmt.Errorf("hook session_id is unavailable")
	}
	return value, nil
}

func turnID(payload map[string]any) (string, error) {
	value := stringField(payload, "turn_id")
	if value == "" {
		return "", fmt.Errorf("hook turn_id is unavailable")
	}
	return value, nil
}

func contextForPayload(payload map[string]any, pluginData string) TurnContext {
	session, err := sessionID(payload)
	if err != nil {
		return TurnContext{Active: false, Route: RouteInactive, ReferencePaths: []string{}}
	}
	turn, err := turnID(payload)
	if err != nil {
		return TurnContext{Active: false, Route: RouteInactive, ReferencePaths: []string{}}
	}
	return readTurnContext(pluginData, session, turn)
}

func pendingContinuation(message string, context TurnContext) *SessionContinuation {
	visible := strings.TrimSpace(message)
	if containsHeading(message, 5) {
		return &SessionContinuation{
			Context: context, Interaction: InteractionDecision, Phase: PhaseApprove,
			ApprovalTransition: TransitionRecordAndHandoff,
		}
	}
	if strings.HasSuffix(visible, "?") {
		return &SessionContinuation{
			Context: context, Interaction: InteractionClarification, Phase: PhaseClarify,
			ApprovalTransition: TransitionResumeDesign,
		}
	}
	return nil
}

func continuationInstruction(continuation *SessionContinuation) string {
	if continuation == nil {
		return ""
	}
	if continuation.Interaction == InteractionDecision {
		return "The preceding response requested a decision. Interpret the user's reply host-natively and in any language. If the user approves, do not merely acknowledge approval: transition to record_and_handoff and safely create and validate the approved ADR, architecture contract, context, and coding handoff when this is a project-bound material decision. Architecture artifact writes under .ai-architect are authorized by approval unless the original request explicitly prohibited creating or modifying files. Approval never authorizes application-code changes. If the original turn was read-only or projectless, preserve that restriction and plainly explain why artifacts were not persisted. If the user revises or rejects the proposal, return to design and persist nothing."
	}
	return "The preceding response requested clarification. Interpret this reply as the answer, retain the clarified constraints, and resume the smallest sufficient architecture workflow without requiring another skill invocation."
}

func resetCheckpoint(pluginData, session string, phase CheckpointPhase, expected []string) error {
	return saveCheckpoint(pluginData, session, WorkflowCheckpoint{
		Phase: phase, ExpectedArtifacts: expected,
	})
}

func handleUserPromptSubmit(payload map[string]any, pluginData, pluginRoot string) (map[string]any, error) {
	session, err := sessionID(payload)
	if err != nil {
		return nil, err
	}
	turn, err := turnID(payload)
	if err != nil {
		return nil, err
	}
	prompt := stringField(payload, "prompt")
	context := withReferenceHints(classifyPrompt(prompt), prompt)
	var continuation *SessionContinuation
	if !context.Active {
		if activationPattern.MatchString(prompt) || strings.Contains(strings.ToLower(prompt), PluginSelectionURI) {
			cancelContinuation(pluginData, session)
			return map[string]any{}, nil
		}
		continuation = consumeContinuation(pluginData, session)
		if continuation == nil {
			return map[string]any{}, nil
		}
		context = continuation.Context
	}
	if context.Route == RouteMissingSkillInvocation {
		cancelContinuation(pluginData, session)
		cleanupStaleContexts(pluginData, time.Now())
		return map[string]any{"decision": "block", "reason": MissingInvocationGuidance}, nil
	}
	if continuation == nil {
		cancelContinuation(pluginData, session)
		if err := resetCheckpoint(pluginData, session, CheckpointActive, nil); err != nil {
			return nil, err
		}
	} else if continuation.Interaction == InteractionDecision {
		if err := resetCheckpoint(pluginData, session, CheckpointDecisionResponse, []string{"adr", "contract", "context", "implementation-plan"}); err != nil {
			return nil, err
		}
	} else if err := resetCheckpoint(pluginData, session, CheckpointActive, nil); err != nil {
		return nil, err
	}
	if err := saveTurnContext(pluginData, session, turn, context); err != nil {
		return nil, err
	}
	additional := developerContext(context, continuation != nil, continuationInstruction(continuation), func() PendingInteraction {
		if continuation == nil {
			return ""
		}
		return continuation.Interaction
	}(), pluginRoot)
	if continuation != nil && continuation.Interaction == InteractionDecision && pluginRoot != "" {
		resource := filepath.Join(pluginRoot, "skills", "ai-software-architect", "assets", "artifact-authoring-bundle.md")
		additional += " Exact installed record-and-handoff resource path (authoritative; read this generated bundle once with a host-native static file tool and do not search for or separately load its canonical source files): " + resource + "."
	}
	return map[string]any{"hookSpecificOutput": map[string]any{
		"hookEventName":     "UserPromptSubmit",
		"additionalContext": additional,
	}}, nil
}

func workspaceField(payload map[string]any) string {
	value := stringField(payload, "cwd")
	if value == "" {
		return ""
	}
	absolute, err := filepath.Abs(value)
	if err != nil {
		return ""
	}
	return absolute
}

func architectureArtifactDenial(toolInput any, workspace string, requireComplete bool) string {
	candidates, err := architect.ProposedArtifactCandidates(toolInput, workspace)
	if err != nil {
		return "AI Software Architect could not reconstruct the complete proposed architecture artifact safely: " + err.Error() + ". Use one complete reviewable write."
	}
	if len(candidates) == 0 {
		return "AI Software Architect could not identify a complete architecture artifact candidate, so the write was denied. Use one reviewable canonical write."
	}
	for _, candidate := range candidates {
		if strings.TrimSpace(candidate.Content) == "" {
			return "AI Software Architect blocked empty artifact " + candidate.Path + "."
		}
		scan := architect.ScanGeneratedArtifact(candidate.Content)
		if !scan.SafeToWrite {
			locations := []string{}
			for index, finding := range scan.Findings {
				if index >= 5 {
					break
				}
				locations = append(locations, fmt.Sprintf("%s at line %d", finding.Category, finding.Line))
			}
			return "AI Software Architect blocked " + candidate.Path + " because the pre-write secret scan reported " + strings.Join(locations, ", ") + "."
		}
		switch candidate.Kind {
		case architect.ArtifactContract:
			result, _ := architect.ValidateContractYAML(candidate.Content)
			if !result.Valid {
				return "AI Software Architect blocked " + candidate.Path + " because the complete architecture contract is invalid: " + strings.Join(result.Errors, "; ") + ". Before retrying, reload artifact-authoring-bundle.md from the installed composite skill and preserve the contract example's nested object shapes."
			}
		case architect.ArtifactADR:
			artifact, err := architect.ParseADRFrontmatter(candidate.Content)
			if err != nil {
				return "AI Software Architect blocked " + candidate.Path + " because its complete ADR frontmatter is invalid: " + err.Error() + "."
			}
			filename := strings.TrimSuffix(filepath.Base(candidate.Path), filepath.Ext(candidate.Path))
			filenameID := filename
			if dash := strings.Index(filenameID, "-"); dash >= 0 {
				if second := strings.Index(filenameID[dash+1:], "-"); second >= 0 {
					filenameID = filenameID[:dash+1+second]
				}
			}
			if filenameID != artifact.Decision.ID {
				return "AI Software Architect blocked " + candidate.Path + " because the ADR filename and decision id disagree."
			}
		}
	}
	if _, err := architect.ValidateArtifactBundleCandidates(candidates, requireComplete); err != nil {
		return "AI Software Architect blocked the record-and-handoff write because the complete artifact bundle is inconsistent: " + err.Error() + ". Load the exact bundled ADR, contract, context, and handoff templates, then submit all four artifact types together in one reviewable write."
	}
	return ""
}

func handlePreToolUse(payload map[string]any, pluginData, pluginRoot string) (map[string]any, error) {
	context := contextForPayload(payload, pluginData)
	workspace := workspaceField(payload)
	reason := toolDenialReason(context, payload["tool_name"], payload["tool_input"], workspace, pluginRoot)
	if reason == "" && context.Active {
		name := normalizeToolName(payload["tool_name"])
		if name == "apply_patch" || name == "edit" || name == "write" {
			if workspace == "" {
				reason = "AI Software Architect denied the architecture artifact write because Codex did not provide a trustworthy workspace root."
			} else {
				session, sessionErr := sessionID(payload)
				if sessionErr != nil {
					return nil, sessionErr
				}
				checkpoint := loadCheckpoint(pluginData, session)
				requireComplete := checkpoint != nil && checkpoint.Phase == CheckpointDecisionResponse
				reason = architectureArtifactDenial(payload["tool_input"], workspace, requireComplete)
			}
		}
	}
	if reason == "" {
		return map[string]any{}, nil
	}
	return map[string]any{"hookSpecificOutput": map[string]any{
		"hookEventName":            "PreToolUse",
		"permissionDecision":       "deny",
		"permissionDecisionReason": reason,
	}}, nil
}

func handlePostToolUse(payload map[string]any, pluginData string) (map[string]any, error) {
	context := contextForPayload(payload, pluginData)
	if !context.Active {
		return map[string]any{}, nil
	}
	name := normalizeToolName(payload["tool_name"])
	if name != "apply_patch" && name != "edit" && name != "write" {
		return map[string]any{}, nil
	}
	workspace := workspaceField(payload)
	if workspace == "" {
		return map[string]any{"continue": false, "stopReason": "AI Software Architect stopped because Codex did not provide a trustworthy workspace root for post-write verification.", "systemMessage": "Architecture artifact post-write verification failed."}, nil
	}
	candidates, err := architect.ProposedArtifactCandidates(payload["tool_input"], workspace)
	if err == nil && len(candidates) == 0 {
		err = fmt.Errorf("no reconstructable architecture artifact candidate")
	}
	if err == nil {
		for _, candidate := range candidates {
			persisted, readErr := os.ReadFile(filepath.Join(workspace, filepath.FromSlash(candidate.Path)))
			if readErr != nil {
				err = readErr
				break
			}
			if string(persisted) != candidate.Content {
				err = fmt.Errorf("persisted content differs from validated candidate: %s", candidate.Path)
				break
			}
		}
	}
	var bundle *architect.ArtifactBundle
	if err == nil {
		bundle, err = architect.ValidateArtifactBundleCandidates(candidates, false)
	}
	if err != nil {
		return map[string]any{"continue": false, "stopReason": "AI Software Architect stopped because post-write verification failed: " + err.Error() + ". Review the repository before continuing.", "systemMessage": "Architecture artifact post-write verification failed."}, nil
	}
	if bundle == nil {
		return map[string]any{}, nil
	}
	session, err := sessionID(payload)
	if err != nil {
		return nil, err
	}
	if err := saveCheckpoint(pluginData, session, WorkflowCheckpoint{Phase: CheckpointComplete, ExpectedArtifacts: []string{"adr", "contract", "context", "implementation-plan"}, ArtifactBundleValidated: true}); err != nil {
		return nil, err
	}
	return map[string]any{"hookSpecificOutput": map[string]any{
		"hookEventName":     "PostToolUse",
		"additionalContext": "The persisted ADR, architecture contract, project context, and coding handoff exactly match the pre-write validated bundle. State that the architecture artifacts were recorded and that application source code was not authorized by this workflow.",
	}}, nil
}

func handlePostCompact(payload map[string]any, pluginData string) (map[string]any, error) {
	session, err := sessionID(payload)
	if err != nil {
		return nil, err
	}
	checkpoint := loadCheckpoint(pluginData, session)
	if checkpoint == nil || checkpoint.Phase == CheckpointComplete {
		return map[string]any{}, nil
	}
	expected := strings.Join(checkpoint.ExpectedArtifacts, ", ")
	if expected == "" {
		expected = "none"
	}
	return map[string]any{"hookSpecificOutput": map[string]any{
		"hookEventName":     "PostCompact",
		"additionalContext": fmt.Sprintf("AI Software Architect typed workflow checkpoint: phase=%s; expected_artifacts=%s; artifact_bundle_validated=%t. Preserve the current approval and read-only boundaries; do not reconstruct workflow state from memory.", checkpoint.Phase, expected, checkpoint.ArtifactBundleValidated),
	}}, nil
}

func handleStop(payload map[string]any, pluginData string) (map[string]any, error) {
	context := contextForPayload(payload, pluginData)
	if !context.Active {
		return map[string]any{}, nil
	}
	session, err := sessionID(payload)
	if err != nil {
		return nil, err
	}
	message := stringField(payload, "last_assistant_message")
	if active, ok := payload["stop_hook_active"].(bool); ok && active {
		pending := pendingContinuation(message, context)
		if pending != nil {
			if err := openContinuation(pluginData, session, *pending); err != nil {
				return nil, err
			}
			phase := CheckpointClarify
			if pending.Interaction == InteractionDecision {
				phase = CheckpointAwaitDecision
			}
			if err := resetCheckpoint(pluginData, session, phase, nil); err != nil {
				return nil, err
			}
		} else {
			cancelContinuation(pluginData, session)
		}
		turn, turnErr := turnID(payload)
		if turnErr == nil {
			removeTurnContext(pluginData, session, turn)
		}
		return map[string]any{}, nil
	}
	violations := finalResponseViolations(context, message)
	if len(violations) > 0 {
		return map[string]any{"decision": "block", "reason": "The AI Software Architect response did not pass its deterministic user-facing contract. Return a complete standalone replacement response that preserves all already-valid content; do not return an addendum or only the missing sentence. Correct these issues: " + strings.Join(violations, "; ") + "."}, nil
	}
	turn, turnErr := turnID(payload)
	if turnErr == nil {
		removeTurnContext(pluginData, session, turn)
	}
	if pending := pendingContinuation(message, context); pending != nil {
		if err := openContinuation(pluginData, session, *pending); err != nil {
			return nil, err
		}
		phase := CheckpointClarify
		if pending.Interaction == InteractionDecision {
			phase = CheckpointAwaitDecision
		}
		if err := resetCheckpoint(pluginData, session, phase, nil); err != nil {
			return nil, err
		}
	} else {
		cancelContinuation(pluginData, session)
	}
	return map[string]any{}, nil
}

func HandleHook(payload map[string]any, pluginData, pluginRoot, expectedEvent string) (map[string]any, error) {
	event, ok := payload["hook_event_name"].(string)
	if !ok || !validHookEvents[event] {
		return nil, fmt.Errorf("hook_event_name is unavailable or unsupported")
	}
	if expectedEvent != "" && event != expectedEvent {
		return nil, fmt.Errorf("hook event does not match the invoked hook entry point")
	}
	if _, err := sessionID(payload); err != nil {
		return nil, err
	}
	switch event {
	case "UserPromptSubmit":
		return handleUserPromptSubmit(payload, pluginData, pluginRoot)
	case "PreToolUse":
		return handlePreToolUse(payload, pluginData, pluginRoot)
	case "PostToolUse":
		return handlePostToolUse(payload, pluginData)
	case "PostCompact":
		return handlePostCompact(payload, pluginData)
	case "Stop":
		return handleStop(payload, pluginData)
	default:
		return nil, fmt.Errorf("unsupported hook event")
	}
}

func RuntimeFailureResponse(expectedEvent, diagnostic string) map[string]any {
	message := "AI Software Architect control-plane validation failed; the protected operation was not allowed." + diagnostic
	switch expectedEvent {
	case "PreToolUse":
		return map[string]any{"hookSpecificOutput": map[string]any{"hookEventName": "PreToolUse", "permissionDecision": "deny", "permissionDecisionReason": message}, "systemMessage": message}
	case "PostToolUse":
		return map[string]any{"continue": false, "stopReason": message, "systemMessage": message}
	default:
		return map[string]any{"systemMessage": message}
	}
}
