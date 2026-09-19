package gitbbq

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// MigrationAssessment describes the legacy scaffold and the Git BBQ paths that
// an approved migration would create or reconcile. It never writes files.
type MigrationAssessment struct {
	Mode      string   `json:"mode"`
	Root      string   `json:"root"`
	Detected  bool     `json:"detected"`
	Legacy    []string `json:"legacy"`
	Proposed  []string `json:"proposed"`
	Conflicts []string `json:"conflicts"`
	Warnings  []string `json:"warnings,omitempty"`
	ReadOnly  bool     `json:"read_only"`
}

// AssessMigration inventories the legacy AI Software Architect scaffold and
// reports the target Git BBQ paths without changing the repository.
func AssessMigration(root string) (MigrationAssessment, error) {
	root = filepath.Clean(root)
	info, err := os.Stat(root)
	if err != nil {
		return MigrationAssessment{}, err
	}
	if !info.IsDir() {
		return MigrationAssessment{}, fmt.Errorf("migration root is not a directory: %s", root)
	}
	existing, err := SnapshotFiles(root)
	if err != nil {
		return MigrationAssessment{}, err
	}
	legacy := make([]string, 0)
	existingSet := make(map[string]bool, len(existing))
	for _, relative := range existing {
		existingSet[relative] = true
		if isLegacyScaffoldPath(relative) {
			legacy = append(legacy, relative)
		}
	}
	sort.Strings(legacy)
	proposed := append([]string(nil), projectPaths(root)...)
	sort.Strings(proposed)
	conflicts := make([]string, 0)
	for _, relative := range proposed {
		if existingSet[relative] {
			conflicts = append(conflicts, relative)
		}
	}
	warnings := []string{
		"migration is read-only until the user reviews this inventory and gives explicit approval for the apply step",
		"legacy .ai-architect artifacts remain in place; converted ADRs will be additive under docs/adr",
	}
	if len(legacy) == 0 {
		warnings = append(warnings, "no legacy AI Software Architect scaffold was detected")
	}
	return MigrationAssessment{
		Mode:      "migration-assessment",
		Root:      root,
		Detected:  len(legacy) > 0,
		Legacy:    legacy,
		Proposed:  proposed,
		Conflicts: conflicts,
		Warnings:  warnings,
		ReadOnly:  true,
	}, nil
}

func isLegacyScaffoldPath(relative string) bool {
	return relative == ".adr-scaffold.yaml" || relative == ".ai-architect" || strings.HasPrefix(relative, ".ai-architect/")
}
