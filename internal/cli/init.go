package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/averyfreeman/adr-repo-governance/internal/index"
)

// InitConfig holds configuration for the init command.
type InitConfig struct {
	Dir    string
	Output *Output
}

// InitResult describes the paths created or reused by the init command.
type InitResult struct {
	ADRDir   string `json:"adr_dir"`
	Template string `json:"template"`
	Index    string `json:"index"`
}

// RunInit bootstraps the ADR directory structure.
func RunInit(cfg *InitConfig) error {
	adrDir := cfg.Dir
	templatesDir := filepath.Join(adrDir, "templates")

	// Create directories
	if err := os.MkdirAll(templatesDir, 0755); err != nil {
		return fmt.Errorf("creating directories: %w", err)
	}
	cfg.Output.Println("Created %s", adrDir)

	// Create template if it doesn't exist
	templatePath := filepath.Join(templatesDir, "adr.md")
	if _, err := os.Stat(templatePath); os.IsNotExist(err) {
		if err := os.WriteFile(templatePath, []byte(defaultTemplate), 0644); err != nil {
			return fmt.Errorf("creating template: %w", err)
		}
		cfg.Output.Println("Created %s", templatePath)
	} else {
		cfg.Output.Println("Template already exists: %s", templatePath)
	}

	// Generate index.yaml from any existing ADR files.
	indexPath := filepath.Join(adrDir, "index.yaml")
	indexExists := false
	if _, err := os.Stat(indexPath); err == nil {
		indexExists = true
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("checking index: %w", err)
	}
	if err := index.WriteToDir(adrDir); err != nil {
		return fmt.Errorf("generating index: %w", err)
	}
	if indexExists {
		cfg.Output.Println("Updated %s", indexPath)
	} else {
		cfg.Output.Println("Created %s", indexPath)
	}

	result := &InitResult{
		ADRDir:   adrDir,
		Template: templatePath,
		Index:    indexPath,
	}

	if cfg.Output.IsStructuredFormat() {
		return cfg.Output.PrintStructured(result)
	}

	cfg.Output.Success("adr-rg initialized in %s", adrDir)
	return nil
}

const defaultTemplate = `---
adr_id: ADR-NNNN
title: "Your Decision Title Here"
status: proposed
date: YYYY-MM-DD
scope:
  paths:
    - "path/to/affected/**"
tags:
  - tag1
  - tag2
constraints:
  - "First constraint that MUST be followed"
  - "Second constraint"
invariants:
  - "Property that must always hold true"
supersedes: []
superseded_by: []
related_adrs: []
---

# ADR-NNNN: Your Decision Title Here

## Context

Describe the context and background that led to this decision. What problem are we solving? What forces are at play?

Decision drivers:
- Key driver 1 that influenced the decision
- Key driver 2
- Key driver 3

{{ if .Language }}
## Language Context

- Language: {{ .DisplayName }}
- Build: {{ .BuildCommand }}
- Test: {{ .TestCommand }}
- Format: {{ .FormatCommand }}
- Documentation: {{ .Documentation }}
{{ end }}

## Decision

State the decision clearly and concisely. Explain the reasoning behind the choice.

### [Chosen Option]: Adopted

**Adopted because:**
- Clear, concrete reason why this option was chosen
- Tie reasons to decision drivers above
- Technical, operational, or strategic justification

**Adopted despite:**
- Known downside or trade-off we consciously accepted
- Cost or weakness compared to alternatives
- Risk we are taking on

## Alternatives Considered

### [Alternative A]: Rejected

**Rejected because:**
- Clear, concrete reason why this option was not chosen
- Technical, organizational, or strategic reason
- How it failed to meet decision drivers

**Rejected despite:**
- Legitimate strength of this option
- Benefit that made it attractive
- Reason it was seriously considered

## Consequences

**Positive:**
- First positive consequence
- Second positive consequence

**Negative:**
- First negative consequence (and mitigation if any)

## Agent Guidance

_Optional section. Include specific instructions for AI agents working in the affected scope._

When working in this area:
- Specific guidance point 1
- Specific guidance point 2
`
