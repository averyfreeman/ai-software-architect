package architect

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

const (
	maxWorkspaceFiles = 500
	maxWorkspaceBytes = 5_000_000
	maxSourceBytes    = 500_000
)

var protectedPathParts = map[string]struct{}{
	".git": {}, ".ssh": {}, ".aws": {}, ".azure": {},
}

func projectRoot(path string) (string, error) {
	if strings.TrimSpace(path) == "" {
		path = "."
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		if os.IsNotExist(err) {
			return abs, nil
		}
		return "", err
	}
	info, err := os.Stat(resolved)
	if err != nil || !info.IsDir() {
		return "", fmt.Errorf("project root is not a directory: %s", path)
	}
	return resolved, nil
}

func relativeSafe(root, relative string) (string, error) {
	if relative == "" || strings.IndexByte(relative, 0) >= 0 {
		return "", fmt.Errorf("relative path is empty or contains a null byte")
	}
	normalized := filepath.Clean(filepath.FromSlash(strings.ReplaceAll(relative, "\\", "/")))
	if filepath.IsAbs(normalized) || hasWindowsDrivePrefix(normalized) || normalized == "." || normalized == ".." || strings.HasPrefix(normalized, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path must remain project-relative: %s", relative)
	}
	for _, part := range strings.Split(filepath.ToSlash(normalized), "/") {
		if part == "" || part == "." {
			continue
		}
		if _, protected := protectedPathParts[part]; protected || strings.HasPrefix(part, ".env") || strings.HasSuffix(part, ".pem") || strings.HasSuffix(part, ".key") || strings.Contains(strings.ToLower(part), "credential") || strings.Contains(strings.ToLower(part), "secret") {
			return "", fmt.Errorf("protected path is not readable: %s", relative)
		}
	}
	joined := filepath.Join(root, normalized)
	rootAbs, _ := filepath.Abs(root)
	joinedAbs, _ := filepath.Abs(joined)
	if joinedAbs != rootAbs && !strings.HasPrefix(joinedAbs, rootAbs+string(filepath.Separator)) {
		return "", fmt.Errorf("path escapes project root: %s", relative)
	}
	return filepath.ToSlash(normalized), nil
}

func hasWindowsDrivePrefix(path string) bool {
	if len(path) < 2 || path[1] != ':' {
		return false
	}
	return (path[0] >= 'A' && path[0] <= 'Z') || (path[0] >= 'a' && path[0] <= 'z')
}

func ensureWritableTarget(root, target string) error {
	return ensureNoSymlinkTarget(root, target)
}

func ensureReadableTarget(root, target string) error {
	return ensureNoSymlinkTarget(root, target)
}

func ensureNoSymlinkTarget(root, target string) error {
	parentRelative, err := filepath.Rel(root, filepath.Dir(target))
	if err != nil {
		return err
	}
	current := root
	if parentRelative != "." {
		for _, part := range strings.Split(parentRelative, string(filepath.Separator)) {
			if part == "" || part == "." {
				continue
			}
			current = filepath.Join(current, part)
			info, statErr := os.Lstat(current)
			if os.IsNotExist(statErr) {
				break
			}
			if statErr != nil {
				return statErr
			}
			if info.Mode()&os.ModeSymlink != 0 {
				return fmt.Errorf("refusing to write through symlinked directory: %s", current)
			}
			if !info.IsDir() {
				return fmt.Errorf("write parent is not a directory: %s", current)
			}
		}
	}
	info, err := os.Lstat(target)
	if err == nil && info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("refusing to overwrite symlink: %s", target)
	}
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func readProjectText(root, relative string, suffixes map[string]struct{}) (string, error) {
	relative, err := relativeSafe(root, relative)
	if err != nil {
		return "", err
	}
	path := filepath.Join(root, filepath.FromSlash(relative))
	info, err := os.Lstat(path)
	if err != nil {
		return "", err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return "", fmt.Errorf("path is not a regular file: %s", relative)
	}
	if info.Size() > maxSourceBytes {
		return "", fmt.Errorf("file exceeds %d-byte source budget: %s", maxSourceBytes, relative)
	}
	if len(suffixes) > 0 {
		if _, ok := suffixes[strings.ToLower(filepath.Ext(relative))]; !ok {
			return "", fmt.Errorf("unsupported source suffix: %s", relative)
		}
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	if strings.IndexByte(string(data), 0) >= 0 {
		return "", fmt.Errorf("binary content is not supported: %s", relative)
	}
	return string(data), nil
}

type sourceFile struct {
	RelativePath string `json:"relative_path"`
	Content      string `json:"content"`
}

func collectSourceFiles(root string, roots []string, extensions map[string]struct{}) ([]sourceFile, int, error) {
	if len(roots) == 0 {
		roots = []string{"."}
	}
	files := make([]sourceFile, 0)
	seen := map[string]struct{}{}
	totalBytes := 0
	skipped := 0
	for _, requested := range roots {
		relative, err := relativeSafe(root, requested)
		if err != nil {
			return nil, skipped, err
		}
		base := filepath.Join(root, filepath.FromSlash(relative))
		info, err := os.Stat(base)
		if err != nil {
			return nil, skipped, err
		}
		walkErr := filepath.WalkDir(base, func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if len(files) >= maxWorkspaceFiles || totalBytes >= maxWorkspaceBytes {
				return fs.SkipAll
			}
			if entry.IsDir() {
				if path != base && strings.HasPrefix(entry.Name(), ".") {
					return fs.SkipDir
				}
				return nil
			}
			if entry.Type()&os.ModeSymlink != 0 {
				skipped++
				return nil
			}
			ext := strings.ToLower(filepath.Ext(entry.Name()))
			if _, ok := extensions[ext]; !ok {
				return nil
			}
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			rel = filepath.ToSlash(rel)
			if _, ok := seen[rel]; ok {
				return nil
			}
			content, err := readProjectText(root, rel, extensions)
			if err != nil {
				skipped++
				return nil
			}
			bytes := len(content)
			if totalBytes+bytes > maxWorkspaceBytes {
				return fs.SkipAll
			}
			seen[rel] = struct{}{}
			files = append(files, sourceFile{RelativePath: rel, Content: content})
			totalBytes += bytes
			return nil
		})
		if walkErr != nil && walkErr != fs.SkipAll {
			return nil, skipped, walkErr
		}
		_ = info
		if len(files) >= maxWorkspaceFiles || totalBytes >= maxWorkspaceBytes {
			break
		}
	}
	return files, skipped, nil
}
