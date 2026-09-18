package codex

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

const (
	MaxHookInputBytes      = 1_000_000
	MaxStateAgeSeconds     = 86_400
	MaxContinuationSeconds = 3_600
	MaxStateFiles          = 512
)

type TurnRoute string

const (
	RouteInactive               TurnRoute = "inactive"
	RouteMissingSkillInvocation TurnRoute = "missing_skill_invocation"
	RouteArchitectureWorkflow   TurnRoute = "architecture_workflow"
)

type TurnContext struct {
	Active         bool      `json:"active"`
	Route          TurnRoute `json:"route"`
	ReferencePaths []string  `json:"reference_paths"`
}

type PendingInteraction string

const (
	InteractionClarification PendingInteraction = "clarification"
	InteractionDecision      PendingInteraction = "decision"
)

type WorkflowPhase string

const (
	PhaseClarify WorkflowPhase = "clarify"
	PhaseApprove WorkflowPhase = "approve"
)

type ApprovalTransition string

const (
	TransitionResumeDesign     ApprovalTransition = "resume_design"
	TransitionRecordAndHandoff ApprovalTransition = "record_and_handoff"
)

type SessionContinuation struct {
	Context            TurnContext        `json:"context"`
	Interaction        PendingInteraction `json:"interaction"`
	Phase              WorkflowPhase      `json:"phase"`
	ApprovalTransition ApprovalTransition `json:"approval_transition"`
}

type CheckpointPhase string

const (
	CheckpointActive           CheckpointPhase = "active"
	CheckpointClarify          CheckpointPhase = "clarify"
	CheckpointAwaitDecision    CheckpointPhase = "await_decision"
	CheckpointDecisionResponse CheckpointPhase = "decision_response"
	CheckpointRecordAndHandoff CheckpointPhase = "record_and_handoff"
	CheckpointComplete         CheckpointPhase = "complete"
)

type WorkflowCheckpoint struct {
	Phase                   CheckpointPhase `json:"phase"`
	ExpectedArtifacts       []string        `json:"expected_artifacts,omitempty"`
	ArtifactBundleValidated bool            `json:"artifact_bundle_validated,omitempty"`
}

func stateDigest(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func atomicWriteJSON(path string, value any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	temporary := path + ".tmp-" + stateDigest(time.Now().UTC().String())
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if err := os.WriteFile(temporary, data, 0o600); err != nil {
		return err
	}
	if err := os.Rename(temporary, path); err == nil {
		return nil
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		_ = os.Remove(temporary)
		return err
	}
	if err := os.Rename(temporary, path); err != nil {
		_ = os.Remove(temporary)
		return err
	}
	return nil
}

func controlPlaneRoot(pluginData string) string {
	return filepath.Join(pluginData, "control-plane")
}

func turnStatePath(pluginData, sessionID, turnID string) string {
	return filepath.Join(controlPlaneRoot(pluginData), "turn-"+stateDigest(sessionID+"\x00"+turnID)+".json")
}

func continuationPath(pluginData, sessionID string) string {
	return filepath.Join(controlPlaneRoot(pluginData), "continuation-"+stateDigest(sessionID)+".json")
}

func checkpointPath(pluginData, sessionID string) string {
	return filepath.Join(controlPlaneRoot(pluginData), "workflow-"+stateDigest(sessionID)+".json")
}

func saveTurnContext(pluginData, sessionID, turnID string, context TurnContext) error {
	if err := atomicWriteJSON(turnStatePath(pluginData, sessionID, turnID), context); err != nil {
		return err
	}
	cleanupStaleContexts(pluginData, time.Now())
	return nil
}

func readTurnContext(pluginData, sessionID, turnID string) TurnContext {
	data, err := os.ReadFile(turnStatePath(pluginData, sessionID, turnID))
	if err != nil {
		return TurnContext{Active: false, Route: RouteInactive, ReferencePaths: []string{}}
	}
	var context TurnContext
	if json.Unmarshal(data, &context) != nil || context.Route == "" {
		return TurnContext{Active: false, Route: RouteInactive, ReferencePaths: []string{}}
	}
	return context
}

func removeTurnContext(pluginData, sessionID, turnID string) {
	_ = os.Remove(turnStatePath(pluginData, sessionID, turnID))
}

func openContinuation(pluginData, sessionID string, continuation SessionContinuation) error {
	return atomicWriteJSON(continuationPath(pluginData, sessionID), continuation)
}

func consumeContinuation(pluginData, sessionID string) *SessionContinuation {
	path := continuationPath(pluginData, sessionID)
	info, statErr := os.Stat(path)
	data, err := os.ReadFile(path)
	_ = os.Remove(path)
	if err != nil {
		return nil
	}
	if statErr == nil && time.Since(info.ModTime()) > MaxContinuationSeconds*time.Second {
		return nil
	}
	var continuation SessionContinuation
	if json.Unmarshal(data, &continuation) != nil || continuation.Context.Route == "" {
		return nil
	}
	return &continuation
}

func cancelContinuation(pluginData, sessionID string) {
	_ = os.Remove(continuationPath(pluginData, sessionID))
}

func saveCheckpoint(pluginData, sessionID string, checkpoint WorkflowCheckpoint) error {
	return atomicWriteJSON(checkpointPath(pluginData, sessionID), checkpoint)
}

func loadCheckpoint(pluginData, sessionID string) *WorkflowCheckpoint {
	data, err := os.ReadFile(checkpointPath(pluginData, sessionID))
	if err != nil {
		return nil
	}
	var checkpoint WorkflowCheckpoint
	if json.Unmarshal(data, &checkpoint) != nil || checkpoint.Phase == "" {
		return nil
	}
	return &checkpoint
}

func cleanupStaleContexts(pluginData string, now time.Time) {
	root := controlPlaneRoot(pluginData)
	entries, err := os.ReadDir(root)
	if err != nil {
		return
	}
	type fileInfo struct {
		path     string
		modified time.Time
	}
	files := []fileInfo{}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		info, err := entry.Info()
		if err == nil {
			files = append(files, fileInfo{path: filepath.Join(root, entry.Name()), modified: info.ModTime()})
		}
	}
	for index := range files {
		for next := index + 1; next < len(files); next++ {
			if files[next].modified.After(files[index].modified) {
				files[index], files[next] = files[next], files[index]
			}
		}
	}
	for index, file := range files {
		if index >= MaxStateFiles || now.Sub(file.modified) > MaxStateAgeSeconds*time.Second {
			_ = os.Remove(file.path)
		}
	}
}
