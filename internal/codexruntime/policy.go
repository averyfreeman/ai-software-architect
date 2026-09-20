package codexruntime

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/averyfreeman/git-bbq/internal/architect"
)

var (
	adrFilenamePattern = regexp.MustCompile("^ADR-[0-9]{3}(?:-[a-z0-9]+(?:-[a-z0-9]+)*)?\\.md$")
	patchFilePattern   = regexp.MustCompile("(?m)^\\*\\*\\* (Add|Update|Delete) File: (.+)$")
	hunkPattern        = regexp.MustCompile("(?m)^@@.*$")
	decisionIDPattern  = regexp.MustCompile("(?m)^\\s*id:\\s*(ADR-[0-9]{3})\\s*$")
)

var (
	shellTools      = map[string]bool{"bash": true, "exec_command": true, "shell_command": true}
	patchTools      = map[string]bool{"apply_patch": true, "edit": true, "write": true}
	webTools        = map[string]bool{"websearch": true, "web_search": true, "search_query": true, "web__run": true}
	powershellReads = map[string]bool{"get-childitem": true, "get-content": true, "select-string": true, "test-path": true}
	gitReads        = map[string]bool{"diff": true, "log": true, "ls-files": true, "show": true, "status": true}
)

func normalizedToolName(value any) string {
	text, ok := value.(string)
	if !ok {
		return ""
	}
	parts := strings.Split(strings.ToLower(text), ".")
	return parts[len(parts)-1]
}

func commandFromInput(value any) (string, bool) {
	if command, ok := value.(string); ok {
		return command, true
	}
	input, ok := value.(map[string]any)
	if !ok {
		return "", false
	}
	command, ok := input["command"].(string)
	return command, ok
}

func shellDenialReason(toolInput any, options Options) string {
	command, ok := commandFromInput(toolInput)
	if !ok {
		return "The AI Software Architect could not verify this shell command's arguments, so it was denied. Use host-native static reads instead."
	}
	stripped := strings.TrimSpace(command)
	snapshot := repositorySnapshotCommand(options.Executable)
	if options.Executable != "" && stripped == snapshot {
		return ""
	}
	if stripped == "" || strings.ContainsAny(stripped, "\r\n;&|{}<>$") || strings.Contains(stripped, string(rune(96))) {
		return "AI Software Architect shell inspection is fail-closed: use exactly one allowlisted static read command without pipelines, variables, call operators, script blocks, redirection, or command composition."
	}
	words := strings.Fields(stripped)
	if len(words) == 0 {
		return "AI Software Architect shell inspection was denied because the command was empty."
	}
	executable := strings.TrimSuffix(strings.ToLower(words[0]), ".exe")
	if powershellReads[executable] {
		return ""
	}
	if executable == "git" && len(words) >= 2 && gitReads[strings.ToLower(words[1])] {
		for _, word := range words[2:] {
			option := strings.ToLower(strings.SplitN(word, "=", 2)[0])
			if option == "--ext-diff" || option == "--textconv" {
				return "AI Software Architect denied a Git option that could execute repository-defined tooling."
			}
		}
		return ""
	}
	if executable == "rg" || executable == "ripgrep" {
		for _, word := range words[1:] {
			option := strings.ToLower(strings.SplitN(word, "=", 2)[0])
			if option == "--pre" || option == "--pre-glob" {
				return "AI Software Architect denied a ripgrep preprocessor that could execute repository-defined tooling."
			}
		}
		return ""
	}
	return "AI Software Architect analysis treats repository code as untrusted data. Only allowlisted static file and Git reads are permitted; interpreters, test runners, package tools, scripts, and other executables are denied."
}

func repositorySnapshotCommand(executable string) string {
	if executable == "" {
		return ""
	}
	if filepath.Separator == '\\' {
		return "& \"" + strings.ReplaceAll(executable, "\"", "\"\"") + "\" --repository-snapshot --root ."
	}
	if strings.ContainsAny(executable, " \t\r\n'\"\\$"+string(rune(96))) {
		return "'" + strings.ReplaceAll(executable, "'", "'\\''") + "' --repository-snapshot --root ."
	}
	return executable + " --repository-snapshot --root ."
}

func canonicalArtifactPath(target, workspace string) (string, string, bool) {
	target = strings.TrimSpace(target)
	if target == "" || strings.ContainsRune(target, 0) {
		return "", "", false
	}
	relative := target
	if filepath.IsAbs(target) {
		if workspace == "" {
			return "", "", false
		}
		workspaceAbs, err := filepath.Abs(workspace)
		if err != nil {
			return "", "", false
		}
		targetAbs, err := filepath.Abs(target)
		if err != nil {
			return "", "", false
		}
		relative, err = filepath.Rel(workspaceAbs, targetAbs)
		if err != nil {
			return "", "", false
		}
	}
	normalized := filepath.ToSlash(relative)
	segments := strings.Split(normalized, "/")
	for _, segment := range segments {
		if segment == "" || segment == "." || segment == ".." {
			return "", "", false
		}
	}
	if normalized == ".ai-architect/project-context.md" {
		return normalized, "context", true
	}
	if normalized == ".ai-architect/architecture-contract.yaml" {
		return normalized, "contract", true
	}
	if normalized == ".ai-architect/implementation-plan.md" {
		return normalized, "implementation-plan", true
	}
	if strings.HasPrefix(normalized, ".ai-architect/decisions/") {
		name := strings.TrimPrefix(normalized, ".ai-architect/decisions/")
		if adrFilenamePattern.MatchString(name) {
			return normalized, "adr", true
		}
	}
	return "", "", false
}

func patchTextFromInput(value any) (string, bool) {
	if text, ok := value.(string); ok {
		return text, strings.Contains(text, "*** Begin Patch")
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
	if len(candidates) != 1 {
		return "", false
	}
	return candidates[0], true
}

func addedContent(body string) (string, error) {
	lines := strings.Split(strings.TrimSuffix(body, "\n"), "\n")
	for _, line := range lines {
		if line != "" && !strings.HasPrefix(line, "+") {
			return "", errors.New("added-file patch contains a non-addition line")
		}
	}
	result := make([]string, 0, len(lines))
	for _, line := range lines {
		if strings.HasPrefix(line, "+") {
			result = append(result, line[1:])
		} else {
			result = append(result, "")
		}
	}
	return strings.Join(result, "\n") + "\n", nil
}

func applyUpdatePatch(original, body string) (string, error) {
	matches := hunkPattern.FindAllStringIndex(body, -1)
	if len(matches) == 0 {
		return "", errors.New("updated-file patch has no reconstructable hunk")
	}
	current := strings.Split(strings.TrimSuffix(original, "\n"), "\n")
	searchFrom := 0
	for index, match := range matches {
		end := len(body)
		if index+1 < len(matches) {
			end = matches[index+1][0]
		}
		hunk := strings.TrimLeft(body[match[1]:end], "\r\n")
		var oldLines, newLines []string
		for _, line := range strings.Split(hunk, "\n") {
			if strings.HasPrefix(line, "\\ No newline at end of file") {
				continue
			}
			if line == "" || (line[0] != ' ' && line[0] != '+' && line[0] != '-') {
				return "", errors.New("updated-file patch contains an unsupported hunk line")
			}
			if line[0] == ' ' || line[0] == '-' {
				oldLines = append(oldLines, line[1:])
			}
			if line[0] == ' ' || line[0] == '+' {
				newLines = append(newLines, line[1:])
			}
		}
		if len(oldLines) == 0 {
			return "", errors.New("updated-file patch has no stable original context")
		}
		locations := make([]int, 0)
		for position := searchFrom; position+len(oldLines) <= len(current); position++ {
			equal := true
			for offset := range oldLines {
				if current[position+offset] != oldLines[offset] {
					equal = false
					break
				}
			}
			if equal {
				locations = append(locations, position)
			}
		}
		if len(locations) != 1 {
			return "", errors.New("updated-file patch context is missing or ambiguous")
		}
		position := locations[0]
		current = append(append(append([]string{}, current[:position]...), newLines...), current[position+len(oldLines):]...)
		searchFrom = position + len(newLines)
	}
	return strings.Join(current, "\n") + "\n", nil
}

func patchCandidates(patch, workspace string) ([]artifactCandidate, error) {
	matches := patchFilePattern.FindAllStringSubmatchIndex(patch, -1)
	if len(matches) == 0 {
		return nil, errors.New("patch contains no reconstructable file section")
	}
	candidates := make([]artifactCandidate, 0, len(matches))
	seen := map[string]bool{}
	for index, match := range matches {
		operation := patch[match[2]:match[3]]
		target := patch[match[4]:match[5]]
		relative, kind, ok := canonicalArtifactPath(target, workspace)
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
			return nil, errors.New("patch terminator is missing")
		}
		body := strings.TrimLeft(patch[start:start+end], "\r\n")
		var content string
		var err error
		switch operation {
		case "Add":
			content, err = addedContent(body)
		case "Update":
			original, readErr := os.ReadFile(filepath.Join(workspace, filepath.FromSlash(relative)))
			if readErr != nil {
				err = readErr
			} else {
				content, err = applyUpdatePatch(string(original), body)
			}
		case "Delete":
			err = errors.New("architecture artifact deletion is not permitted by this workflow")
		}
		if err != nil {
			return nil, err
		}
		candidates = append(candidates, artifactCandidate{Path: relative, Content: content, Kind: kind, Operation: operation})
	}
	return candidates, nil
}

func structuredCandidate(value any, workspace string) ([]artifactCandidate, error) {
	input, ok := value.(map[string]any)
	if !ok {
		return nil, nil
	}
	var target string
	for _, key := range []string{"file_path", "path", "target"} {
		if candidate, ok := input[key].(string); ok {
			target = candidate
			break
		}
	}
	if target == "" {
		return nil, nil
	}
	relative, kind, ok := canonicalArtifactPath(target, workspace)
	if !ok {
		return nil, fmt.Errorf("unsupported architecture artifact path: %s", target)
	}
	path := filepath.Join(workspace, filepath.FromSlash(relative))
	info, statErr := os.Stat(path)
	operation := "Add"
	if statErr == nil && !info.IsDir() {
		operation = "Update"
	} else if statErr != nil && !os.IsNotExist(statErr) {
		return nil, statErr
	}
	var content string
	if oldText, oldOK := input["old_string"].(string); oldOK {
		newText, newOK := input["new_string"].(string)
		if !newOK || oldText == "" {
			return nil, errors.New("structured edit original text is missing or ambiguous")
		}
		original, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		originalText := string(original)
		if strings.Count(originalText, oldText) != 1 {
			return nil, errors.New("structured edit original text is missing or ambiguous")
		}
		content = strings.Replace(originalText, oldText, newText, 1)
	} else {
		for _, key := range []string{"content", "text"} {
			if candidate, ok := input[key].(string); ok {
				content = candidate
				break
			}
		}
		if content == "" {
			return nil, errors.New("structured write does not provide complete reconstructable content")
		}
	}
	return []artifactCandidate{{Path: relative, Content: content, Kind: kind, Operation: operation}}, nil
}

func proposedCandidates(value any, workspace string) ([]artifactCandidate, error) {
	if patch, ok := patchTextFromInput(value); ok {
		return patchCandidates(patch, workspace)
	}
	return structuredCandidate(value, workspace)
}

func artifactDenialReason(value any, workspace string, requireComplete bool) string {
	candidates, err := proposedCandidates(value, workspace)
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
		validation := architect.ValidateArtifact(candidate.Content, candidate.Kind)
		if !validation.Valid {
			return "AI Software Architect blocked " + candidate.Path + " because its complete artifact is invalid: " + validationSummary(validation) + "."
		}
		if candidate.Kind == "adr" {
			match := decisionIDPattern.FindStringSubmatch(candidate.Content)
			if len(match) != 2 || !strings.HasPrefix(filepath.Base(candidate.Path), match[1]) {
				return "AI Software Architect blocked " + candidate.Path + " because its ADR filename and decision id disagree."
			}
		}
	}
	kinds := map[string]int{}
	for _, candidate := range candidates {
		kinds[candidate.Kind]++
	}
	if requireComplete || (kinds["contract"] > 0 && len(kinds) > 1) {
		bundleInput := architect.ArtifactBundleInput{}
		decisions := make([]architect.BundleDecisionInput, 0)
		for _, candidate := range candidates {
			switch candidate.Kind {
			case "contract":
				if kinds["contract"] != 1 {
					return "AI Software Architect blocked the record-and-handoff write because it requires exactly one architecture contract."
				}
				bundleInput.ContractYAML = candidate.Content
			case "context":
				if kinds["context"] != 1 {
					return "AI Software Architect blocked the record-and-handoff write because it requires exactly one project context."
				}
				bundleInput.ProjectContext = candidate.Content
			case "implementation-plan":
				if kinds["implementation-plan"] != 1 {
					return "AI Software Architect blocked the record-and-handoff write because it requires exactly one coding handoff."
				}
				bundleInput.CodingHandoff = candidate.Content
			case "adr":
				decisions = append(decisions, architect.BundleDecisionInput{
					Filename: filepath.Base(candidate.Path),
					Content:  candidate.Content,
				})
			}
		}
		bundleInput.Decisions = decisions
		_, result := architect.ValidateArtifactBundle(bundleInput)
		if !result.Valid {
			return "AI Software Architect blocked the record-and-handoff write because the complete artifact bundle is inconsistent: " + validationSummary(result) + "."
		}
	}
	return ""
}

func validationSummary(result architect.ValidationResult) string {
	issues := append([]architect.ValidationIssue{}, result.Errors...)
	sort.Slice(issues, func(left, right int) bool { return issues[left].Path < issues[right].Path })
	if len(issues) > 3 {
		issues = issues[:3]
	}
	parts := make([]string, 0, len(issues))
	for _, issue := range issues {
		parts = append(parts, issue.Path+": "+issue.Message)
	}
	return strings.Join(parts, "; ")
}

func patchIsLimited(value any, workspace string) bool {
	if patch, ok := patchTextFromInput(value); ok {
		matches := patchFilePattern.FindAllStringSubmatch(patch, -1)
		if len(matches) == 0 {
			return false
		}
		for _, match := range matches {
			if match[1] == "Delete" {
				return false
			}
			if _, _, ok := canonicalArtifactPath(match[2], workspace); !ok {
				return false
			}
		}
		return true
	}
	input, ok := value.(map[string]any)
	if !ok {
		return false
	}
	for _, key := range []string{"file_path", "path", "target"} {
		if target, ok := input[key].(string); ok {
			_, _, valid := canonicalArtifactPath(target, workspace)
			return valid
		}
	}
	return false
}

func toolDenialReason(context turnContext, toolName, toolInput any, workspace string, options Options) string {
	if !context.Active {
		return ""
	}
	name := normalizedToolName(toolName)
	if webTools[name] {
		return "AI Software Architect canonical references are bundled with the plugin; web lookup is disabled during the architecture workflow. Use the reference paths supplied by the active skill context."
	}
	if shellTools[name] {
		return shellDenialReason(toolInput, options)
	}
	if patchTools[name] && !patchIsLimited(toolInput, workspace) {
		return "The AI Software Architect never writes application code. Its patch surface is limited to .ai-architect/project-context.md, .ai-architect/architecture-contract.yaml, .ai-architect/implementation-plan.md, and canonical .ai-architect/decisions/ADR-NNN[-slug].md files."
	}
	return ""
}
