package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type persistedState struct {
	Offsets map[string]int64 `json:"offsets"`
}

func loadOffsets(path string) (map[string]int64, error) {
	if path == "" {
		return map[string]int64{}, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]int64{}, nil
		}
		return nil, fmt.Errorf("read state file: %w", err)
	}
	var state persistedState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("parse state file: %w", err)
	}
	if state.Offsets == nil {
		state.Offsets = map[string]int64{}
	}
	return state.Offsets, nil
}

func saveOffsets(path string, offsets map[string]int64) error {
	if path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create state directory: %w", err)
	}
	body, err := json.MarshalIndent(persistedState{Offsets: offsets}, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal state file: %w", err)
	}
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, body, 0o600); err != nil {
		return fmt.Errorf("write temp state file: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("replace state file: %w", err)
	}
	return nil
}
