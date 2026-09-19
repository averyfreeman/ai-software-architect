package architect

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSetupSupportsJavaProjectProfile(t *testing.T) {
	root := t.TempDir()
	answers := testAnswers()
	answers.Language = "java"

	result, err := SetupProject(root, answers, false)
	if err != nil {
		t.Fatal(err)
	}
	if result.Language != "java" {
		t.Fatalf("language = %q", result.Language)
	}
	skill, err := os.ReadFile(filepath.Join(root, ".agents", "skills", "java-generation", "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(skill), "mvn test") {
		t.Fatalf("Java skill omitted test command: %s", skill)
	}
}

func TestSetupSupportsCSharpProfileAndAutoDetection(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "Example.csproj"), []byte("<Project />\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	answers := testAnswers()
	answers.Language = "auto"

	result, err := SetupProject(root, answers, false)
	if err != nil {
		t.Fatal(err)
	}
	if result.Language != "csharp" {
		t.Fatalf("language = %q, want csharp", result.Language)
	}
	skill, err := os.ReadFile(filepath.Join(root, ".agents", "skills", "csharp-generation", "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(skill), "dotnet test") {
		t.Fatalf("C# skill omitted test command: %s", skill)
	}
}
