#!/usr/bin/env bash
# gotk-rewrite.sh — Claude Code PreToolUse hook
#
# Intercepts Bash tool calls and rewrites them through `gotk` for token
# compression. Installed via: gotk init
#
# Protocol:
#   stdin  — JSON: {session_id, transcript_path, tool_name, tool_input:{command,...}}
#   stdout — Modified tool_input JSON (if rewrite applies), else empty
#   exit 0 — Allow (with or without modification)
#   exit 2 — Block (unused here)

set -euo pipefail

GOTK_BIN="${GOTK_BIN:-gotk}"

# Delegate all logic to `gotk hook` which handles JSON parsing robustly in Go.
# If gotk is not installed or fails, exit 0 to allow the command through unchanged.
if command -v "$GOTK_BIN" >/dev/null 2>&1; then
    "$GOTK_BIN" hook
else
    exit 0
fi
