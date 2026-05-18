package state

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	windowDuration  = 5 * time.Minute
	bypassThreshold = 3
)

type Record struct {
	Key     string    `json:"key"`
	Count   int       `json:"count"`
	LastRun time.Time `json:"last_run"`
}

type State struct {
	mu      sync.Mutex
	Records map[string]*Record `json:"records"`
	path    string
}

// Load reads persisted state from disk, returning a fresh state on any error.
func Load() (*State, error) {
	path := stateFilePath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &State{Records: make(map[string]*Record), path: path}, nil
		}
		return nil, err
	}
	var s State
	if err := json.Unmarshal(data, &s); err != nil {
		return &State{Records: make(map[string]*Record), path: path}, nil
	}
	s.path = path
	if s.Records == nil {
		s.Records = make(map[string]*Record)
	}
	return &s, nil
}

// Track records a command execution and returns how many times it ran within
// the time window. When count >= bypassThreshold, bypass is true.
func (s *State) Track(cmd string, args []string) (count int, bypass bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := commandKey(cmd, args)
	now := time.Now()

	rec, ok := s.Records[key]
	if !ok || now.Sub(rec.LastRun) > windowDuration {
		s.Records[key] = &Record{Key: key, Count: 1, LastRun: now}
		s.prune()
		_ = s.save()
		return 1, false
	}

	rec.Count++
	rec.LastRun = now
	s.prune()
	_ = s.save()
	return rec.Count, rec.Count >= bypassThreshold
}

func (s *State) prune() {
	cutoff := time.Now().Add(-windowDuration)
	for k, r := range s.Records {
		if r.LastRun.Before(cutoff) {
			delete(s.Records, k)
		}
	}
}

func (s *State) save() error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0644)
}

func commandKey(cmd string, args []string) string {
	full := cmd + " " + strings.Join(args, " ")
	sum := md5.Sum([]byte(full))
	return fmt.Sprintf("%x", sum)
}

func stateFilePath() string {
	if dir := os.Getenv("XDG_CONFIG_HOME"); dir != "" {
		return filepath.Join(dir, "gotk", "state.json")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(os.TempDir(), "gotk_state.json")
	}
	return filepath.Join(home, ".config", "gotk", "state.json")
}
