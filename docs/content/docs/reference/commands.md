---
title: "CLI Commands"
description: "Complete reference for all gotk commands and flags."
order: 1
---

# CLI Commands

## gotk \<command\> [args...]

Proxy mode. Executes `command` with output compression active.

```bash
gotk go test ./...
gotk cargo build --release
gotk pytest -v tests/
gotk git log --oneline -50
```

The exit code of the child process is preserved exactly.

---

## gotk stats

Shows token savings history from `~/.local/share/gotk/history.jsonl`.

```bash
gotk stats
```

```
gotk token savings — 47 execuções registradas

Tokenizador:         cl100k_base BPE (tiktoken)
Total economizado:   128,430 tokens
Economia média:      82.3%

Top comandos:
  Comando                  Runs  Economia  Tokens economizados  Tempo médio
  cargo test                  8    99.2%               84,210       12.3s
  go test ./...              15    96.8%               22,140        4.1s
```

```bash
# Clear history
gotk stats --clear
```

---

## gotk rewrite \<command\>

Rewrites a command string and prints the result. Exits with a verdict code. Used internally by the PreToolUse hook.

```bash
gotk rewrite "go test ./..."
# stdout: gotk go test ./...
# exit:   0 (Allow)

gotk rewrite "vim main.go"
# stdout: vim main.go
# exit:   1 (Passthrough — no profile, run unchanged)

gotk rewrite "rm -rf /"
# exit:   2 (Deny)
```

| Exit code | Verdict | Meaning |
|---|---|---|
| `0` | Allow | Rewritten as `gotk <cmd>` |
| `1` | Passthrough | No registered profile, run as-is |
| `2` | Deny | Destructive or interactive command |

---

## gotk hook [--agent \<agent\>]

PreToolUse hook mode. Reads the agent's JSON tool input from stdin and outputs modified JSON if the command should be wrapped. Called automatically by the installed hook — you rarely need to run this directly.

```bash
# Default: claude
gotk hook

# Specific agent
gotk hook --agent cursor
gotk hook --agent gemini
```

**Test manually:**
```bash
echo '{"tool_name":"Bash","tool_input":{"command":"go test ./..."}}' \
  | gotk hook --agent claude
# stdout: {"command":"gotk go test ./..."}
```

---

## gotk init [--agent \<agent\>] [flags]

Installs or removes the gotk hook for the given agent.

```bash
gotk init                          # Claude Code (default)
gotk init --agent cursor
gotk init --agent gemini
gotk init --agent windsurf
gotk init --agent cline
gotk init --agent all              # All five agents
```

| Flag | Description |
|---|---|
| `--agent <name>` | Target agent (default: `claude`) |
| `--uninstall` | Remove the hook |
| `--show` | Print installation status without changing anything |
| `--dry-run` | Print what would be written without writing |

---

## gotk trust [file]

Computes the SHA-256 of a filter file and stores it in `~/.config/gotk/trusted.json` so gotk will apply it.

```bash
# Auto-detect: .gotk/filters.toml → .gotk/filters.json
gotk trust

# Explicit path
gotk trust .gotk/filters.toml
gotk trust ~/.config/gotk/filters.toml
```

Re-run after every change to the filter file.

---

## gotk --version

```bash
gotk --version
# gotk v0.1.0
```

---

## gotk --help

Prints full usage with all commands, flags, and examples.
