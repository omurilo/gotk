---
title: "Getting Started"
description: "Install gotk and connect it to your AI agent in under 2 minutes."
order: 1
---

# Getting Started

`gotk` wraps shell commands and compresses their output in real-time so your AI agent (Claude Code, Cursor, Gemini, etc.) receives only what matters — errors, failures, and meaningful results — instead of thousands of tokens of noise.

## Prerequisites

- Go 1.22+ **or** a pre-built binary (see [Installation](/docs/installation))
- One of the [supported AI agents](/docs/agents/claude-code)

## 1 — Install

```bash
go install github.com/omurilo/gotk@latest
```

Confirm the install:

```bash
gotk --version
```

## 2 — Connect to your agent

Run `init` once to register the PreToolUse hook inside your agent's settings file:

```bash
# Claude Code (default)
gotk init

# Cursor
gotk init --agent cursor

# Gemini CLI
gotk init --agent gemini
```

Restart your agent. From this point on every Bash command the agent runs is automatically proxied through `gotk`.

## 3 — Use manually (no hook required)

You can always call `gotk` directly without any hook:

```bash
gotk go test ./...
gotk cargo build --release
gotk pytest -v
gotk npm install
```

## 4 — Check savings

After a few commands, view how many tokens were saved:

```bash
gotk stats
```

```
gotk token savings — 23 execuções registradas

Total economizado:   41,820 tokens
Economia média:      84.7%
```

## What gets filtered

| Output type | What gotk does |
|---|---|
| Passing tests (`PASS`, `ok`) | Collapsed to a one-line count |
| Repeated identical lines | `[Repetido 47 vezes]: <line>` |
| Error blocks | **Never touched** — every line preserved |
| Stack traces | First 5 frames kept, rest counted |
| npm/cargo progress noise | Suppressed |
| Security audit output | **Never filtered** (bypass active) |

## Next steps

- [Installation options](/docs/installation) — Homebrew, binary download, build from source
- [Agents](/docs/agents/claude-code) — per-agent setup guides
- [Configuration](/docs/configuration) — tune tee mode, stack frames, excluded commands
- [Custom filters](/docs/custom-filters) — write project-specific TOML rules
