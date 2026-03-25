package provider

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type persistedProviderState struct {
	CooldownUntil time.Time `json:"cooldown_until,omitempty"`
}

type providerStateFile struct {
	Providers map[string]persistedProviderState `json:"providers"`
}

var providerStateMu sync.Mutex

func providerStatePath() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ""
	}
	return filepath.Join(home, ".local", "share", "freeapi", "provider-state.json")
}

func readProviderState() (*providerStateFile, error) {
	path := providerStatePath()
	if path == "" {
		return &providerStateFile{Providers: map[string]persistedProviderState{}}, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &providerStateFile{Providers: map[string]persistedProviderState{}}, nil
		}
		return nil, err
	}

	var state providerStateFile
	if err := json.Unmarshal(data, &state); err != nil {
		return &providerStateFile{Providers: map[string]persistedProviderState{}}, nil
	}
	if state.Providers == nil {
		state.Providers = map[string]persistedProviderState{}
	}
	return &state, nil
}

func writeProviderState(state *providerStateFile) error {
	path := providerStatePath()
	if path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

func loadProviderCooldown(name string) time.Time {
	providerStateMu.Lock()
	defer providerStateMu.Unlock()

	state, err := readProviderState()
	if err != nil {
		return time.Time{}
	}
	entry, ok := state.Providers[name]
	if !ok {
		return time.Time{}
	}
	return entry.CooldownUntil
}

func persistProviderCooldown(name string, until time.Time) {
	providerStateMu.Lock()
	defer providerStateMu.Unlock()

	state, err := readProviderState()
	if err != nil {
		return
	}
	if state.Providers == nil {
		state.Providers = map[string]persistedProviderState{}
	}
	state.Providers[name] = persistedProviderState{CooldownUntil: until}
	_ = writeProviderState(state)
}

func clearProviderCooldown(name string) {
	providerStateMu.Lock()
	defer providerStateMu.Unlock()

	state, err := readProviderState()
	if err != nil {
		return
	}
	delete(state.Providers, name)
	_ = writeProviderState(state)
}
