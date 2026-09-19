package architect

import (
	"strings"
	"testing"
)

func TestAnalyzeInlineDependenciesRejectsWindowsAbsolutePath(t *testing.T) {
	_, err := AnalyzeInlineDependencies(
		[]SourceFile{{
			RelativePath: `C:\Users\agent\project.go`,
			Content:      "package main\n",
		}},
		[]string{"go"},
	)
	if err == nil || !strings.Contains(err.Error(), "project-relative") {
		t.Fatalf("Windows absolute path result = %v", err)
	}
}
