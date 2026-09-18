package architect

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type SourceFileInput struct {
	RelativePath string `json:"relative_path" yaml:"relative_path"`
	Content      string `json:"content" yaml:"content"`
}

type DependencyStatementInput struct {
	RelativePath string `json:"relative_path" yaml:"relative_path"`
	StartLine    int    `json:"start_line" yaml:"start_line"`
	Statement    string `json:"statement" yaml:"statement"`
}

type RepositoryAnalysisInput struct {
	RelativeRoots        []string                   `json:"relative_roots"`
	SourceFiles          []SourceFileInput          `json:"source_files"`
	DependencyStatements []DependencyStatementInput `json:"dependency_statements"`
	Languages            []string                   `json:"languages"`
}

type DependencyEdge struct {
	Source   string `json:"source"`
	Target   string `json:"target"`
	Evidence string `json:"evidence"`
}

type DependencyGraphEvidence struct {
	Edges         []DependencyEdge `json:"edges"`
	FilesExamined int              `json:"files_examined"`
	FilesSkipped  int              `json:"files_skipped"`
	Warnings      []string         `json:"warnings,omitempty"`
	Truncated     bool             `json:"truncated,omitempty"`
}

var (
	pythonImportPattern     = regexp.MustCompile(`^\s*import\s+(.+?)\s*(?:#.*)?$`)
	pythonFromPattern       = regexp.MustCompile(`^\s*from\s+([.A-Za-z_][.A-Za-z0-9_]*)\s+import\s+.+?(?:#.*)?$`)
	pythonDynamicPattern    = regexp.MustCompile(`\b(__import__|importlib\.import_module)\s*\(`)
	pythonModuleNamePattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_.]*$`)
)

func AnalyzeRepositoryDependencies(reader *WorkspaceReader, request RepositoryAnalysisInput) (DependencyGraphEvidence, error) {
	modeCount := 0
	if len(request.RelativeRoots) > 0 {
		modeCount++
	}
	if len(request.SourceFiles) > 0 {
		modeCount++
	}
	if len(request.DependencyStatements) > 0 {
		modeCount++
	}
	if modeCount > 1 {
		return DependencyGraphEvidence{}, &WorkspaceError{Code: "invalid-input", Message: "provide exactly one dependency evidence source"}
	}
	if len(request.DependencyStatements) > 0 {
		return analyzeDependencyStatements(request.DependencyStatements)
	}
	if len(request.SourceFiles) > 0 {
		return analyzeInlineSourceFiles(request.SourceFiles)
	}
	if reader == nil {
		return DependencyGraphEvidence{}, &WorkspaceError{Code: "workspace-unavailable", Message: "dependency evidence source is unavailable"}
	}
	paths := request.RelativeRoots
	if len(paths) == 0 {
		paths = []string{"."}
	}
	files, err := reader.IterFiles(paths, map[string]bool{".py": true})
	if err != nil {
		return DependencyGraphEvidence{}, err
	}
	result := DependencyGraphEvidence{Edges: []DependencyEdge{}, Warnings: []string{}}
	seenBytes := 0
	for _, path := range files {
		if result.FilesExamined >= MaxWorkspaceFiles {
			result.Truncated = true
			break
		}
		content, readErr := reader.ReadText(path, map[string]bool{".py": true})
		if readErr != nil {
			result.FilesSkipped++
			result.Warnings = append(result.Warnings, fmt.Sprintf("Skipped %s: %T", path, readErr))
			continue
		}
		if seenBytes+len(content) > MaxWorkspaceBytes {
			result.Truncated = true
			break
		}
		seenBytes += len(content)
		result.FilesExamined++
		source := pythonModuleName(path)
		lines := strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n")
		for lineNumber := 0; lineNumber < len(lines); lineNumber++ {
			line := lines[lineNumber]
			targets := staticImportTargets(line)
			if len(targets) == 0 && startsMultilineFromImport(line) {
				statement := line
				end := lineNumber
				for end+1 < len(lines) && !strings.Contains(statement, ")") {
					end++
					statement += "\n" + lines[end]
				}
				targets = staticImportTargets(strings.Join(strings.Fields(statement), " "))
				lineNumber = end
			}
			if pythonDynamicPattern.MatchString(line) {
				result.Warnings = append(result.Warnings, fmt.Sprintf("Dynamic import at %s:%d was not resolved", path, lineNumber+1))
			}
			for _, target := range targets {
				result.Edges = append(result.Edges, DependencyEdge{Source: source, Target: target, Evidence: path + ":" + strconv.Itoa(lineNumber+1)})
				if len(result.Edges) >= MaxDependencyEdges {
					result.Truncated = true
					break
				}
			}
			if result.Truncated {
				break
			}
		}
		if result.Truncated {
			break
		}
	}
	if len(result.Warnings) > 100 {
		result.Warnings = result.Warnings[:100]
	}
	return result, nil
}

func AnalyzeDependencyStatements(statements []DependencyStatementInput) (DependencyGraphEvidence, error) {
	return analyzeDependencyStatements(statements)
}

func analyzeDependencyStatements(statements []DependencyStatementInput) (DependencyGraphEvidence, error) {
	result := DependencyGraphEvidence{Edges: []DependencyEdge{}, Warnings: []string{"Fast statement mode analyzes only host-supplied static imports; dynamic imports and omitted statements are not evaluated"}}
	seen := map[string]bool{}
	examinedPaths := map[string]bool{}
	for _, item := range statements {
		path, err := ValidateInlinePythonPath(strings.TrimSpace(item.RelativePath))
		if err != nil {
			return DependencyGraphEvidence{}, err
		}
		if item.StartLine < 1 || item.StartLine > 10_000_000 || strings.TrimSpace(item.Statement) == "" || len([]byte(item.Statement)) > MaxDependencyStatementSize || strings.IndexByte(item.Statement, 0) >= 0 {
			return DependencyGraphEvidence{}, &WorkspaceError{Code: "invalid-input", Message: "dependency statement is invalid", RelativePath: path}
		}
		key := path + ":" + strconv.Itoa(item.StartLine)
		if seen[key] {
			return DependencyGraphEvidence{}, &WorkspaceError{Code: "invalid-input", Message: "dependency statement paths and lines must be unique", RelativePath: path}
		}
		seen[key] = true
		targets := staticImportTargets(strings.Join(strings.Fields(strings.ReplaceAll(item.Statement, "\r\n", "\n")), " "))
		if len(targets) == 0 {
			return DependencyGraphEvidence{}, &WorkspaceError{Code: "invalid-input", Message: "dependency statement must contain exactly one static import", RelativePath: path}
		}
		examinedPaths[path] = true
		for _, target := range targets {
			result.Edges = append(result.Edges, DependencyEdge{Source: pythonModuleName(path), Target: target, Evidence: fmt.Sprintf("%s:%d", path, item.StartLine)})
			if len(result.Edges) >= MaxDependencyEdges {
				result.Truncated = true
				break
			}
		}
		if result.Truncated {
			break
		}
	}
	result.FilesExamined = len(examinedPaths)
	return result, nil
}

func analyzeInlineSourceFiles(sources []SourceFileInput) (DependencyGraphEvidence, error) {
	if len(sources) > MaxInlineSourceFiles {
		return DependencyGraphEvidence{}, &WorkspaceError{Code: "budget-exhausted", Message: "source_files exceed the file budget"}
	}
	normalized := make(map[string]string, len(sources))
	totalBytes := 0
	for _, source := range sources {
		path, err := ValidateInlinePythonPath(strings.TrimSpace(source.RelativePath))
		if err != nil {
			return DependencyGraphEvidence{}, err
		}
		if _, exists := normalized[path]; exists {
			return DependencyGraphEvidence{}, &WorkspaceError{Code: "invalid-input", Message: "source_files paths must be unique", RelativePath: path}
		}
		if strings.IndexByte(source.Content, 0) >= 0 {
			return DependencyGraphEvidence{}, &WorkspaceError{Code: "unsupported-format", Message: "inline source contains a null byte", RelativePath: path}
		}
		contentBytes := len([]byte(source.Content))
		if contentBytes > MaxInlineSourceFileBytes {
			return DependencyGraphEvidence{}, &WorkspaceError{Code: "budget-exhausted", Message: "inline source exceeds the single-file budget", RelativePath: path}
		}
		totalBytes += contentBytes
		if totalBytes > MaxInlineSourceBytes {
			return DependencyGraphEvidence{}, &WorkspaceError{Code: "budget-exhausted", Message: "inline sources exceed the total byte budget"}
		}
		normalized[path] = source.Content
	}
	paths := make([]string, 0, len(normalized))
	for path := range normalized {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	result := DependencyGraphEvidence{Edges: []DependencyEdge{}, Warnings: []string{"Inline source mode analyzes only host-supplied files; coverage may be incomplete"}}
	for _, path := range paths {
		content := normalized[path]
		lines := strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n")
		source := pythonModuleName(path)
		fileEdges := []DependencyEdge{}
		fileWarnings := []string{}
		for lineNumber := 0; lineNumber < len(lines); lineNumber++ {
			line := lines[lineNumber]
			targets := staticImportTargets(line)
			if len(targets) == 0 && startsMultilineFromImport(line) {
				statement := line
				end := lineNumber
				for end+1 < len(lines) && !strings.Contains(statement, ")") {
					end++
					statement += "\n" + lines[end]
				}
				targets = staticImportTargets(strings.Join(strings.Fields(statement), " "))
				lineNumber = end
			}
			if pythonDynamicPattern.MatchString(line) {
				fileWarnings = append(fileWarnings, fmt.Sprintf("Dynamic import at %s:%d was not resolved", path, lineNumber+1))
			}
			for _, target := range targets {
				fileEdges = append(fileEdges, DependencyEdge{Source: source, Target: target, Evidence: fmt.Sprintf("%s:%d", path, lineNumber+1)})
				if len(fileEdges)+len(result.Edges) >= MaxDependencyEdges {
					result.Truncated = true
					break
				}
			}
			if result.Truncated {
				break
			}
		}
		result.FilesExamined++
		result.Edges = append(result.Edges, fileEdges...)
		result.Warnings = append(result.Warnings, fileWarnings...)
		if result.Truncated {
			break
		}
	}
	if len(result.Warnings) > 100 {
		result.Warnings = result.Warnings[:100]
	}
	return result, nil
}

func staticImportTargets(line string) []string {
	line = strings.TrimSpace(line)
	if matches := pythonFromPattern.FindStringSubmatch(line); len(matches) == 2 {
		return []string{matches[1]}
	}
	if matches := pythonImportPattern.FindStringSubmatch(line); len(matches) == 2 {
		result := []string{}
		for _, item := range strings.Split(matches[1], ",") {
			module := strings.TrimSpace(strings.SplitN(item, " as ", 2)[0])
			if pythonModuleNamePattern.MatchString(module) {
				result = append(result, module)
			}
		}
		return result
	}
	return nil
}

func startsMultilineFromImport(line string) bool {
	trimmed := strings.TrimSpace(line)
	return strings.HasPrefix(trimmed, "from ") && strings.Contains(trimmed, " import ") && !strings.Contains(trimmed, ")")
}

func pythonModuleName(path string) string {
	module := strings.TrimSuffix(strings.ReplaceAll(path, "\\", "/"), ".py")
	module = strings.ReplaceAll(module, "/", ".")
	module = strings.TrimSuffix(module, ".__init__")
	return module
}

func MatchesComponent(module, component string) bool {
	normalized := strings.ReplaceAll(component, "-", "_")
	module = strings.TrimLeft(module, ".")
	return module == normalized || strings.HasPrefix(module, normalized+".") || contains(strings.Split(module, "."), normalized)
}

func CheckArchitectureBoundaries(reader *WorkspaceReader, contractYAML string, request RepositoryAnalysisInput) (ConformanceReport, error) {
	contract, err := ParseContractYAML(contractYAML)
	if err != nil {
		return ConformanceReport{Scope: "invalid-contract", Findings: []ConformanceFinding{{ID: "F-001", Classification: "possible-drift", Severity: "high", Confidence: "high", Rule: "The architecture contract must be valid before boundaries are checked.", Evidence: []string{err.Error()}, Recommendation: "Correct and validate the architecture contract, then rerun the check."}}}, nil
	}
	graph, err := AnalyzeRepositoryDependencies(reader, request)
	if err != nil {
		return ConformanceReport{}, err
	}
	report := ConformanceReport{Scope: contract.Scope, Findings: []ConformanceFinding{}, FilesExamined: graph.FilesExamined, FilesSkipped: graph.FilesSkipped, Warnings: graph.Warnings, Truncated: graph.Truncated}
	for _, rule := range contract.DependencyRules {
		if rule.Policy != "deny" {
			continue
		}
		var evidence []string
		for _, edge := range graph.Edges {
			if MatchesComponent(edge.Source, rule.Source) && MatchesComponent(edge.Target, rule.Target) {
				evidence = append(evidence, edge.Evidence)
			}
		}
		if len(evidence) == 0 {
			continue
		}
		if len(evidence) > 50 {
			evidence = evidence[:50]
		}
		report.Findings = append(report.Findings, ConformanceFinding{ID: fmt.Sprintf("F-%03d", len(report.Findings)+1), Classification: "confirmed-violation", Severity: "high", Confidence: "high", Rule: fmt.Sprintf("Dependency from %s to %s is denied: %s", rule.Source, rule.Target, rule.Rationale), Evidence: evidence, Recommendation: "Remove the dependency or revise the accepted architecture decision."})
		if len(report.Findings) >= 200 {
			report.Truncated = true
			break
		}
	}
	return report, nil
}
