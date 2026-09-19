package architect

import (
	"fmt"
	"regexp"
	"strings"
)

type dependencyStatement struct {
	RelativePath string `json:"relative_path"`
	StartLine    int    `json:"start_line"`
	Statement    string `json:"statement"`
}

var (
	goImportPattern      = regexp.MustCompile(`^\s*import\s+(?:\(\s*)?["']([^"']+)["']`)
	goBlockImportPattern = regexp.MustCompile(`^\s*["']([^"']+)["']`)
	pythonImportPattern  = regexp.MustCompile(`^\s*import\s+([A-Za-z0-9_\.]+)|^\s*from\s+([\.A-Za-z0-9_]+)\s+import\s+`)
	jsImportPattern      = regexp.MustCompile(`^\s*import(?:[^"']+from\s+)?["']([^"']+)["']|^\s*(?:const|let|var)\s+.*?require\(["']([^"']+)["']\)`)
	rustImportPattern    = regexp.MustCompile(`^\s*(?:use|extern\s+crate)\s+([A-Za-z0-9_:]+)`)
	dynamicImportPattern = regexp.MustCompile(`\b(?:__import__|importlib\.import_module|require|import)\s*\(`)
)

func languageExtensions(languages []string) map[string]struct{} {
	if len(languages) == 0 {
		languages = []string{"go"}
	}
	result := map[string]struct{}{}
	for _, language := range languages {
		switch strings.ToLower(strings.TrimSpace(language)) {
		case "go":
			result[".go"] = struct{}{}
		case "python", "py":
			result[".py"] = struct{}{}
		case "typescript", "javascript", "ts", "js":
			result[".ts"] = struct{}{}
			result[".tsx"] = struct{}{}
			result[".js"] = struct{}{}
			result[".jsx"] = struct{}{}
		case "rust", "rs":
			result[".rs"] = struct{}{}
		}
	}
	return result
}

func importsForLine(language, line string, inGoBlock bool) (target string, isImport bool) {
	line = strings.TrimSpace(line)
	switch strings.ToLower(language) {
	case "go":
		match := goImportPattern.FindStringSubmatch(line)
		if len(match) == 2 {
			return match[1], true
		}
		if inGoBlock {
			match = goBlockImportPattern.FindStringSubmatch(line)
			if len(match) == 2 {
				return match[1], true
			}
		}
	case "python", "py":
		match := pythonImportPattern.FindStringSubmatch(line)
		if len(match) == 3 {
			if match[1] != "" {
				return match[1], true
			}
			return match[2], true
		}
	case "typescript", "javascript", "ts", "js":
		match := jsImportPattern.FindStringSubmatch(line)
		if len(match) == 3 {
			if match[1] != "" {
				return match[1], true
			}
			return match[2], true
		}
	case "rust", "rs":
		match := rustImportPattern.FindStringSubmatch(line)
		if len(match) == 2 {
			return match[1], true
		}
	}
	return "", false
}

func analyzeDependencies(files []sourceFile, languages []string, skipped int, truncated bool) DependencyResult {
	result := DependencyResult{FilesExamined: len(files), FilesSkipped: skipped, Truncated: truncated}
	for _, file := range files {
		language := languageForExtension(file.RelativePath)
		inGoBlock := false
		for lineNumber, line := range strings.Split(file.Content, "\n") {
			trimmed := strings.TrimSpace(line)
			if language == "go" && strings.HasPrefix(trimmed, "import (") {
				inGoBlock = true
				continue
			}
			if language == "go" && inGoBlock && trimmed == ")" {
				inGoBlock = false
				continue
			}
			if target, ok := importsForLine(language, line, inGoBlock); ok {
				result.Edges = append(result.Edges, DependencyEdge{Source: moduleName(file.RelativePath, language), Target: target, Evidence: fmt.Sprintf("%s:%d", file.RelativePath, lineNumber+1)})
				if len(result.Edges) >= 5_000 {
					result.Truncated = true
					return result
				}
			} else if dynamicImportPattern.MatchString(line) {
				result.Warnings = append(result.Warnings, fmt.Sprintf("dynamic import at %s:%d was not resolved", file.RelativePath, lineNumber+1))
			}
		}
	}
	if len(result.Warnings) > 100 {
		result.Warnings = result.Warnings[:100]
		result.Truncated = true
	}
	_ = languages
	return result
}

func languageForExtension(path string) string {
	switch strings.ToLower(strings.TrimSpace(path[strings.LastIndex(path, "."):])) {
	case ".go":
		return "go"
	case ".py":
		return "python"
	case ".ts", ".tsx":
		return "typescript"
	case ".js", ".jsx":
		return "javascript"
	case ".rs":
		return "rust"
	default:
		return "go"
	}
}

func moduleName(path, language string) string {
	path = strings.ReplaceAll(path, "\\", "/")
	switch language {
	case "python":
		return strings.TrimSuffix(strings.ReplaceAll(path, "/", "."), ".py")
	case "rust":
		return strings.TrimSuffix(path, ".rs")
	default:
		return path
	}
}
