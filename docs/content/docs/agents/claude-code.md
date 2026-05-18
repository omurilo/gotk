---
title: "Claude Code"
description: "Install and configure the gotk PreToolUse hook for Claude Code."
order: 1
---

# Claude Code

## Install

```bash
gotk init
# equivalent to: gotk init --agent claude
```

This patches `~/.claude/settings.json` to register a `PreToolUse` hook for the `Bash` tool:

```json
{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Bash",
        "hooks": [
          { "type": "command", "command": "gotk hook claude" }
        ]
      }
    ]
  }
}
```

Restart Claude Code after installing.

## How it works

Before Claude Code runs any Bash command, the hook is called with the tool input as JSON on stdin:

```json
{
  "session_id": "...",
  "tool_name": "Bash",
  "tool_input": { "command": "go test ./..." }
}
```

`gotk hook claude` inspects the command and — if wrapping is safe — outputs modified JSON with `gotk` prepended:

```json
{ "command": "gotk go test ./..." }
```

Claude Code uses the modified command. The agent never sees the raw verbose output.

## Commands that are never wrapped

The hook skips wrapping in these cases:

- Command already starts with `gotk`
- Command contains shell metacharacters: `|`, `>`, `<`, `&&`, `||`, `;`, `` ` ``, `$(`
- Command contains security keywords: `audit`, `snyk`, `trivy`, `cve`, `scan`, etc.
- Command is in `hooks.exclude_commands` in your config

## Uninstall

```bash
gotk init --uninstall
```

## Troubleshoot

**Hook not triggering after restart:**
Verify the entry is present in `~/.claude/settings.json`:
```bash
cat ~/.claude/settings.json | grep -A4 gotk
```

**Preview what the hook would rewrite:**
```bash
echo '{"tool_name":"Bash","tool_input":{"command":"go test ./..."}}' | gotk hook claude
```

**Check install status:**
```bash
gotk init --show
```
