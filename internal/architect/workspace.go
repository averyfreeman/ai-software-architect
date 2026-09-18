package architect

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"
)

type WorkspaceError struct {
	Code         string `json:"code"`
	Message      string `json:"message"`
	RelativePath string `json:"relative_path,omitempty"`
}

func (e *WorkspaceError) Error() string {
	if e.RelativePath == "" {
		return e.Message
	}
	return e.Message + ": " + e.RelativePath
}

var drivePathPattern = regexp.MustCompile(`^[A-Za-z]:[\\/]`)

var protectedExactNames = map[string]bool{
	".env": true, ".npmrc": true, ".pypirc": true,
	"credentials": true, "credentials.json": true,
	"id_rsa": true, "id_ed25519": true,
}

func NormalizeRelativePath(path string) (string, error) {
	if path == "" || strings.IndexFunc(path, func(r rune) bool { return r < 32 }) >= 0 {
		return "", &WorkspaceError{Code: "invalid-input", Message: "relative path is empty or invalid"}
	}
	normalized := strings.ReplaceAll(path, "\\", "/")
	if filepath.IsAbs(path) || strings.HasPrefix(normalized, "/") || drivePathPattern.MatchString(normalized) {
		return "", &WorkspaceError{Code: "boundary-violation", Message: "path must remain workspace-relative", RelativePath: path}
	}
	parts := make([]string, 0, len(strings.Split(normalized, "/")))
	for _, part := range strings.Split(normalized, "/") {
		switch part {
		case "", ".":
			continue
		case "..":
			return "", &WorkspaceError{Code: "boundary-violation", Message: "path must remain workspace-relative", RelativePath: path}
		default:
			parts = append(parts, part)
		}
	}
	if len(parts) == 0 {
		return ".", nil
	}
	return strings.Join(parts, "/"), nil
}

func IsProtectedPath(relative string) bool {
	normalized := strings.ToLower(strings.ReplaceAll(relative, "\\", "/"))
	parts := strings.Split(normalized, "/")
	for index, part := range parts {
		if part == ".git" || part == ".ssh" || part == ".aws" || part == ".azure" || (part == ".config" && index+1 < len(parts) && parts[index+1] == "gcloud") {
			return true
		}
	}
	name := parts[len(parts)-1]
	if protectedExactNames[name] || name == ".env" || strings.HasPrefix(name, ".env.") || strings.HasPrefix(name, "credentials") || strings.HasPrefix(name, "service-account") || strings.HasPrefix(name, "id_rsa") {
		return true
	}
	ext := strings.ToLower(filepath.Ext(name))
	return ext == ".pem" || ext == ".key" || ext == ".p12" || ext == ".pfx"
}

func ValidateInlinePythonPath(path string) (string, error) {
	relative, err := NormalizeRelativePath(path)
	if err != nil {
		return "", err
	}
	for _, part := range strings.Split(relative, "/") {
		if strings.HasPrefix(part, ".") {
			return "", &WorkspaceError{Code: "protected-path", Message: "hidden inline source paths are not accepted", RelativePath: path}
		}
	}
	if IsProtectedPath(relative) {
		return "", &WorkspaceError{Code: "protected-path", Message: "protected inline source path is not accepted", RelativePath: relative}
	}
	if strings.ToLower(filepath.Ext(relative)) != ".py" {
		return "", &WorkspaceError{Code: "unsupported-format", Message: "inline source file type is not supported", RelativePath: relative}
	}
	return relative, nil
}

type WorkspaceReader struct {
	Root string
}

func NewWorkspaceReader(root string) (*WorkspaceReader, error) {
	resolved, err := filepath.Abs(root)
	if err != nil {
		return nil, &WorkspaceError{Code: "workspace-unavailable", Message: "workspace root is unavailable"}
	}
	if isReparsePoint(resolved) {
		return nil, &WorkspaceError{Code: "workspace-unavailable", Message: "workspace root is not a real local directory"}
	}
	resolved, err = filepath.EvalSymlinks(resolved)
	if err != nil {
		return nil, &WorkspaceError{Code: "workspace-unavailable", Message: "workspace root is unavailable"}
	}
	info, err := os.Stat(resolved)
	if err != nil || !info.IsDir() || isReparsePoint(resolved) {
		return nil, &WorkspaceError{Code: "workspace-unavailable", Message: "workspace root is not a real local directory"}
	}
	return &WorkspaceReader{Root: resolved}, nil
}

func (r *WorkspaceReader) candidate(relativePath string) (string, string, error) {
	relative, err := NormalizeRelativePath(relativePath)
	if err != nil {
		return "", "", err
	}
	if IsProtectedPath(relative) {
		return "", "", &WorkspaceError{Code: "protected-path", Message: "protected path is not readable", RelativePath: relative}
	}
	candidate := filepath.Join(r.Root, filepath.FromSlash(relative))
	if relative == "." {
		candidate = r.Root
	}
	if err := rejectIndirectionBetween(r.Root, candidate); err != nil {
		return "", "", &WorkspaceError{Code: "boundary-violation", Message: "filesystem indirection is not allowed", RelativePath: relative}
	}
	resolved, err := filepath.EvalSymlinks(candidate)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", "", &WorkspaceError{Code: "not-found", Message: "requested path does not exist", RelativePath: relative}
		}
		return "", "", &WorkspaceError{Code: "workspace-unavailable", Message: "requested path cannot be resolved", RelativePath: relative}
	}
	if !withinRoot(r.Root, resolved) {
		return "", "", &WorkspaceError{Code: "boundary-violation", Message: "resolved path escapes workspace", RelativePath: relative}
	}
	return relative, resolved, nil
}

func (r *WorkspaceReader) ReadText(relativePath string, allowedSuffixes map[string]bool) (string, error) {
	relative, resolved, err := r.candidate(relativePath)
	if err != nil {
		return "", err
	}
	if !allowedSuffixes[strings.ToLower(filepath.Ext(resolved))] {
		return "", &WorkspaceError{Code: "unsupported-format", Message: "file type is not supported", RelativePath: relative}
	}
	file, err := os.Open(resolved)
	if err != nil {
		return "", &WorkspaceError{Code: "not-found", Message: "requested path does not exist", RelativePath: relative}
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() || isReparsePoint(resolved) {
		return "", &WorkspaceError{Code: "unsupported-format", Message: "path is not a regular file", RelativePath: relative}
	}
	if info.Size() > MaxWorkspaceFileBytes {
		return "", &WorkspaceError{Code: "budget-exhausted", Message: "file exceeds the single-file budget", RelativePath: relative}
	}
	content, err := io.ReadAll(io.LimitReader(file, MaxWorkspaceFileBytes+1))
	if err != nil {
		return "", &WorkspaceError{Code: "unsupported-format", Message: "file cannot be read", RelativePath: relative}
	}
	if len(content) > MaxWorkspaceFileBytes {
		return "", &WorkspaceError{Code: "budget-exhausted", Message: "file exceeds the single-file budget", RelativePath: relative}
	}
	if strings.IndexByte(string(content), 0) >= 0 {
		return "", &WorkspaceError{Code: "unsupported-format", Message: "binary file is not supported", RelativePath: relative}
	}
	if !utf8.Valid(content) {
		return "", &WorkspaceError{Code: "unsupported-format", Message: "file is not valid UTF-8", RelativePath: relative}
	}
	return string(content), nil
}

func (r *WorkspaceReader) IterFiles(relativeRoots []string, suffixes map[string]bool) ([]string, error) {
	if len(relativeRoots) == 0 {
		relativeRoots = []string{"."}
	}
	seen := map[string]bool{}
	result := make([]string, 0)
	for _, root := range relativeRoots {
		relative, resolved, err := r.candidate(root)
		if err != nil {
			return nil, err
		}
		info, err := os.Stat(resolved)
		if err != nil {
			return nil, err
		}
		if info.IsDir() {
			err = filepath.WalkDir(resolved, func(path string, entry os.DirEntry, walkErr error) error {
				if walkErr != nil {
					return nil
				}
				if path != resolved && entry.IsDir() && strings.HasPrefix(entry.Name(), ".") {
					return filepath.SkipDir
				}
				if path != resolved && isReparsePoint(path) {
					if entry.IsDir() {
						return filepath.SkipDir
					}
					return nil
				}
				if entry.IsDir() || !entry.Type().IsRegular() {
					return nil
				}
				rel, relErr := filepath.Rel(r.Root, path)
				if relErr != nil {
					return nil
				}
				rel = filepath.ToSlash(rel)
				if rel == "." || seen[rel] || IsProtectedPath(rel) || !suffixes[strings.ToLower(filepath.Ext(rel))] {
					return nil
				}
				seen[rel] = true
				result = append(result, rel)
				if len(result) >= MaxWorkspaceFiles {
					return io.EOF
				}
				return nil
			})
			if err != nil && !errors.Is(err, io.EOF) {
				return nil, err
			}
		} else if info.Mode().IsRegular() && suffixes[strings.ToLower(filepath.Ext(resolved))] && !IsProtectedPath(relative) {
			seen[relative] = true
			result = append(result, relative)
		}
		if len(result) >= MaxWorkspaceFiles {
			break
		}
	}
	sort.Strings(result)
	return result, nil
}

func withinRoot(root, candidate string) bool {
	relative, err := filepath.Rel(root, candidate)
	if err != nil {
		return false
	}
	return relative == "." || (relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) && !filepath.IsAbs(relative))
}

func rejectIndirectionBetween(root, candidate string) error {
	relative, err := filepath.Rel(root, candidate)
	if err != nil || filepath.IsAbs(relative) {
		return fmt.Errorf("path is outside root")
	}
	current := root
	if relative == "." {
		return nil
	}
	for _, part := range strings.Split(relative, string(filepath.Separator)) {
		current = filepath.Join(current, part)
		if isReparsePoint(current) {
			return fmt.Errorf("filesystem indirection is not allowed")
		}
	}
	return nil
}
