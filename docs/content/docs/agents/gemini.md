---
title: "Gemini CLI"
description: "Install the gotk hook for Gemini CLI."
order: 3
---

# Gemini CLI

## Install

```bash
gotk init --agent gemini
```

This writes a hook script and patches `~/.gemini/settings.json`.

Restart Gemini CLI after installing.

## Uninstall

```bash
gotk init --agent gemini --uninstall
```

## Troubleshoot

```bash
gotk init --show
gotk init --agent gemini --dry-run

echo '{"tool_name":"Bash","tool_input":{"command":"pytest -v"}}' | gotk hook gemini
```
