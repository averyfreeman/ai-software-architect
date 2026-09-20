package codexruntime

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type stateStore struct {
	root string
	now  func() time.Time
}

func newStateStore(options Options) stateStore {
	now := options.Now
	if now == nil {
		now = time.Now
	}
	return stateStore{root: filepath.Join(options.PluginData, "control-plane"), now: now}
}

func stateDigest(value string) string {
	digest := sha256.Sum256([]byte(value))
	return hex.EncodeToString(digest[:])
}

func (store stateStore) contextPath(sessionID, turnID string) (string, error) {
	if sessionID == "" || turnID == "" {
		return "", errors.New("hook session_id and turn_id are required")
	}
	return filepath.Join(store.root, "turn-"+stateDigest(sessionID+"\x00"+turnID)+".json"), nil
}

func (store stateStore) continuationPath(sessionID string) (string, error) {
	if sessionID == "" {
		return "", errors.New("hook session_id is required")
	}
	return filepath.Join(store.root, "continuation-"+stateDigest(sessionID)+".json"), nil
}

func (store stateStore) checkpointPath(sessionID string) (string, error) {
	if sessionID == "" {
		return "", errors.New("hook session_id is required")
	}
	return filepath.Join(store.root, "workflow-"+stateDigest(sessionID)+".json"), nil
}

func (store stateStore) write(path string, value any) error {
	if err := os.MkdirAll(store.root, 0o755); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(store.root, ".state-*.tmp")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	data, err := json.Marshal(value)
	if err == nil {
		_, err = temporary.Write(append(data, '\n'))
	}
	if closeErr := temporary.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		if removeErr := os.Remove(path); removeErr != nil && !os.IsNotExist(removeErr) {
			return err
		}
		if renameErr := os.Rename(temporaryPath, path); renameErr != nil {
			return renameErr
		}
	}
	now := store.now()
	return os.Chtimes(path, now, now)
}

func (store stateStore) loadContext(sessionID, turnID string) turnContext {
	path, err := store.contextPath(sessionID, turnID)
	if err != nil {
		return turnContext{Route: InactiveRoute}
	}
	var context turnContext
	data, err := os.ReadFile(path)
	if err != nil || json.Unmarshal(data, &context) != nil || context.Route == "" {
		return turnContext{Route: InactiveRoute}
	}
	if info, statErr := os.Stat(path); statErr == nil && store.now().Sub(info.ModTime()) > MaxStateAge {
		_ = os.Remove(path)
		return turnContext{Route: InactiveRoute}
	}
	return context
}

func (store stateStore) saveContext(sessionID, turnID string, context turnContext) error {
	path, err := store.contextPath(sessionID, turnID)
	if err != nil {
		return err
	}
	return store.write(path, context)
}

func (store stateStore) clearContext(sessionID, turnID string) {
	if path, err := store.contextPath(sessionID, turnID); err == nil {
		_ = os.Remove(path)
	}
}

func (store stateStore) saveCheckpoint(sessionID string, value checkpoint) error {
	path, err := store.checkpointPath(sessionID)
	if err != nil {
		return err
	}
	return store.write(path, value)
}

func (store stateStore) loadCheckpoint(sessionID string) (checkpoint, bool) {
	path, err := store.checkpointPath(sessionID)
	if err != nil {
		return checkpoint{}, false
	}
	var value checkpoint
	data, err := os.ReadFile(path)
	if err != nil || json.Unmarshal(data, &value) != nil || value.Phase == "" {
		return checkpoint{}, false
	}
	return value, true
}

func (store stateStore) saveContinuation(sessionID string, value continuation) error {
	path, err := store.continuationPath(sessionID)
	if err != nil {
		return err
	}
	return store.write(path, value)
}

func (store stateStore) consumeContinuation(sessionID string) (continuation, bool) {
	path, err := store.continuationPath(sessionID)
	if err != nil {
		return continuation{}, false
	}
	defer os.Remove(path)
	info, err := os.Stat(path)
	if err != nil || store.now().Sub(info.ModTime()) > MaxContinuationAge {
		return continuation{}, false
	}
	var value continuation
	data, err := os.ReadFile(path)
	if err != nil || json.Unmarshal(data, &value) != nil || !value.Context.Active {
		return continuation{}, false
	}
	return value, true
}

func (store stateStore) cancelContinuation(sessionID string) {
	if path, err := store.continuationPath(sessionID); err == nil {
		_ = os.Remove(path)
	}
}

func (store stateStore) cleanup() {
	entries, err := os.ReadDir(store.root)
	if err != nil {
		return
	}
	type candidate struct {
		path     string
		modified time.Time
	}
	candidates := make([]candidate, 0)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		info, err := entry.Info()
		if err == nil {
			candidates = append(candidates, candidate{
				path:     filepath.Join(store.root, entry.Name()),
				modified: info.ModTime(),
			})
		}
	}
	sort.Slice(candidates, func(left, right int) bool {
		return candidates[left].modified.After(candidates[right].modified)
	})
	now := store.now()
	for index, item := range candidates {
		if index >= MaxStateFiles || now.Sub(item.modified) > MaxStateAge {
			_ = os.Remove(item.path)
		}
	}
}
