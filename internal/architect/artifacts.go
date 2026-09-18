package architect

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type ArtifactKind string

const (
	ArtifactADR                ArtifactKind = "adr"
	ArtifactContract           ArtifactKind = "contract"
	ArtifactContext            ArtifactKind = "context"
	ArtifactImplementationPlan ArtifactKind = "implementation-plan"
)

type ArtifactCandidate struct {
	Path      string       `json:"path"`
	Content   string       `json:"content"`
	Kind      ArtifactKind `json:"kind"`
	Operation string       `json:"operation"`
}

var adrFilenamePattern = regexp.MustCompile(`^ADR-[0-9]{3}(?:-[a-z0-9]+(?:-[a-z0-9]+)*)?\.md$`)

func CanonicalArtifactPath(target, workspace string) (string, ArtifactKind, bool) {
	raw := strings.TrimSpace(target)
	if raw == "" || strings.IndexByte(raw, 0) >= 0 {
		return "", "", false
	}
	normalized := strings.ReplaceAll(raw, "\\", "/")
	if filepath.IsAbs(raw) || strings.HasPrefix(normalized, "/") || drivePathPattern.MatchString(normalized) {
		if workspace == "" {
			return "", "", false
		}
		absolute, err := filepath.Abs(raw)
		if err != nil {
			return "", "", false
		}
		root, err := filepath.Abs(workspace)
		if err != nil {
			return "", "", false
		}
		relative, err := filepath.Rel(root, absolute)
		if err != nil || !withinRoot(root, absolute) {
			return "", "", false
		}
		normalized = filepath.ToSlash(relative)
	}
	parts := strings.Split(normalized, "/")
	for _, part := range parts {
		if part == "" || part == "." || part == ".." {
			return "", "", false
		}
	}
	canonical := strings.Join(parts, "/")
	switch canonical {
	case ".ai-architect/project-context.md":
		return canonical, ArtifactContext, true
	case ".ai-architect/architecture-contract.yaml":
		return canonical, ArtifactContract, true
	case ".ai-architect/implementation-plan.md":
		return canonical, ArtifactImplementationPlan, true
	}
	if strings.HasPrefix(canonical, ".ai-architect/decisions/") {
		name := strings.TrimPrefix(canonical, ".ai-architect/decisions/")
		if adrFilenamePattern.MatchString(name) {
			return canonical, ArtifactADR, true
		}
	}
	return "", "", false
}

func IsCanonicalArtifactPath(target, workspace string) bool {
	_, _, ok := CanonicalArtifactPath(target, workspace)
	return ok
}

var (
	fileSectionPattern = regexp.MustCompile(`(?m)^\*\*\* (Add|Update|Delete) File: (.+)$`)
	hunkPattern        = regexp.MustCompile(`(?m)^@@.*$`)
)

func ProposedArtifactCandidates(toolInput any, workspace string) ([]ArtifactCandidate, error) {
	patch := patchTextFromToolInput(toolInput)
	if patch != "" {
		return patchCandidates(patch, workspace)
	}
	return structuredWriteCandidate(toolInput, workspace)
}

func patchTextFromToolInput(value any) string {
	if text, ok := value.(string); ok && strings.Contains(text, "*** Begin Patch") && strings.Contains(text, "*** End Patch") {
		return text
	}
	var candidates []string
	var collect func(any, int)
	collect = func(current any, depth int) {
		if depth > 4 {
			return
		}
		switch item := current.(type) {
		case string:
			if strings.Contains(item, "*** Begin Patch") && strings.Contains(item, "*** End Patch") {
				candidates = append(candidates, item)
			}
		case map[string]any:
			for _, nested := range item {
				collect(nested, depth+1)
			}
		case []any:
			for _, nested := range item {
				collect(nested, depth+1)
			}
		}
	}
	collect(value, 0)
	if len(candidates) == 1 {
		return candidates[0]
	}
	return ""
}

func patchCandidates(patch, workspace string) ([]ArtifactCandidate, error) {
	matches := fileSectionPattern.FindAllStringSubmatchIndex(patch, -1)
	if len(matches) == 0 {
		return nil, fmt.Errorf("patch contains no reconstructable file section")
	}
	result := make([]ArtifactCandidate, 0, len(matches))
	seen := map[string]bool{}
	for index, match := range matches {
		operation := patch[match[2]:match[3]]
		target := strings.TrimSpace(patch[match[4]:match[5]])
		if operation == "Delete" {
			return nil, fmt.Errorf("architecture artifact deletion is not permitted by this workflow")
		}
		relative, kind, ok := CanonicalArtifactPath(target, workspace)
		if !ok {
			return nil, fmt.Errorf("unsupported architecture artifact path: %s", target)
		}
		if seen[relative] {
			return nil, fmt.Errorf("patch contains duplicate artifact target: %s", relative)
		}
		seen[relative] = true
		start := match[1]
		end := strings.Index(patch[start:], "*** End Patch")
		if index+1 < len(matches) {
			end = matches[index+1][0] - start
		}
		if end < 0 {
			return nil, fmt.Errorf("patch terminator is missing")
		}
		body := strings.TrimLeft(patch[start:start+end], "\r\n")
		var content string
		var err error
		if operation == "Add" {
			content, err = addedContent(body)
		} else {
			original, readErr := os.ReadFile(filepath.Join(workspace, filepath.FromSlash(relative)))
			if readErr != nil {
				return nil, readErr
			}
			content, err = applyUpdatePatch(string(original), body)
		}
		if err != nil {
			return nil, err
		}
		result = append(result, ArtifactCandidate{Path: relative, Content: content, Kind: kind, Operation: operation})
	}
	return result, nil
}

func addedContent(body string) (string, error) {
	lines := strings.Split(strings.TrimSuffix(strings.ReplaceAll(body, "\r\n", "\n"), "\n"), "\n")
	for _, line := range lines {
		if line != "" && !strings.HasPrefix(line, "+") {
			return "", fmt.Errorf("added-file patch contains a non-addition line")
		}
	}
	if len(lines) == 1 && lines[0] == "" {
		return "\n", nil
	}
	for index, line := range lines {
		if strings.HasPrefix(line, "+") {
			lines[index] = line[1:]
		}
	}
	return strings.Join(lines, "\n") + "\n", nil
}

func applyUpdatePatch(original, body string) (string, error) {
	matches := hunkPattern.FindAllStringIndex(body, -1)
	if len(matches) == 0 {
		return "", fmt.Errorf("updated-file patch has no reconstructable hunk")
	}
	current := strings.Split(strings.TrimSuffix(strings.ReplaceAll(original, "\r\n", "\n"), "\n"), "\n")
	if len(current) == 1 && current[0] == "" {
		current = nil
	}
	searchFrom := 0
	for index, match := range matches {
		start := match[1]
		end := len(body)
		if index+1 < len(matches) {
			end = matches[index+1][0]
		}
		oldLines, newLines, err := patchHunkLines(body[start:end])
		if err != nil {
			return "", err
		}
		if len(oldLines) == 0 {
			return "", fmt.Errorf("updated-file patch has no stable original context")
		}
		locations := []int{}
		for position := searchFrom; position+len(oldLines) <= len(current); position++ {
			matched := true
			for offset := range oldLines {
				if current[position+offset] != oldLines[offset] {
					matched = false
					break
				}
			}
			if matched {
				locations = append(locations, position)
			}
		}
		if len(locations) != 1 {
			return "", fmt.Errorf("updated-file patch context is missing or ambiguous")
		}
		position := locations[0]
		replacement := append([]string{}, current[:position]...)
		replacement = append(replacement, newLines...)
		replacement = append(replacement, current[position+len(oldLines):]...)
		current = replacement
		searchFrom = position + len(newLines)
	}
	return strings.Join(current, "\n") + "\n", nil
}

func patchHunkLines(hunk string) ([]string, []string, error) {
	oldLines, newLines := []string{}, []string{}
	for _, line := range strings.Split(strings.TrimLeft(hunk, "\r\n"), "\n") {
		if strings.HasPrefix(line, "\\ No newline at end of file") {
			continue
		}
		if line == "" || (line[0] != ' ' && line[0] != '+' && line[0] != '-') {
			return nil, nil, fmt.Errorf("updated-file patch contains an unsupported hunk line")
		}
		if line[0] == ' ' || line[0] == '-' {
			oldLines = append(oldLines, line[1:])
		}
		if line[0] == ' ' || line[0] == '+' {
			newLines = append(newLines, line[1:])
		}
	}
	return oldLines, newLines, nil
}

func structuredWriteCandidate(toolInput any, workspace string) ([]ArtifactCandidate, error) {
	values, ok := toolInput.(map[string]any)
	if !ok {
		return nil, nil
	}
	var target string
	for _, key := range []string{"file_path", "path", "target"} {
		if value, ok := values[key].(string); ok {
			target = value
			break
		}
	}
	if target == "" {
		return nil, nil
	}
	relative, kind, ok := CanonicalArtifactPath(target, workspace)
	if !ok {
		return nil, fmt.Errorf("unsupported architecture artifact path: %s", target)
	}
	path := filepath.Join(workspace, filepath.FromSlash(relative))
	operation := "Add"
	if info, err := os.Stat(path); err == nil && info.Mode().IsRegular() {
		operation = "Update"
	}
	var content string
	oldString, oldOK := values["old_string"].(string)
	newString, newOK := values["new_string"].(string)
	if oldOK && newOK {
		original, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		if oldString == "" || strings.Count(string(original), oldString) != 1 {
			return nil, fmt.Errorf("structured edit original text is missing or ambiguous")
		}
		content = strings.Replace(string(original), oldString, newString, 1)
	} else {
		for _, key := range []string{"content", "text"} {
			if value, ok := values[key].(string); ok {
				content = value
				break
			}
		}
		if content == "" && values["content"] == nil && values["text"] == nil {
			return nil, fmt.Errorf("structured write does not provide complete reconstructable content")
		}
	}
	return []ArtifactCandidate{{Path: relative, Content: content, Kind: kind, Operation: operation}}, nil
}

func ValidateArtifactBundleCandidates(candidates []ArtifactCandidate, requireComplete bool) (*ArtifactBundle, error) {
	kinds := map[ArtifactKind]bool{}
	for _, candidate := range candidates {
		kinds[candidate.Kind] = true
	}
	if !requireComplete && !(kinds[ArtifactContract] && len(kinds) > 1) {
		return nil, nil
	}
	missing := []string{}
	for _, kind := range []ArtifactKind{ArtifactADR, ArtifactContract, ArtifactContext, ArtifactImplementationPlan} {
		if !kinds[kind] {
			missing = append(missing, string(kind))
		}
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("record-and-handoff must write one complete artifact bundle; missing %s", strings.Join(missing, ", "))
	}
	var contractCandidate, contextCandidate, handoffCandidate *ArtifactCandidate
	var decisions []DecisionArtifact
	for index := range candidates {
		candidate := &candidates[index]
		switch candidate.Kind {
		case ArtifactContract:
			if contractCandidate != nil {
				return nil, fmt.Errorf("record-and-handoff requires exactly one contract")
			}
			contractCandidate = candidate
		case ArtifactContext:
			if contextCandidate != nil {
				return nil, fmt.Errorf("record-and-handoff requires exactly one project context")
			}
			contextCandidate = candidate
		case ArtifactImplementationPlan:
			if handoffCandidate != nil {
				return nil, fmt.Errorf("record-and-handoff requires exactly one coding handoff")
			}
			handoffCandidate = candidate
		case ArtifactADR:
			artifact, err := ParseADRFrontmatter(candidate.Content)
			if err != nil {
				return nil, err
			}
			name := strings.TrimSuffix(filepath.Base(candidate.Path), filepath.Ext(candidate.Path))
			id := name
			if dash := strings.Index(id, "-"); dash >= 0 {
				if second := strings.Index(id[dash+1:], "-"); second >= 0 {
					id = id[:dash+1+second]
				}
			}
			if id != artifact.Decision.ID {
				return nil, fmt.Errorf("ADR filename and decision id disagree")
			}
			decisions = append(decisions, artifact)
		}
	}
	if contractCandidate == nil || contextCandidate == nil || handoffCandidate == nil || len(decisions) == 0 {
		return nil, fmt.Errorf("record-and-handoff requires exactly one contract, project context, coding handoff, and at least one ADR")
	}
	contract, err := ParseContractYAML(contractCandidate.Content)
	if err != nil {
		return nil, fmt.Errorf("architecture contract: %w", err)
	}
	bundle := &ArtifactBundle{Contract: contract, Decisions: decisions, ProjectContext: contextCandidate.Content, CodingHandoff: handoffCandidate.Content}
	if err := ValidateBundle(*bundle); err != nil {
		return nil, err
	}
	return bundle, nil
}

func ScanGeneratedArtifact(content string) ArtifactSecretScanResult {
	privateKey := regexp.MustCompile(`-----BEGIN (?:[A-Z0-9 ]+ )?PRIVATE KEY-----`)
	tokens := []*regexp.Regexp{regexp.MustCompile(`\bsk-[A-Za-z0-9_-]{20,}\b`), regexp.MustCompile(`\bgh[pousr]_[A-Za-z0-9]{20,}\b`), regexp.MustCompile(`\bAKIA[A-Z0-9]{16}\b`)}
	credential := regexp.MustCompile(`(?i)\b(?:password|passwd|api[_-]?key|client[_-]?secret|access[_-]?token)\b\s*[:=]\s*["']?([^\s"'#,;]{12,})`)
	safeMarkers := []string{"example", "placeholder", "changeme", "not-a-real", "${", "{{", "<"}
	findings := []SecretFinding{}
	for lineNumber, line := range strings.Split(content, "\n") {
		if privateKey.MatchString(line) {
			findings = append(findings, SecretFinding{Category: "private-key", Line: lineNumber + 1})
		}
		for _, pattern := range tokens {
			if pattern.MatchString(line) {
				findings = append(findings, SecretFinding{Category: "token", Line: lineNumber + 1})
				break
			}
		}
		if match := credential.FindStringSubmatch(line); len(match) == 2 {
			value := strings.ToLower(match[1])
			safe := false
			for _, marker := range safeMarkers {
				if strings.Contains(value, marker) {
					safe = true
					break
				}
			}
			if !safe {
				findings = append(findings, SecretFinding{Category: "credential", Line: lineNumber + 1})
			}
		}
		if len(findings) >= 100 {
			return ArtifactSecretScanResult{SafeToWrite: false, Findings: findings, Truncated: true}
		}
	}
	return ArtifactSecretScanResult{SafeToWrite: len(findings) == 0, Findings: findings}
}
