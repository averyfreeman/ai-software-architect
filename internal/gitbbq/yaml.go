package gitbbq

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

func readYAML(path string, target any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}
	return nil
}

func marshalYAML(value any) ([]byte, error) {
	data, err := yaml.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("marshal YAML: %w", err)
	}
	return data, nil
}

func writeGenerated(root, relative string, data []byte, force bool) (bool, error) {
	clean := filepath.Clean(filepath.FromSlash(relative))
	if filepath.IsAbs(clean) || clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(os.PathSeparator)) {
		return false, fmt.Errorf("refusing unsafe generated path %q", relative)
	}
	path := filepath.Join(root, clean)
	if info, err := os.Lstat(path); err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			return false, fmt.Errorf("refusing to write symlink %s", relative)
		}
		if !force {
			return false, nil
		}
	} else if !os.IsNotExist(err) {
		return false, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return false, err
	}
	temporary, err := os.CreateTemp(filepath.Dir(path), ".gitbbq-*")
	if err != nil {
		return false, err
	}
	temporaryName := temporary.Name()
	defer os.Remove(temporaryName)
	if err := temporary.Chmod(0o644); err != nil {
		temporary.Close()
		return false, err
	}
	if _, err := temporary.Write(data); err != nil {
		temporary.Close()
		return false, err
	}
	if err := temporary.Close(); err != nil {
		return false, err
	}
	if err := os.Rename(temporaryName, path); err != nil {
		return false, err
	}
	return true, nil
}

func loadManifest(root string) (Manifest, error) {
	var manifest Manifest
	if err := readYAML(filepath.Join(root, ManifestFilename), &manifest); err != nil {
		return Manifest{}, err
	}
	if err := ValidateManifest(manifest); err != nil {
		return Manifest{}, err
	}
	return manifest, nil
}

func ReadManifest(root string) (Manifest, error) {
	return loadManifest(root)
}

func loadGitHabits(root string) (GitHabits, error) {
	var config GitHabits
	if err := readYAML(filepath.Join(root, GitHabitsFilename), &config); err != nil {
		return GitHabits{}, err
	}
	if err := ValidateGitHabits(config); err != nil {
		return GitHabits{}, err
	}
	return config, nil
}

func ReadGitHabits(root string) (GitHabits, error) {
	return loadGitHabits(root)
}
