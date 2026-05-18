---
title: "Agents"
description: "Setup guides for each supported AI coding agent."
order: 5
nav:
  - claude-code
  - cursor
  - gemini
  - others
---

# Agents

gotk integrates with AI coding agents through their PreToolUse hook system. When installed, the agent rewrites every Bash command it issues through `gotk` before execution — no manual wrapping required.

| Agent | Hook mechanism | Command |
|---|---|---|
| Claude Code | `~/.claude/settings.json` | `gotk init` |
| Cursor | `~/.cursor/hooks.json` | `gotk init --agent cursor` |
| Gemini CLI | `~/.gemini/settings.json` | `gotk init --agent gemini` |
| Windsurf | `.windsurfrules` (project) | `gotk init --agent windsurf` |
| Cline / Roo Code | `.clinerules` (project) | `gotk init --agent cline` |

Install all at once:

```bash
gotk init --agent all
```
