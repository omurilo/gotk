---
title: "Windsurf & Cline"
description: "Install gotk for Windsurf Cascade and Cline/Roo Code."
order: 4
---

# Windsurf & Cline

Windsurf and Cline don't support global hook config files. Instead, gotk writes project-local rule files that instruct the agent to use `gotk`.

## Windsurf

```bash
# Run once per project
gotk init --agent windsurf
```

This creates or appends to `.windsurfrules` in the current directory:

```
When running shell commands, prefix them with "gotk " to reduce token usage.
Example: instead of "go test ./...", run "gotk go test ./...".
Do not wrap: commands with pipes (|), redirects (>), subshells ($(...)), or gotk itself.
```

To remove:
```bash
gotk init --agent windsurf --uninstall
```

## Cline / Roo Code

```bash
# Run once per project
gotk init --agent cline
```

This creates or appends to `.clinerules` in the current directory with the same instructions.

To remove:
```bash
gotk init --agent cline --uninstall
```

## Install all agents at once

```bash
gotk init --agent all
```

This installs gotk for all five supported agents in one command: Claude Code, Cursor, Gemini, Windsurf, and Cline.

```bash
# Preview without writing anything
gotk init --agent all --dry-run
```
