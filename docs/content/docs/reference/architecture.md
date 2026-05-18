---
title: "Architecture"
description: "How gotk's pipeline works internally — for contributors and advanced users."
order: 4
---

# Architecture

## Pipeline overview

```
AI agent: "go test ./..."
         │
         ▼  PreToolUse hook rewrites to: "gotk go test ./..."
         │
    gotk proxy
         │
   ┌─────┴──────────────────────────────────────────────────────┐
   │           os/exec  +  StdoutPipe / StderrPipe              │
   │           two goroutines → lineEvent channel (buf 512)      │
   └─────┬──────────────────────────────────────────────────────┘
         │
   ┌─────▼──────────────────────────────┐
   │   TOML filter match?               │  (tomlfilter.LoadAll)
   │   Yes → buffer all lines           │
   │   No  → stream                     │
   └─────┬──────────────────────────────┘
         │                     │
   ┌─────▼──────┐        ┌─────▼───────────────────────────────┐
   │ Format      │        │ Plain text pipeline                  │
   │ Detector    │        │                                      │
   │ (first line)│        │  1. Hot-reloadable project rules     │
   │             │        │     (regex suppress/collapse)        │
   │ TAP │ NDJSON│        │                                      │
   │ JUnit XML   │        │  2. Filter engine                    │
   │             │        │     • Profile-level suppress         │
   │ → Parser    │        │     • Error block detection + buffer │
   │   emit      │        │     • Grouper (errors by file)       │
   │   summary   │        │     • Success collapsing             │
   └─────┬───────┘        │     • Deduplication                  │
         │                │                                      │
         │                │  3. Truncator                        │
         │                │     • Stack frame capping            │
         │                │     • Grep result capping            │
         │                │     • ls -l stripping                │
         │                └─────┬───────────────────────────────┘
         │                      │
         └──────────┬───────────┘
                    │
             ┌──────▼──────┐
             │  TeeLogger   │ ──→ gotk_raw.log (on failure)
             └──────┬───────┘
                    │
             ┌──────▼──────┐
             │  stdout /   │
             │  stderr     │
             └──────┬───────┘
                    │
             ┌──────▼────────────────────┐
             │  Token tracker            │
             │  cl100k_base BPE count    │
             │  → history.jsonl          │
             └───────────────────────────┘
```

## Package layout

| Package | Responsibility |
|---|---|
| `main` | CLI dispatch, flag parsing, `stats` / `trust` rendering |
| `internal/proxy` | Child process execution, pipeline orchestration, hook mode |
| `internal/filter` | Filter state machine, Grouper, Truncator, ToolProfiles |
| `internal/format` | Format detection (TAP/NDJSON/JUnit), structured parsers |
| `internal/tomlfilter` | RTK-compatible TOML filter engine, built-in filter files |
| `internal/registry` | Command registry (80+ rules), `Rewrite()` verdict |
| `internal/config` | JSON config loading, project filter trust verification |
| `internal/logger` | TeeLogger (failures/always/never mode) |
| `internal/state` | Loop detection, per-command run counter |
| `internal/tokens` | cl100k_base BPE via tiktoken-go, bytes/4 fallback |
| `internal/tracker` | JSONL history writer/reader, `GetSummary()` |

## Filter state machine

The `Filter` struct processes one line at a time through three states:

```
StateNormal
  │  isErrorStart(line)  ──────────────────────→ StateErrorBlock
  │  isSuccessLine(line) ──────────────────────→ StateSuccessCollecting
  │  otherwise: dedup window, suppress rules
  │
StateErrorBlock
  │  buffers all lines for Grouper
  │  2 consecutive empty lines ────────────────→ StateNormal (emit grouped)
  │
StateSuccessCollecting
  │  accumulates consecutive success lines
  │  non-success line ──────────────────────────→ StateNormal
  │  emits: "[N linhas OK omitidas — gotk]"
```

## TOML filter pipeline

When a TOML filter matches the command, all output is buffered until the child exits, then processed through 9 stages in order:

1. `strip_ansi`
2. `replace` (regex substitution per line)
3. `match_output` (short-circuit on full-output pattern)
4. `strip_lines_matching`
5. `keep_lines_matching`
6. `truncate_lines_at`
7. `max_lines`
8. `tail_lines`
9. `on_empty`

## Hook protocol

The PreToolUse hook (`gotk hook --agent <name>`) reads JSON from stdin and writes modified JSON to stdout:

```
stdin:   { "tool_name": "Bash", "tool_input": { "command": "go test ./..." } }
stdout:  { "command": "gotk go test ./..." }
exit 0
```

The hook silently exits 0 (no output) when:
- The command contains shell metacharacters (`|`, `>`, `&&`, etc.)
- The command already starts with `gotk`
- The command contains security keywords
- `gotk rewrite` returns verdict `Passthrough` or `Deny`

## Exit code preservation

`gotk` always exits with the child process's exit code:

```go
var exitErr *exec.ExitError
if errors.As(waitErr, &exitErr) {
    return exitErr.ExitCode(), nil
}
```

This ensures CI/CD pipelines, pre-commit hooks, and `&&` chains behave correctly.
