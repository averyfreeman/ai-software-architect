// Package architect contains the host-neutral Go core for the agent-first
// project scaffold. It owns durable project contracts and deterministic checks;
// host adapters remain responsible for model interaction and permissions.
package architect

import "time"

const (
	SchemaVersion = "1.1.0"
	ToolVersion   = "0.1.0-dev"

	ConfigFilename        = ".adr-scaffold.yaml"
	GitHabitsFilename     = ".githabits.yaml"
	ArchitectDir          = ".ai-architect"
	DecisionsDir          = ".ai-architect/decisions"
	ContextFilename       = ".ai-architect/project-context.md"
	ContractFilename      = ".ai-architect/architecture-contract.yaml"
	HandoffFilename       = ".ai-architect/implementation-plan.md"
	DecisionIndexFilename = ".ai-architect/decisions/index.json"
)

// Motivation is deliberately explicit because an agent must be able to recover
// why a project or decision exists without guessing from prose.
type Motivation struct {
	WhyBuilding     string `yaml:"why_building" json:"why_building"`
	Problem         string `yaml:"problem" json:"problem"`
	Audience        string `yaml:"audience" json:"audience"`
	Alternatives    string `yaml:"alternatives" json:"alternatives"`
	WhyThisSolution string `yaml:"why_this_solution" json:"why_this_solution"`
	Source          string `yaml:"source,omitempty" json:"source,omitempty"`
}

func (m Motivation) Fields() map[string]string {
	return map[string]string{
		"why_building":      m.WhyBuilding,
		"problem":           m.Problem,
		"audience":          m.Audience,
		"alternatives":      m.Alternatives,
		"why_this_solution": m.WhyThisSolution,
	}
}

// DecisionOption is the machine-readable decision matrix row.
type DecisionOption struct {
	ID        string   `yaml:"id" json:"id"`
	Name      string   `yaml:"name" json:"name"`
	FitScore  int      `yaml:"fit_score" json:"fit_score"`
	Benefits  []string `yaml:"benefits" json:"benefits"`
	Drawbacks []string `yaml:"drawbacks" json:"drawbacks"`
	Risks     []string `yaml:"risks" json:"risks"`
	Evidence  []string `yaml:"evidence" json:"evidence"`
	Outcome   string   `yaml:"outcome" json:"outcome"`
}

type ActionContract struct {
	GoodOutcomes     []string `yaml:"good_outcomes" json:"good_outcomes"`
	BadOutcomes      []string `yaml:"bad_outcomes" json:"bad_outcomes"`
	CorrectExample   string   `yaml:"correct_example" json:"correct_example"`
	IncorrectExample string   `yaml:"incorrect_example" json:"incorrect_example"`
}

// Decision is the canonical structured decision body. The Markdown body is a
// deterministic, reviewable rendering of this object.
type Decision struct {
	ID                   string           `yaml:"id" json:"id"`
	Title                string           `yaml:"title" json:"title"`
	Date                 string           `yaml:"date" json:"date"`
	Status               string           `yaml:"status" json:"status"`
	Motivation           Motivation       `yaml:"motivation" json:"motivation"`
	Context              string           `yaml:"context" json:"context"`
	Drivers              []string         `yaml:"drivers" json:"drivers"`
	ConsideredOptionIDs  []string         `yaml:"considered_option_ids" json:"considered_option_ids"`
	SelectedOptionID     *string          `yaml:"selected_option_id" json:"selected_option_id"`
	Decision             string           `yaml:"decision" json:"decision"`
	PositiveConsequences []string         `yaml:"positive_consequences" json:"positive_consequences"`
	NegativeConsequences []string         `yaml:"negative_consequences" json:"negative_consequences"`
	Assumptions          []string         `yaml:"assumptions" json:"assumptions"`
	ValidationCriteria   []string         `yaml:"validation_criteria" json:"validation_criteria"`
	Supersedes           []string         `yaml:"supersedes" json:"supersedes"`
	DecisionMatrix       []DecisionOption `yaml:"decision_matrix" json:"decision_matrix"`
	ActionContract       ActionContract   `yaml:"action_contract" json:"action_contract"`
}

type DecisionArtifact struct {
	SchemaVersion string   `yaml:"schema_version" json:"schema_version"`
	Revision      int      `yaml:"revision" json:"revision"`
	Decision      Decision `yaml:"decision" json:"decision"`
}

type DecisionRecord struct {
	Artifact DecisionArtifact `json:"artifact"`
	Body     string           `json:"body"`
	Filename string           `json:"filename"`
	Path     string           `json:"path"`
}

type DecisionIndexEntry struct {
	ID      string   `json:"id"`
	Title   string   `json:"title"`
	Status  string   `json:"status"`
	Date    string   `json:"date,omitempty"`
	File    string   `json:"file"`
	Tags    []string `json:"tags,omitempty"`
	Options []string `json:"options,omitempty"`
}

type DecisionIndex struct {
	SchemaVersion string               `json:"schema_version"`
	GeneratedAt   time.Time            `json:"generated_at"`
	DecisionCount int                  `json:"decision_count"`
	Decisions     []DecisionIndexEntry `json:"decisions"`
}

type FeatureToggles struct {
	Agents             bool `yaml:"agents" json:"agents"`
	Skills             bool `yaml:"skills" json:"skills"`
	Claude             bool `yaml:"claude" json:"claude"`
	ADR                bool `yaml:"adr" json:"adr"`
	Context            bool `yaml:"context" json:"context"`
	Contract           bool `yaml:"contract" json:"contract"`
	ImplementationPlan bool `yaml:"implementation_plan" json:"implementation_plan"`
	Pages              bool `yaml:"pages" json:"pages"`
	Memory             bool `yaml:"memory" json:"memory"`
}

type ScaffoldPaths struct {
	Architect string `yaml:"architect" json:"architect"`
	Decisions string `yaml:"decisions" json:"decisions"`
	DocsSite  string `yaml:"docs_site" json:"docs_site"`
}

type ScaffoldConfig struct {
	Version  int             `yaml:"version" json:"version"`
	Language string          `yaml:"language" json:"language"`
	Features FeatureToggles  `yaml:"features" json:"features"`
	Paths    ScaffoldPaths   `yaml:"paths" json:"paths"`
	Defaults map[string]bool `yaml:"persistent_defaults,omitempty" json:"persistent_defaults,omitempty"`
}

type RemoteConfig struct {
	Owner      string `yaml:"owner,omitempty" json:"owner,omitempty"`
	Name       string `yaml:"name,omitempty" json:"name,omitempty"`
	Visibility string `yaml:"visibility" json:"visibility"`
}

// GitHabitsConfig contains only lifecycle and remote automation preferences.
// Scaffold content belongs in ScaffoldConfig; keeping these files separate is
// intentional so an agent can change documentation policy without changing git
// mutation policy.
type GitHabitsConfig struct {
	Version      int             `yaml:"version" json:"version"`
	Branch       string          `yaml:"branch" json:"branch"`
	Versioning   string          `yaml:"versioning" json:"versioning"`
	InitialTag   string          `yaml:"initial_tag" json:"initial_tag"`
	CommitStyle  string          `yaml:"commit_style" json:"commit_style"`
	Init         bool            `yaml:"init" json:"init"`
	Commit       bool            `yaml:"commit" json:"commit"`
	Tag          bool            `yaml:"tag" json:"tag"`
	Push         bool            `yaml:"push" json:"push"`
	CreateRemote bool            `yaml:"create_remote" json:"create_remote"`
	Remote       RemoteConfig    `yaml:"remote" json:"remote"`
	Defaults     map[string]bool `yaml:"persistent_defaults,omitempty" json:"persistent_defaults,omitempty"`
}

type ProjectContext struct {
	SchemaVersion string     `yaml:"schema_version" json:"schema_version"`
	Language      string     `yaml:"language" json:"language"`
	Motivation    Motivation `yaml:"motivation" json:"motivation"`
	Warnings      []string   `yaml:"warnings,omitempty" json:"warnings,omitempty"`
}

type ClarificationQuestion struct {
	ID             string  `yaml:"id" json:"id"`
	Question       string  `yaml:"question" json:"question"`
	DecisionImpact string  `yaml:"decision_impact" json:"decision_impact"`
	Critical       bool    `yaml:"critical" json:"critical"`
	Answer         *string `yaml:"answer,omitempty" json:"answer,omitempty"`
}

type QualityAttribute struct {
	Name             string  `yaml:"name" json:"name"`
	Priority         int     `yaml:"priority" json:"priority"`
	Rationale        string  `yaml:"rationale" json:"rationale"`
	MeasurableSignal *string `yaml:"measurable_signal,omitempty" json:"measurable_signal,omitempty"`
}

type Component struct {
	ID               string   `yaml:"id" json:"id"`
	Responsibility   string   `yaml:"responsibility" json:"responsibility"`
	OwnsData         []string `yaml:"owns_data" json:"owns_data"`
	PublicInterfaces []string `yaml:"public_interfaces" json:"public_interfaces"`
}

type ExternalBoundary struct {
	ID             string `yaml:"id" json:"id"`
	Responsibility string `yaml:"responsibility" json:"responsibility"`
}

type DependencyRule struct {
	Source       string  `yaml:"source" json:"source"`
	Target       string  `yaml:"target" json:"target"`
	Policy       string  `yaml:"policy" json:"policy"`
	ViaInterface *string `yaml:"via_interface,omitempty" json:"via_interface,omitempty"`
	Rationale    string  `yaml:"rationale" json:"rationale"`
}

type ArchitectureContract struct {
	SchemaVersion       string                  `yaml:"schema_version" json:"schema_version"`
	Revision            int                     `yaml:"revision" json:"revision"`
	Scope               string                  `yaml:"scope" json:"scope"`
	ArchitectureStyle   *string                 `yaml:"architecture_style,omitempty" json:"architecture_style,omitempty"`
	QualityAttributes   []QualityAttribute      `yaml:"quality_attributes" json:"quality_attributes"`
	Components          []Component             `yaml:"components" json:"components"`
	ExternalBoundaries  []ExternalBoundary      `yaml:"external_boundaries" json:"external_boundaries"`
	DependencyRules     []DependencyRule        `yaml:"dependency_rules" json:"dependency_rules"`
	RequiredPractices   []string                `yaml:"required_practices" json:"required_practices"`
	ProhibitedPractices []string                `yaml:"prohibited_practices" json:"prohibited_practices"`
	DecisionIDs         []string                `yaml:"decision_ids" json:"decision_ids"`
	UnresolvedQuestions []ClarificationQuestion `yaml:"unresolved_questions" json:"unresolved_questions"`
}

// BundleDecisionInput is the canonical filename/content pair used when an
// agent proposes a complete record-and-handoff write before persistence.
type BundleDecisionInput struct {
	Filename string `json:"filename"`
	Content  string `json:"content"`
}

// ArtifactBundleInput keeps the four durable artifact types separate while
// allowing one deterministic cross-artifact validation pass.
type ArtifactBundleInput struct {
	ContractYAML   string                `json:"contract"`
	Decisions      []BundleDecisionInput `json:"decisions"`
	ProjectContext string                `json:"project_context"`
	CodingHandoff  string                `json:"coding_handoff"`
}

// ArtifactBundle is returned only after each structured artifact has been
// parsed. Narrative files remain separate so agents do not mistake procedure
// (AGENTS.md) for historical decision record.
type ArtifactBundle struct {
	Contract       ArchitectureContract `json:"contract"`
	Decisions      []DecisionArtifact   `json:"decisions"`
	ProjectContext string               `json:"project_context"`
	CodingHandoff  string               `json:"coding_handoff"`
}

type ArtifactBundleSummary struct {
	Result              ValidationResult `json:"result"`
	ContractDecisionIDs []string         `json:"contract_decision_ids"`
	DecisionIDs         []string         `json:"decision_ids"`
	DecisionCount       int              `json:"decision_count"`
}

type ValidationIssue struct {
	Path     string `json:"path"`
	Message  string `json:"message"`
	Severity string `json:"severity"`
	Code     string `json:"code"`
}

type ValidationResult struct {
	Valid     bool              `json:"valid"`
	Errors    []ValidationIssue `json:"errors,omitempty"`
	Warnings  []ValidationIssue `json:"warnings,omitempty"`
	Truncated bool              `json:"truncated,omitempty"`
}

type SecretFinding struct {
	Category string `json:"category"`
	Line     int    `json:"line"`
}

type SecretScanResult struct {
	SafeToWrite bool            `json:"safe_to_write"`
	Findings    []SecretFinding `json:"findings,omitempty"`
	Truncated   bool            `json:"truncated,omitempty"`
}

type DependencyEdge struct {
	Source   string `json:"source"`
	Target   string `json:"target"`
	Evidence string `json:"evidence"`
}

type DependencyResult struct {
	Edges         []DependencyEdge `json:"edges"`
	FilesExamined int              `json:"files_examined"`
	FilesSkipped  int              `json:"files_skipped"`
	Warnings      []string         `json:"warnings,omitempty"`
	Truncated     bool             `json:"truncated,omitempty"`
}

type ToolError struct {
	Code         string `json:"code"`
	Message      string `json:"message"`
	RelativePath string `json:"relative_path,omitempty"`
	Retryable    bool   `json:"retryable,omitempty"`
}

type SetupAnswer struct {
	Language        string            `json:"language"`
	WhyBuilding     string            `json:"why_building"`
	Problem         string            `json:"problem"`
	Audience        string            `json:"audience"`
	Alternatives    string            `json:"alternatives"`
	WhyThisSolution string            `json:"why_this_solution"`
	AutoReason      string            `json:"auto_reason"`
	Options         map[string]string `json:"options"`
}

type QuestionSpec struct {
	ID              string   `json:"id"`
	Prompt          string   `json:"prompt"`
	Response        string   `json:"response"`
	Required        bool     `json:"required"`
	Default         string   `json:"default,omitempty"`
	Choices         []string `json:"choices,omitempty"`
	Persistent      bool     `json:"persistent"`
	AskedPerProject bool     `json:"asked_per_project"`
}
