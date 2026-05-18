---
title: "Configuration"
description: "Configure gotk via ~/.config/gotk/config.json to tune tee mode, stack frame limits, and hook exclusions."
order: 3
---

# Configuration

`gotk` reads its global config from:

```
~/.config/gotk/config.json
```

The file is optional. If it doesn't exist, production-safe defaults are used. Missing fields also fall back to defaults, so you only need to specify what you want to override.

## Full example

```json
{
  "tee": {
    "mode": "failures"
  },
  "hooks": {
    "exclude_commands": ["curl", "wget", "ssh", "vim", "nano"]
  },
  "filter": {
    "max_stack_frames": 5,
    "max_grep_results": 30
  }
}
```

## tee

Controls when raw (unfiltered) output is written to `gotk_raw.log`.

| Value | Behaviour |
|---|---|
| `"failures"` | **(default)** Write raw log only when the command exits non-zero. The file is deleted on success. |
| `"always"` | Always append to `gotk_raw.log`, regardless of exit code. |
| `"never"` | Never write to disk. |

The log is written to the current working directory by default. Override with `GOTK_LOG_DIR`:

```bash
export GOTK_LOG_DIR=/tmp/gotk-logs
```

## hooks.exclude_commands

A list of command names the PreToolUse hook will **never** wrap with `gotk`. Useful for interactive commands or tools where filtering would cause problems.

```json
{
  "hooks": {
    "exclude_commands": ["curl", "wget", "ssh", "psql", "mysql", "vim"]
  }
}
```

The match is case-insensitive and checks only the first word of the command.

## filter.max_stack_frames

Maximum number of stack trace frames shown per error block. Frames beyond this limit are replaced with a single summary line:

```
[... 23 frames omitidos — gotk]
```

- **Default:** `5`
- **Disable:** `0` (unlimited)

## filter.max_grep_results

Maximum number of grep-style result lines shown before truncating. The truncation message shows the total count of suppressed lines.

- **Default:** `30`
- **Disable:** `0` (unlimited)

## Security bypass (built-in, non-configurable)

Commands containing these keywords are **never filtered**, regardless of config:

```
audit  vuln  vulnerability  security  snyk  trivy  cve
scan   owasp  grype  semgrep  gosec  bandit
```

## Loop protection (built-in, non-configurable)

If the exact same command runs **3 or more times within 5 minutes**, the filter is automatically disabled for that run. This prevents context loss when the AI retries a failing command.

```
[gotk] filtro desativado: mesmo comando executado 3+ vezes (loop guard)
```

## Disabling the filter entirely

```bash
GOTK_NO_FILTER=1 gotk go test ./...
```
