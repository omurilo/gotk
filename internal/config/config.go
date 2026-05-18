package config

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// TeeMode controls when raw output is written to disk.
type TeeMode string

const (
	TeeModeAlways   TeeMode = "always"
	TeeModeFailures TeeMode = "failures"
	TeeModeNever    TeeMode = "never"
)

// Config is the global gotk configuration (~/.config/gotk/config.json).
type Config struct {
	Tee    TeeConfig    `json:"tee"`
	Hooks  HooksConfig  `json:"hooks"`
	Filter FilterConfig `json:"filter"`
}

type TeeConfig struct {
	// Mode is "always" | "failures" | "never". Default: "failures".
	Mode TeeMode `json:"mode"`
}

type HooksConfig struct {
	// ExcludeCommands lists commands the hook will never wrap with gotk.
	ExcludeCommands []string `json:"exclude_commands"`
}

type FilterConfig struct {
	// MaxGrepResults caps grep-style output lines (0 = unlimited).
	MaxGrepResults int `json:"max_grep_results"`
	// MaxStackFrames caps stack trace depth per block (0 = unlimited).
	MaxStackFrames int `json:"max_stack_frames"`
}

// ProjectRule is a custom filter rule from .gotk/filters.json.
type ProjectRule struct {
	// Match is a regex pattern applied to each output line.
	Match string `json:"match"`
	// Action is "suppress" | "collapse" | "keep".
	Action string `json:"action"`
}

// ProjectFilters holds project-level filter rules with a trust hash.
type ProjectFilters struct {
	TrustHash string        `json:"trust_hash"`
	Rules     []ProjectRule `json:"rules"`
}

// defaults returns a Config populated with production-safe defaults.
func defaults() Config {
	return Config{
		Tee:   TeeConfig{Mode: TeeModeFailures},
		Hooks: HooksConfig{},
		Filter: FilterConfig{
			MaxGrepResults: 30,
			MaxStackFrames: 5,
		},
	}
}

// Load reads the global config. Missing file returns defaults.
func Load() Config {
	cfg := defaults()
	path := globalConfigPath()

	data, err := os.ReadFile(path)
	if err != nil {
		return cfg
	}
	// Partial decode: only override fields present in the file.
	_ = json.Unmarshal(data, &cfg)

	if cfg.Tee.Mode == "" {
		cfg.Tee.Mode = TeeModeFailures
	}
	return cfg
}

// LoadProjectFilters reads .gotk/filters.json from the current directory.
// Returns nil if the file does not exist or is not trusted.
// NOTE: prefer TOML filters (.gotk/filters.toml) — this JSON loader is kept
// for backward compatibility only.
func LoadProjectFilters() *ProjectFilters {
	path := ".gotk/filters.json"
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	var pf ProjectFilters
	if err := json.Unmarshal(data, &pf); err != nil {
		return nil
	}

	if !isTrusted(path, data, pf.TrustHash) {
		fmt.Fprintf(os.Stderr,
			"[gotk] aviso: .gotk/filters.json não está confiável — execute 'gotk trust' para ativar\n")
		return nil
	}
	return &pf
}

// TrustToml records the SHA-256 of a TOML filter file so it is auto-loaded.
// path must be a .toml filter file (e.g. ".gotk/filters.toml").
func TrustToml(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("ler %s: %w", path, err)
	}

	hash := sha256Hash(data)
	trustedPath := trustedFilePath()
	if err := os.MkdirAll(filepath.Dir(trustedPath), 0755); err != nil {
		return err
	}

	trusted := make(map[string]string)
	if existing, err := os.ReadFile(trustedPath); err == nil {
		_ = json.Unmarshal(existing, &trusted)
	}

	abs, _ := filepath.Abs(path)
	trusted[abs] = hash
	out, _ := json.MarshalIndent(trusted, "", "  ")

	if err := os.WriteFile(trustedPath, out, 0644); err != nil {
		return err
	}
	fmt.Printf("[gotk] confiança registrada para %s (hash: %s...)\n", abs, hash[:24])
	return nil
}

// Trust computes the SHA-256 of .gotk/filters.json (excluding the trust_hash
// field) and stores it in ~/.config/gotk/trusted.json.
func Trust() error {
	path := ".gotk/filters.json"
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("ler %s: %w", path, err)
	}

	// Decode, remove trust_hash, re-encode deterministically.
	var obj map[string]any
	if err := json.Unmarshal(data, &obj); err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}
	delete(obj, "trust_hash")
	canonical, err := json.Marshal(obj)
	if err != nil {
		return err
	}

	hash := sha256Hash(canonical)

	// Persist hash in trusted.json
	trustedPath := trustedFilePath()
	if err := os.MkdirAll(filepath.Dir(trustedPath), 0755); err != nil {
		return err
	}

	trusted := make(map[string]string)
	if existing, err := os.ReadFile(trustedPath); err == nil {
		_ = json.Unmarshal(existing, &trusted)
	}

	abs, _ := filepath.Abs(path)
	trusted[abs] = hash
	out, _ := json.MarshalIndent(trusted, "", "  ")

	if err := os.WriteFile(trustedPath, out, 0644); err != nil {
		return err
	}
	fmt.Printf("[gotk] confiança registrada para %s (hash: %s)\n", abs, hash[:16]+"...")
	return nil
}

func isTrusted(path string, data []byte, storedHash string) bool {
	if storedHash == "" {
		return false
	}

	// Same logic as Trust(): strip trust_hash, re-encode, compare.
	var obj map[string]any
	if err := json.Unmarshal(data, &obj); err != nil {
		return false
	}
	delete(obj, "trust_hash")
	canonical, err := json.Marshal(obj)
	if err != nil {
		return false
	}
	computed := sha256Hash(canonical)

	// Also check the trusted.json database.
	trustedPath := trustedFilePath()
	dbData, err := os.ReadFile(trustedPath)
	if err != nil {
		return false
	}
	var trusted map[string]string
	if err := json.Unmarshal(dbData, &trusted); err != nil {
		return false
	}
	abs, _ := filepath.Abs(path)
	dbHash, ok := trusted[abs]
	if !ok {
		return false
	}
	return dbHash == computed && (storedHash == "" || storedHash == computed)
}

func sha256Hash(data []byte) string {
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func globalConfigPath() string {
	if dir := os.Getenv("XDG_CONFIG_HOME"); dir != "" {
		return filepath.Join(dir, "gotk", "config.json")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(os.TempDir(), "gotk_config.json")
	}
	return filepath.Join(home, ".config", "gotk", "config.json")
}

func trustedFilePath() string {
	if dir := os.Getenv("XDG_CONFIG_HOME"); dir != "" {
		return filepath.Join(dir, "gotk", "trusted.json")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(os.TempDir(), "gotk_trusted.json")
	}
	return filepath.Join(home, ".config", "gotk", "trusted.json")
}
