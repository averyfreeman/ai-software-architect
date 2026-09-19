package gitbbq

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func writeOwnershipLedger(root string, created []string, force bool) (bool, error) {
	path := filepath.Join(root, filepath.FromSlash(OwnershipFilename))
	if _, err := os.Lstat(path); err == nil && !force {
		return false, nil
	} else if err != nil && !os.IsNotExist(err) {
		return false, err
	}
	files := make(map[string]string, len(created))
	for _, relative := range created {
		if relative == OwnershipFilename {
			continue
		}
		clean, err := safeGeneratedPath(relative)
		if err != nil {
			return false, err
		}
		digest, err := hashFile(filepath.Join(root, filepath.FromSlash(clean)))
		if err != nil {
			return false, err
		}
		files[clean] = digest
	}
	data, err := json.MarshalIndent(OwnershipLedger{Version: 1, Files: files}, "", "  ")
	if err != nil {
		return false, err
	}
	data = append(data, '\n')
	return writeGenerated(root, OwnershipFilename, data, force)
}

// AssessUninstall reports only unchanged files owned by Git BBQ as removable.
// Modified, symlinked, and non-regular paths are conflicts and are preserved.
func AssessUninstall(root string) (UninstallAssessment, error) {
	root = filepath.Clean(root)
	if root == "." {
		root, _ = os.Getwd()
	}
	ledger, err := readOwnershipLedger(root)
	if err != nil {
		return UninstallAssessment{}, err
	}
	assessment := UninstallAssessment{Mode: "uninstall-assessment", Root: root, Removable: []string{}, Conflicts: []string{}, Missing: []string{}, ReadOnly: true}
	paths := make([]string, 0, len(ledger.Files))
	for relative := range ledger.Files {
		paths = append(paths, relative)
	}
	sort.Strings(paths)
	for _, relative := range paths {
		clean, err := safeGeneratedPath(relative)
		if err != nil {
			assessment.Conflicts = append(assessment.Conflicts, relative)
			continue
		}
		path := filepath.Join(root, filepath.FromSlash(clean))
		info, err := os.Lstat(path)
		if os.IsNotExist(err) {
			assessment.Missing = append(assessment.Missing, clean)
			continue
		}
		if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
			assessment.Conflicts = append(assessment.Conflicts, clean)
			continue
		}
		digest, err := hashFile(path)
		if err != nil || digest != ledger.Files[relative] {
			assessment.Conflicts = append(assessment.Conflicts, clean)
			continue
		}
		assessment.Removable = append(assessment.Removable, clean)
	}
	return assessment, nil
}

// Uninstall removes only unchanged files recorded in the ownership ledger.
// It never follows symlinks, removes user-modified files, or touches legacy
// project artifacts outside the ledger.
func Uninstall(root string) (UninstallResult, error) {
	assessment, err := AssessUninstall(root)
	if err != nil {
		return UninstallResult{}, err
	}
	result := UninstallResult{Root: assessment.Root, Removed: []string{}, Preserved: append([]string(nil), assessment.Conflicts...), Missing: append([]string(nil), assessment.Missing...)}
	for _, relative := range assessment.Removable {
		if err := os.Remove(filepath.Join(assessment.Root, filepath.FromSlash(relative))); err != nil {
			return result, fmt.Errorf("remove %s: %w", relative, err)
		}
		result.Removed = append(result.Removed, relative)
	}
	ledgerPath := filepath.Join(assessment.Root, filepath.FromSlash(OwnershipFilename))
	if len(assessment.Conflicts) > 0 {
		result.Preserved = append(result.Preserved, OwnershipFilename)
	} else if err := os.Remove(ledgerPath); err == nil {
		result.Removed = append(result.Removed, OwnershipFilename)
	} else if !os.IsNotExist(err) {
		return result, fmt.Errorf("remove %s: %w", OwnershipFilename, err)
	}
	removeEmptyParents(assessment.Root, result.Removed)
	sort.Strings(result.Removed)
	sort.Strings(result.Preserved)
	sort.Strings(result.Missing)
	return result, nil
}

func readOwnershipLedger(root string) (OwnershipLedger, error) {
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(OwnershipFilename)))
	if err != nil {
		return OwnershipLedger{}, fmt.Errorf("read %s: %w", OwnershipFilename, err)
	}
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	var ledger OwnershipLedger
	if err := decoder.Decode(&ledger); err != nil {
		return OwnershipLedger{}, fmt.Errorf("parse %s: %w", OwnershipFilename, err)
	}
	if ledger.Version != 1 {
		return OwnershipLedger{}, fmt.Errorf("unsupported ownership ledger version %d", ledger.Version)
	}
	if ledger.Files == nil {
		return OwnershipLedger{}, fmt.Errorf("ownership ledger files are required")
	}
	return ledger, nil
}

func hashFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:]), nil
}

func safeGeneratedPath(relative string) (string, error) {
	clean := filepath.Clean(filepath.FromSlash(relative))
	if filepath.IsAbs(clean) || clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("refusing unsafe owned path %q", relative)
	}
	return filepath.ToSlash(clean), nil
}

func removeEmptyParents(root string, removed []string) {
	directories := make(map[string]bool)
	for _, relative := range removed {
		parent := filepath.Dir(filepath.FromSlash(relative))
		for parent != "." && parent != string(filepath.Separator) {
			directories[parent] = true
			parent = filepath.Dir(parent)
		}
	}
	paths := make([]string, 0, len(directories))
	for relative := range directories {
		paths = append(paths, relative)
	}
	sort.Slice(paths, func(i, j int) bool {
		return strings.Count(paths[i], string(filepath.Separator)) > strings.Count(paths[j], string(filepath.Separator))
	})
	for _, relative := range paths {
		_ = os.Remove(filepath.Join(root, filepath.FromSlash(relative)))
	}
}
