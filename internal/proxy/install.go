package proxy

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// InstallOptions controls gotk init behaviour.
type InstallOptions struct {
	Agent     string // "claude" (default), "cursor", "gemini", "windsurf", "cline", "all"
	Global    bool   // install globally (default for hook-based agents)
	Uninstall bool
	Show      bool
	DryRun    bool
}

// Install sets up gotk hooks for the requested agent(s).
func Install(opts InstallOptions) error {
	if opts.Agent == "" {
		opts.Agent = "claude"
	}

	if opts.Show {
		return showStatus(opts)
	}

	agents := []string{opts.Agent}
	if opts.Agent == "all" {
		agents = []string{"claude", "cursor", "gemini", "windsurf", "cline"}
	}

	for _, ag := range agents {
		var err error
		if opts.Uninstall {
			err = uninstallAgent(ag, opts)
		} else {
			err = installAgent(ag, opts)
		}
		if err != nil {
			return fmt.Errorf("agent %s: %w", ag, err)
		}
	}
	return nil
}

func installAgent(agent string, opts InstallOptions) error {
	switch agent {
	case "claude":
		return installClaude(opts)
	case "cursor":
		return installCursor(opts)
	case "gemini":
		return installGemini(opts)
	case "windsurf":
		return installWindsurf(opts)
	case "cline":
		return installCline(opts)
	default:
		return fmt.Errorf("unknown agent %q (supported: claude, cursor, gemini, windsurf, cline, all)", agent)
	}
}

func uninstallAgent(agent string, opts InstallOptions) error {
	switch agent {
	case "claude":
		return uninstallClaude(opts)
	case "cursor":
		return uninstallCursor(opts)
	case "gemini":
		return uninstallGemini(opts)
	case "windsurf":
		fmt.Println("[gotk] windsurf: remove 'gotk' instructions from .windsurfrules manually")
		return nil
	case "cline":
		fmt.Println("[gotk] cline: remove 'gotk' instructions from .clinerules manually")
		return nil
	default:
		return fmt.Errorf("unknown agent %q", agent)
	}
}

// ─── Claude Code ──────────────────────────────────────────────────────────────

const claudeHookCommand = "gotk hook claude"

func installClaude(opts InstallOptions) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("resolve home dir: %w", err)
	}

	settingsPath := filepath.Join(home, ".claude", "settings.json")

	if opts.DryRun {
		fmt.Printf("[dry-run] would patch Claude Code settings: %s\n", settingsPath)
		fmt.Printf("[dry-run] command: %s\n", claudeHookCommand)
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(settingsPath), 0755); err != nil {
		return fmt.Errorf("create .claude dir: %w", err)
	}

	patched, err := patchClaudeSettings(settingsPath, claudeHookCommand)
	if err != nil {
		return fmt.Errorf("patch settings: %w", err)
	}

	if patched {
		fmt.Printf("[gotk] Claude Code: hook registered (%s)\n", settingsPath)
	} else {
		fmt.Printf("[gotk] Claude Code: hook already present (%s)\n", settingsPath)
	}
	fmt.Printf("[gotk] command: %s\n", claudeHookCommand)
	fmt.Println("[gotk] Restart Claude Code to activate. Test with: git status")
	return nil
}

func uninstallClaude(opts InstallOptions) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	settingsPath := filepath.Join(home, ".claude", "settings.json")
	if opts.DryRun {
		fmt.Printf("[dry-run] would remove gotk hook from %s\n", settingsPath)
		return nil
	}
	removed, err := removeClaudeHook(settingsPath)
	if err != nil {
		return err
	}
	if removed {
		fmt.Println("[gotk] Claude Code: hook removed. Restart Claude Code.")
	} else {
		fmt.Println("[gotk] Claude Code: hook was not installed")
	}
	return nil
}

func patchClaudeSettings(path, hookCmd string) (patched bool, err error) {
	var cfg map[string]any

	data, readErr := os.ReadFile(path)
	if readErr != nil {
		if !os.IsNotExist(readErr) {
			return false, readErr
		}
		cfg = make(map[string]any)
	} else {
		if jsonErr := json.Unmarshal(data, &cfg); jsonErr != nil {
			return false, fmt.Errorf("parse settings.json: %w", jsonErr)
		}
	}

	hooks, _ := cfg["hooks"].(map[string]any)
	if hooks == nil {
		hooks = make(map[string]any)
	}

	preToolUse, _ := hooks["PreToolUse"].([]any)

	// Idempotency: check if already present
	for _, item := range preToolUse {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		hooksArr, _ := m["hooks"].([]any)
		for _, h := range hooksArr {
			hm, ok := h.(map[string]any)
			if !ok {
				continue
			}
			if cmd, _ := hm["command"].(string); cmd == hookCmd {
				return false, nil
			}
		}
	}

	// Remove legacy script-based gotk entries
	filtered := preToolUse[:0]
	for _, item := range preToolUse {
		m, ok := item.(map[string]any)
		if !ok {
			filtered = append(filtered, item)
			continue
		}
		hooksArr, _ := m["hooks"].([]any)
		isLegacyGotk := false
		for _, h := range hooksArr {
			hm, ok := h.(map[string]any)
			if !ok {
				continue
			}
			if cmd, _ := hm["command"].(string); strings.Contains(cmd, "gotk") {
				isLegacyGotk = true
				break
			}
		}
		if !isLegacyGotk {
			filtered = append(filtered, item)
		}
	}

	filtered = append(filtered, map[string]any{
		"matcher": "Bash",
		"hooks": []any{
			map[string]any{"type": "command", "command": hookCmd},
		},
	})
	hooks["PreToolUse"] = filtered
	cfg["hooks"] = hooks

	out, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return false, err
	}
	return true, os.WriteFile(path, out, 0644)
}

func removeClaudeHook(path string) (bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}

	var cfg map[string]any
	if err := json.Unmarshal(data, &cfg); err != nil {
		return false, err
	}

	hooks, _ := cfg["hooks"].(map[string]any)
	if hooks == nil {
		return false, nil
	}
	preToolUse, _ := hooks["PreToolUse"].([]any)
	if len(preToolUse) == 0 {
		return false, nil
	}

	filtered := preToolUse[:0]
	removed := false
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
			if cmd, _ := hm["command"].(string); strings.Contains(cmd, "gotk") {
				hasGotk = true
				break
			}
		}
		if hasGotk {
			removed = true
		} else {
			filtered = append(filtered, item)
		}
	}

	if !removed {
		return false, nil
	}

	hooks["PreToolUse"] = filtered
	cfg["hooks"] = hooks
	out, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return false, err
	}
	return true, os.WriteFile(path, out, 0644)
}

// ─── Cursor ───────────────────────────────────────────────────────────────────

const cursorHookCommand = "gotk hook cursor"

func installCursor(opts InstallOptions) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	hooksJSONPath := filepath.Join(home, ".cursor", "hooks.json")

	if opts.DryRun {
		fmt.Printf("[dry-run] would patch Cursor hooks: %s\n", hooksJSONPath)
		fmt.Printf("[dry-run] command: %s\n", cursorHookCommand)
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(hooksJSONPath), 0755); err != nil {
		return err
	}

	patched, err := patchCursorHooks(hooksJSONPath)
	if err != nil {
		return fmt.Errorf("patch Cursor hooks.json: %w", err)
	}

	if patched {
		fmt.Printf("[gotk] Cursor: hook registered (%s)\n", hooksJSONPath)
	} else {
		fmt.Printf("[gotk] Cursor: hook already present (%s)\n", hooksJSONPath)
	}
	fmt.Printf("[gotk] command: %s\n", cursorHookCommand)
	fmt.Println("[gotk] Cursor reloads hooks.json automatically. Test with: git status")
	return nil
}

func uninstallCursor(opts InstallOptions) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	hooksJSONPath := filepath.Join(home, ".cursor", "hooks.json")
	if opts.DryRun {
		fmt.Printf("[dry-run] would remove gotk hook from %s\n", hooksJSONPath)
		return nil
	}
	removed, err := removeCursorHook(hooksJSONPath)
	if err != nil {
		return err
	}
	if removed {
		fmt.Println("[gotk] Cursor: hook removed.")
	} else {
		fmt.Println("[gotk] Cursor: hook was not installed")
	}
	return nil
}

func patchCursorHooks(path string) (bool, error) {
	var root map[string]any

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			root = map[string]any{"version": 1}
		} else {
			return false, err
		}
	} else {
		if jsonErr := json.Unmarshal(data, &root); jsonErr != nil || root == nil {
			root = map[string]any{"version": 1}
		}
	}

	// Idempotency check
	if hooks, ok := root["hooks"].(map[string]any); ok {
		if ptu, ok := hooks["preToolUse"].([]any); ok {
			for _, item := range ptu {
				m, ok := item.(map[string]any)
				if !ok {
					continue
				}
				if cmd, _ := m["command"].(string); cmd == cursorHookCommand {
					return false, nil
				}
			}
		}
	}

	// Insert entry
	hooks, _ := root["hooks"].(map[string]any)
	if hooks == nil {
		hooks = make(map[string]any)
	}
	ptu, _ := hooks["preToolUse"].([]any)
	ptu = append(ptu, map[string]any{
		"command": cursorHookCommand,
		"matcher": "Shell",
	})
	hooks["preToolUse"] = ptu
	root["hooks"] = hooks

	if _, ok := root["version"]; !ok {
		root["version"] = 1
	}

	out, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return false, err
	}
	return true, os.WriteFile(path, out, 0644)
}

func removeCursorHook(path string) (bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	var root map[string]any
	if err := json.Unmarshal(data, &root); err != nil {
		return false, err
	}
	hooks, ok := root["hooks"].(map[string]any)
	if !ok {
		return false, nil
	}
	ptu, ok := hooks["preToolUse"].([]any)
	if !ok {
		return false, nil
	}
	filtered := ptu[:0]
	removed := false
	for _, item := range ptu {
		m, ok := item.(map[string]any)
		if !ok {
			filtered = append(filtered, item)
			continue
		}
		if cmd, _ := m["command"].(string); strings.Contains(cmd, "gotk") {
			removed = true
		} else {
			filtered = append(filtered, item)
		}
	}
	if !removed {
		return false, nil
	}
	hooks["preToolUse"] = filtered
	root["hooks"] = hooks
	out, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return false, err
	}
	return true, os.WriteFile(path, out, 0644)
}

// ─── Gemini CLI ───────────────────────────────────────────────────────────────

const geminiHookScript = "#!/bin/bash\nexec gotk hook gemini\n"

func installGemini(opts InstallOptions) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	hookDir := filepath.Join(home, ".gemini", "hooks")
	hookPath := filepath.Join(hookDir, "gotk-hook-gemini.sh")
	settingsPath := filepath.Join(home, ".gemini", "settings.json")

	if opts.DryRun {
		fmt.Printf("[dry-run] would write Gemini hook script: %s\n", hookPath)
		fmt.Printf("[dry-run] would patch Gemini settings: %s\n", settingsPath)
		return nil
	}

	if err := os.MkdirAll(hookDir, 0755); err != nil {
		return fmt.Errorf("create gemini hooks dir: %w", err)
	}

	if err := os.WriteFile(hookPath, []byte(geminiHookScript), 0755); err != nil {
		return fmt.Errorf("write gemini hook: %w", err)
	}
	fmt.Printf("[gotk] Gemini: hook script written (%s)\n", hookPath)

	if err := patchGeminiSettings(settingsPath, hookPath); err != nil {
		return fmt.Errorf("patch Gemini settings: %w", err)
	}
	fmt.Printf("[gotk] Gemini: settings patched (%s)\n", settingsPath)
	fmt.Println("[gotk] Restart Gemini CLI to activate. Test with: git status")
	return nil
}

func uninstallGemini(opts InstallOptions) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	hookPath := filepath.Join(home, ".gemini", "hooks", "gotk-hook-gemini.sh")
	if opts.DryRun {
		fmt.Printf("[dry-run] would remove %s\n", hookPath)
		return nil
	}
	if err := os.Remove(hookPath); err != nil && !os.IsNotExist(err) {
		return err
	}
	fmt.Println("[gotk] Gemini: hook removed. Restart Gemini CLI.")
	return nil
}

func patchGeminiSettings(path, hookPath string) error {
	var cfg map[string]any

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			cfg = make(map[string]any)
		} else {
			return err
		}
	} else {
		if jsonErr := json.Unmarshal(data, &cfg); jsonErr != nil {
			cfg = make(map[string]any)
		}
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	hooks, _ := cfg["hooks"].(map[string]any)
	if hooks == nil {
		hooks = make(map[string]any)
	}

	// Idempotency: check for existing gotk entry in BeforeTool
	beforeTool, _ := hooks["BeforeTool"].([]any)
	for _, item := range beforeTool {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		hooksArr, _ := m["hooks"].([]any)
		for _, h := range hooksArr {
			hm, ok := h.(map[string]any)
			if !ok {
				continue
			}
			if cmd, _ := hm["command"].(string); strings.Contains(cmd, "gotk") {
				return nil // already present
			}
		}
	}

	entry := map[string]any{
		"matcher": "run_shell_command",
		"hooks": []any{
			map[string]any{"type": "command", "command": hookPath},
		},
	}
	beforeTool = append(beforeTool, entry)
	hooks["BeforeTool"] = beforeTool
	cfg["hooks"] = hooks

	out, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, out, 0644)
}

// ─── Windsurf ─────────────────────────────────────────────────────────────────

const windsurfRules = `# gotk (Go Token Killer) — Token-Optimized Commands

Always prefix shell commands with ` + "`gotk`" + ` for automatic token compression.
gotk intercepts output and reduces token usage 60-90% for tests, builds, and git.

## Usage
` + "```bash" + `
gotk go test ./...        # Go tests — failures only
gotk cargo build          # Rust build — errors only
gotk git status           # Compact status
gotk npm install          # Compact install output
` + "```" + `

gotk passes through any command it doesn't recognize unchanged.
`

func installWindsurf(opts InstallOptions) error {
	rulesPath := ".windsurfrules"

	existing, _ := os.ReadFile(rulesPath)
	if strings.Contains(string(existing), "gotk") {
		fmt.Println("[gotk] Windsurf: .windsurfrules already contains gotk rules")
		return nil
	}

	if opts.DryRun {
		fmt.Printf("[dry-run] would write %s\n", rulesPath)
		return nil
	}

	var content string
	if len(strings.TrimSpace(string(existing))) > 0 {
		content = strings.TrimSpace(string(existing)) + "\n\n" + windsurfRules
	} else {
		content = windsurfRules
	}

	if err := os.WriteFile(rulesPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("write .windsurfrules: %w", err)
	}
	fmt.Printf("[gotk] Windsurf: rules written to %s\n", rulesPath)
	fmt.Println("[gotk] Cascade will use gotk for token savings. Restart Windsurf.")
	return nil
}

// ─── Cline / Roo Code ─────────────────────────────────────────────────────────

const clineRules = `# gotk (Go Token Killer) — Token-Optimized Commands

Always prefix shell commands with ` + "`gotk`" + ` for automatic token compression.
gotk intercepts output and reduces token usage 60-90% for tests, builds, and git.

## Usage
` + "```bash" + `
gotk go test ./...        # Go tests — failures only
gotk cargo build          # Rust build — errors only
gotk git status           # Compact status
gotk npm install          # Compact install output
` + "```" + `

gotk passes through any command it doesn't recognize unchanged.
`

func installCline(opts InstallOptions) error {
	rulesPath := ".clinerules"

	existing, _ := os.ReadFile(rulesPath)
	if strings.Contains(string(existing), "gotk") {
		fmt.Println("[gotk] Cline: .clinerules already contains gotk rules")
		return nil
	}

	if opts.DryRun {
		fmt.Printf("[dry-run] would write %s\n", rulesPath)
		return nil
	}

	var content string
	if len(strings.TrimSpace(string(existing))) > 0 {
		content = strings.TrimSpace(string(existing)) + "\n\n" + clineRules
	} else {
		content = clineRules
	}

	if err := os.WriteFile(rulesPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("write .clinerules: %w", err)
	}
	fmt.Printf("[gotk] Cline: rules written to %s\n", rulesPath)
	fmt.Println("[gotk] Cline will use gotk for token savings. Test with: git status")
	return nil
}

// ─── Show status ───────────────────────────────────────────────────────────────

func showStatus(opts InstallOptions) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	fmt.Println("gotk installation status:")
	fmt.Println()

	// Claude Code
	settingsPath := filepath.Join(home, ".claude", "settings.json")
	data, err := os.ReadFile(settingsPath)
	claudeInstalled := false
	if err == nil {
		claudeInstalled = strings.Contains(string(data), "gotk")
	}
	checkmark(claudeInstalled, "Claude Code", settingsPath)

	// Cursor
	cursorPath := filepath.Join(home, ".cursor", "hooks.json")
	data, err = os.ReadFile(cursorPath)
	cursorInstalled := false
	if err == nil {
		cursorInstalled = strings.Contains(string(data), "gotk")
	}
	checkmark(cursorInstalled, "Cursor", cursorPath)

	// Gemini
	geminiHookPath := filepath.Join(home, ".gemini", "hooks", "gotk-hook-gemini.sh")
	_, geminiErr := os.Stat(geminiHookPath)
	checkmark(geminiErr == nil, "Gemini CLI", geminiHookPath)

	// Windsurf (project-local)
	_, wsErr := os.Stat(".windsurfrules")
	wsContent, _ := os.ReadFile(".windsurfrules")
	wsInstalled := wsErr == nil && strings.Contains(string(wsContent), "gotk")
	checkmark(wsInstalled, "Windsurf (local)", ".windsurfrules")

	// Cline (project-local)
	_, clErr := os.Stat(".clinerules")
	clContent, _ := os.ReadFile(".clinerules")
	clInstalled := clErr == nil && strings.Contains(string(clContent), "gotk")
	checkmark(clInstalled, "Cline (local)", ".clinerules")

	return nil
}

func checkmark(installed bool, name, path string) {
	mark := "✗"
	if installed {
		mark = "✓"
	}
	fmt.Printf("  %s  %-20s  %s\n", mark, name, path)
}
