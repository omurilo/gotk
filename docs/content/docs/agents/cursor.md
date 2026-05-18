---
title: "Cursor"
description: "Install the gotk hook for Cursor."
order: 2
---

# Cursor

## Install

```bash
gotk init --agent cursor
```

This patches `~/.cursor/hooks.json` with a PreToolUse hook for Bash commands.

Restart Cursor after installing.

## Uninstall

```bash
gotk init --agent cursor --uninstall
```

## Preview & troubleshoot

```bash
# Check installation status
gotk init --show

# Dry-run (no files written)
gotk init --agent cursor --dry-run

# Test the hook manually
echo '{"tool_name":"Bash","tool_input":{"command":"cargo test"}}' | gotk hook cursor
```
