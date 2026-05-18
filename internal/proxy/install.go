package proxy

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Install copies the hook script to ~/.claude/hooks/ and patches
// ~/.claude/settings.json to register the PreToolUse hook.
func Install() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("resolve home dir: %w", err)
	}

	hooksDir := filepath.Join(home, ".claude", "hooks")
	if err := os.MkdirAll(hooksDir, 0755); err != nil {
		return fmt.Errorf("create hooks dir: %w", err)
	}

	hookDest := filepath.Join(hooksDir, "gotk-rewrite.sh")
	if err := os.WriteFile(hookDest, []byte(hookScript), 0755); err != nil {
		return fmt.Errorf("write hook script: %w", err)
	}
	fmt.Printf("[gotk] hook instalado em %s\n", hookDest)

	settingsPath := filepath.Join(home, ".claude", "settings.json")
	if err := patchSettings(settingsPath, hookDest); err != nil {
		return fmt.Errorf("patch settings: %w", err)
	}
	fmt.Printf("[gotk] hook registrado em %s\n", settingsPath)
	fmt.Println("[gotk] instalação concluída. Reinicie o Claude Code para ativar.")
	return nil
}

func patchSettings(path, hookScript string) error {
	var cfg map[string]any

	data, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		cfg = make(map[string]any)
	} else {
		if err := json.Unmarshal(data, &cfg); err != nil {
			return fmt.Errorf("parse settings.json: %w", err)
		}
	}

	hooks, _ := cfg["hooks"].(map[string]any)
	if hooks == nil {
		hooks = make(map[string]any)
	}

	preToolUse, _ := hooks["PreToolUse"].([]any)
	// Remove any existing gotk entry to avoid duplicates
	filtered := preToolUse[:0]
	for _, item := range preToolUse {
		m, ok := item.(map[string]any)
		if !ok {
			filtered = append(filtered, item)
			continue
		}
		hooksArr, _ := m["hooks"].([]any)
		hasGotk := false
		for _, h := range hooksArr {
			hm, ok := h.(map[string]any)
			if !ok {
				continue
			}
			if cmd, _ := hm["command"].(string); cmd == hookScript {
				hasGotk = true
				break
			}
		}
		if !hasGotk {
			filtered = append(filtered, item)
		}
	}

	gotkEntry := map[string]any{
		"matcher": "Bash",
		"hooks": []any{
			map[string]any{
				"type":    "command",
				"command": hookScript,
			},
		},
	}
	filtered = append(filtered, gotkEntry)
	hooks["PreToolUse"] = filtered
	cfg["hooks"] = hooks

	out, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, out, 0644)
}

// hookScript is the content written to ~/.claude/hooks/gotk-rewrite.sh.
const hookScript = `#!/usr/bin/env bash
# gotk-rewrite.sh — Claude Code PreToolUse hook
# Rewrites Bash commands through gotk for token compression.
set -euo pipefail
GOTK_BIN="${GOTK_BIN:-gotk}"
if command -v "$GOTK_BIN" >/dev/null 2>&1; then
    "$GOTK_BIN" hook
else
    exit 0
fi
`
