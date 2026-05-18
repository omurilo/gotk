---
title: "Custom Filters"
description: "Write project-specific TOML filters to compress output from any tool, using the same format as RTK."
order: 4
---

# Custom Filters

gotk supports project-level and user-level filter rules written in TOML. The format is **fully compatible with RTK's filter files** — existing `.rtk/filters.toml` files work without modification.

## File locations

gotk searches for filter files in this order (first match per command wins):

| File | Scope |
|---|---|
| `.gotk/filters.toml` | Project (gotk) |
| `.rtk/filters.toml` | Project (RTK compatibility) |
| `~/.config/gotk/filters.toml` | User-global |

## Minimal example

```toml
[filters.my-server]
description = "Strip my-server startup noise"
match_command = "^my-server\\b"
strip_ansi = true
strip_lines_matching = [
  "^\\s*$",
  "^\\[DEBUG\\]",
  "^\\[INFO\\] Initializing",
]
on_empty = "my-server: no errors"
```

Save to `.gotk/filters.toml`, then trust the file:

```bash
gotk trust
```

From then on, `gotk my-server` uses your custom filter.

## Trusting filter files

Project filter files must be explicitly trusted before gotk applies them. This prevents a malicious `.gotk/filters.toml` committed by someone else from running automatically.

```bash
# Trust .gotk/filters.toml (default)
gotk trust

# Trust a specific file
gotk trust .gotk/filters.toml

# Trust a global user filter
gotk trust ~/.config/gotk/filters.toml
```

The SHA-256 hash of the file is stored in `~/.config/gotk/trusted.json`. If the file changes, run `gotk trust` again.

## All filter stages

Stages run in the order listed. All fields are optional.

### strip_ansi

Removes ANSI escape codes (colors, cursor movement) before any other stage.

```toml
strip_ansi = true
```

### replace

Regex substitution applied to each line. Runs before line-level filters.

```toml
[[filters.my-tool.replace]]
pattern = "\\d{4}-\\d{2}-\\d{2}T\\d{2}:\\d{2}:\\d{2}Z"
replacement = "<timestamp>"

[[filters.my-tool.replace]]
pattern = "session=[a-f0-9]+"
replacement = "session=<redacted>"
```

### match_output

Short-circuit rule: if the **entire output** matches `pattern` (and does not match `unless`), emit `message` instead of all lines. Useful for collapsing fully-successful runs.

```toml
[[filters.my-tool.match_output]]
pattern = "^All checks passed"
message = "my-tool: ok ✓"

[[filters.my-tool.match_output]]
pattern = "0 errors"
unless = "warning"
message = "my-tool: 0 errors"
```

### strip_lines_matching

Drop any line matching at least one of the listed regexes.

```toml
strip_lines_matching = [
  "^\\s*$",
  "^Downloading ",
  "^Progress: ",
  "\\[INFO\\].*heartbeat",
]
```

### keep_lines_matching

Keep **only** lines matching at least one of the listed regexes. Applied after `strip_lines_matching`.

```toml
keep_lines_matching = [
  "ERROR",
  "WARN",
  "FAIL",
  "started on port",
]
```

### truncate_lines_at

Truncate individual lines that exceed N characters.

```toml
truncate_lines_at = 200
```

### max_lines

Keep only the first N lines of the final output.

```toml
max_lines = 50
```

### tail_lines

Keep only the **last** N lines. Applied after `max_lines`.

```toml
tail_lines = 20
```

### on_empty

If the output is empty after all stages, emit this string instead of silence.

```toml
on_empty = "my-tool: completed with no output"
```

## Complete example

```toml
[filters.spring-boot]
description = "Spring Boot startup — keep warnings, errors, and the ready line"
match_command = "^(java|mvn spring-boot:run|./mvnw spring-boot:run)\\b"
strip_ansi = true

strip_lines_matching = [
  "^\\s*$",
  "^  \\.",                            # banner decorations
  "^\\/\\/ ",                          # banner decorations
  "^\\s+:: Spring Boot ::",           # banner
  "^\\d{4}-\\d{2}-\\d{2}T.*INFO.*\\[main\\].*Tomcat initialized",
  "^\\d{4}-\\d{2}-\\d{2}T.*INFO.*\\[main\\].*Starting ",
]

keep_lines_matching = [
  "WARN",
  "ERROR",
  "FATAL",
  "Started .* in .* seconds",
  "APPLICATION FAILED TO START",
]

on_empty = "spring-boot: started (no warnings)"
```

## Built-in filter files

gotk ships with over 60 built-in TOML filters in `internal/tomlfilter/filters/`. These cover tools like `gradle`, `make`, `terraform`, `docker`, `uv sync`, `poetry`, and many more. Built-in filters are always active — no `gotk trust` required.

To see which filter applies to a command:

```bash
gotk rewrite "gradle build"
```
