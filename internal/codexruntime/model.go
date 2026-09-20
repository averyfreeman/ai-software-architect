package codexruntime

import "time"

const (
	MaxHookInputBytes       = 1_000_000
	MaxStateAge             = 24 * time.Hour
	MaxContinuationAge      = time.Hour
	MaxStateFiles           = 512
	MaxSerializedSnapshot   = 131_072
	MaxSnapshotDirectories  = 200
	MaxSnapshotEntries      = 2_000
	MaxSnapshotCandidates   = 200
	MaxSnapshotFiles        = 40
	MaxSnapshotFileBytes    = 16_384
	MaxSnapshotTotalBytes   = 65_536
	MaxSnapshotDepth        = 6
	SnapshotSchemaVersion   = "1.0.0"
	ArchitectureWorkflow    = "architecture_workflow"
	MissingSkillInvocation  = "missing_skill_invocation"
	InactiveRoute           = "inactive"
	CheckpointActive        = "active"
	CheckpointClarify       = "clarify"
	CheckpointAwaitDecision = "await_decision"
	CheckpointDecision      = "decision_response"
	CheckpointComplete      = "complete"
)

var RequiredEvents = [...]string{
	"UserPromptSubmit",
	"PreToolUse",
	"PostToolUse",
	"PostCompact",
	"Stop",
}

type Options struct {
	PluginData string
	PluginRoot string
	Executable string
	Now        func() time.Time
}

type turnContext struct {
	Active         bool     `json:"active"`
	Route          string   `json:"route"`
	ReferencePaths []string `json:"reference_paths,omitempty"`
}

type checkpoint struct {
	Phase               string   `json:"phase"`
	ExpectedArtifacts   []string `json:"expected_artifacts,omitempty"`
	ArtifactBundleValid bool     `json:"artifact_bundle_validated,omitempty"`
}

type continuation struct {
	Context            turnContext `json:"context"`
	Interaction        string      `json:"interaction"`
	Phase              string      `json:"phase"`
	ApprovalTransition string      `json:"approval_transition"`
}

type hookEvent struct {
	Name           string
	SessionID      string
	TurnID         string
	CWD            string
	Prompt         string
	ToolName       any
	ToolInput      any
	LastAssistant  string
	StopHookActive bool
}

type artifactCandidate struct {
	Path      string
	Content   string
	Kind      string
	Operation string
}
