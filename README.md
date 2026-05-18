# gotk — Go Token Killer

**A high-performance CLI proxy that reduces AI agent token consumption by 60–90% on common development commands.**

`gotk` wraps shell commands, compresses their output in real-time, and passes only what matters to the AI context window. It is a Go port of [RTK (Rust Token Killer)](https://github.com/rtk-ai/rtk), with full compatibility with RTK's TOML filter format.

```
$ gotk go test ./...
[gotk: formato NDJSON detectado — saída comprimida]
ok   github.com/myorg/myapp/pkg/auth (0.312s)
ok   github.com/myorg/myapp/pkg/db (1.043s)
FAIL github.com/myorg/myapp/pkg/api (0.021s) — 1/12 falharam:
  --- FAIL: TestCreateUser
    expected status 201, got 400
    body: {"error":"email already exists"}
```

Instead of 4,800 tokens of verbose test output, the AI receives ~40 tokens with all the information it needs.

---

## Why gotk?

AI coding agents like Claude Code, Cursor, and Gemini spend most of their context window reading noise — compiler progress bars, passing test names, npm warnings, Docker cache lines. `gotk` filters that noise in real-time without losing a single error line.

| Command | Raw tokens | With gotk | Savings |
|---|---|---|---|
| `cargo test` (262 tests) | 4,823 | 11 | 99.8% |
| `go test ./...` (80 tests) | 1,240 | 38 | 97% |
| `git status` | 180 | 35 | 80% |
| `npm install` | 620 | 42 | 93% |
| `tsc --noEmit` (12 errors) | 890 | 210 | 76% |

---

## Installation

### go install
```sh
go install github.com/omurilo/gotk@latest
```

### Download binary

Download the latest release for your platform from the [Releases page](https://github.com/omurilo/gotk/releases).

| OS | Architecture | File |
|---|---|---|
| macOS | Apple Silicon | `gotk_darwin_arm64.tar.gz` |
| macOS | Intel | `gotk_darwin_amd64.tar.gz` |
| Linux | x86-64 | `gotk_linux_amd64.tar.gz` |
| Linux | ARM64 | `gotk_linux_arm64.tar.gz` |
| Windows | x86-64 | `gotk_windows_amd64.zip` |

Verify the checksum with the provided `checksums.txt`.

### Build from source
```sh
git clone https://github.com/omurilo/gotk
cd gotk
go build -o gotk .
sudo mv gotk /usr/local/bin/
```

---

## Quick start

### 1. Install the hook for your AI agent

```sh
# Claude Code (default)
gotk init

# Cursor
gotk init --agent cursor

# Gemini CLI
gotk init --agent gemini

# Install for all supported agents at once
gotk init --agent all
```

Restart your AI agent after installation. All Bash commands will now automatically be proxied through `gotk`.

### 2. Use manually (no hook required)

```sh
gotk go test ./...
gotk cargo build --release
gotk pytest -v
gotk git log --oneline -50
gotk npm install
gotk kubectl get pods -A
```

### 3. Check your token savings

```sh
gotk stats
```

```
gotk token savings — 47 execuções registradas

Tokenizador:         cl100k_base BPE (tiktoken)
Total economizado:   128,430 tokens
Economia média:      82.3%

Top comandos por tokens economizados:
  Comando                          Runs  Economia  Tokens economizados  Tempo médio
  ---------------------------------------------------------------------------
  cargo test                          8    99.2%               84,210       12.3s
  go test ./...                      15    96.8%               22,140        4.1s
  npm install                         6    93.1%                9,840        8.7s
```

---

## How it works

```
AI agent issues: go test ./...
       │
       ▼ PreToolUse hook rewrites to: gotk go test ./...
       │
       ▼ gotk runs: go test -json ./... (or plain)
       │
   ┌───┴──────────────────────────────────────────┐
   │           stdout + stderr (goroutines)       │
   └───┬──────────────────────────────────────────┘
       │
   Format detector (first line)
       │
   ┌───┴──────────────────────┬─────────────────────┐
   │  Structured format        │  Plain text          │
   │  TAP / NDJSON / JUnit XML │  TOML filter engine  │
   │  → parser → summary       │  → Filter → Truncator│
   └───────────────────────────┴─────────────────────┘
       │
   TeeLogger → gotk_raw.log (on failure)
       │
   Token tracker → ~/.local/share/gotk/history.jsonl
       │
   Compressed output → AI context window
```

**Four compression strategies:**

| Strategy | Example |
|---|---|
| **Filtering** | Suppress `=== RUN TestFoo`, npm warnings, Docker cache lines |
| **Deduplication** | `[Repetido 47 vezes]: timeout connecting to db` |
| **Success collapse** | `[312 linhas OK omitidas — gotk]` |
| **Error preservation** | Error blocks, stack traces, and failures are **never** truncated |

---

## Supported agents

| Agent | Hook location | Flag |
|---|---|---|
| [Claude Code](https://claude.ai/code) | `~/.claude/settings.json` | `--agent claude` (default) |
| [Cursor](https://cursor.sh) | `~/.cursor/hooks.json` | `--agent cursor` |
| [Gemini CLI](https://github.com/google-gemini/gemini-cli) | `~/.gemini/settings.json` | `--agent gemini` |
| [Windsurf](https://codeium.com/windsurf) | `.windsurfrules` (project) | `--agent windsurf` |
| [Cline / Roo Code](https://github.com/cline/cline) | `.clinerules` (project) | `--agent cline` |

```sh
# Install / uninstall / dry-run
gotk init --agent cursor
gotk init --agent cursor --uninstall
gotk init --agent all --dry-run

# Check installation status
gotk init --show
```

---

## Supported commands

gotk ships with built-in profiles for 80+ command families:

| Category | Commands |
|---|---|
| **Go** | `go test`, `go build`, `go vet`, `go mod` |
| **Rust** | `cargo test`, `cargo build`, `cargo clippy` |
| **Python** | `pytest`, `python`, `uv sync`, `poetry install` |
| **JavaScript** | `npm`, `yarn`, `pnpm`, `bun`, `jest`, `vitest` |
| **Git** | `git`, `gh`, `glab`, `jj` |
| **Docker** | `docker`, `docker compose` |
| **Kubernetes** | `kubectl`, `helm`, `k9s` |
| **Cloud** | `aws`, `gcloud`, `terraform`, `tofu` |
| **Build tools** | `make`, `gradle`, `mvn`, `nx`, `turbo`, `just` |
| **Linters** | `eslint`, `biome`, `oxlint`, `tsc`, `shellcheck` |
| **Ruby** | `bundle`, `rake`, `gem` |
| **.NET** | `dotnet` |
| **Swift** | `xcodebuild`, `swift build` |
| **Java** | `mvn`, `gradle`, `spring-boot` |
| **Elixir** | `mix compile`, `mix format` |
| **System** | `ls`, `find`, `grep`, `rg`, `ps`, `df`, `du` |
| **Other** | `ansible-playbook`, `rsync`, `ssh`, `mise`, `pre-commit` |

---

## Configuration

Global config at `~/.config/gotk/config.json`:

```json
{
  "tee": {
    "mode": "failures"
  },
  "hooks": {
    "exclude_commands": ["curl", "wget", "ssh"]
  },
  "filter": {
    "max_stack_frames": 5,
    "max_grep_results": 30
  }
}
```

| Key | Default | Description |
|---|---|---|
| `tee.mode` | `"failures"` | When to write raw log: `"always"`, `"failures"`, `"never"` |
| `hooks.exclude_commands` | `[]` | Commands the hook will never wrap |
| `filter.max_stack_frames` | `5` | Maximum stack trace frames per error block |
| `filter.max_grep_results` | `30` | Maximum grep output lines before truncating |

---

## Custom TOML filters (RTK-compatible)

gotk supports the same TOML filter format as RTK. Existing `.rtk/filters.toml` files work without modification.

**Load order** (first match per command wins):

1. `.gotk/filters.toml` — project-local (gotk)
2. `.rtk/filters.toml` — project-local (RTK compatibility)
3. `~/.config/gotk/filters.toml` — user-global

**Example filter** (`.gotk/filters.toml`):

```toml
[filters.my-server]
description = "Strip my-server startup noise"
match_command = "^my-server\\b"
strip_ansi = true
strip_lines_matching = [
  "^\\s*$",
  "^\\[INFO\\] Initializing",
  "^\\[DEBUG\\]",
]
keep_lines_matching = ["ERROR", "WARN", "started on port"]
max_lines = 50
on_empty = "my-server: started"
```

**Available filter stages:**

| Field | Type | Description |
|---|---|---|
| `match_command` | regex | Command pattern that activates this filter |
| `strip_ansi` | bool | Remove ANSI escape codes |
| `strip_lines_matching` | `[regex]` | Drop lines matching any pattern |
| `keep_lines_matching` | `[regex]` | Keep only lines matching any pattern |
| `replace` | `[{pattern, replacement}]` | Regex substitution on each line |
| `match_output` | `[{pattern, message}]` | Short-circuit: emit `message` if full output matches |
| `truncate_lines_at` | int | Maximum characters per line |
| `max_lines` | int | Maximum total output lines |
| `tail_lines` | int | Keep only the last N lines |
| `on_empty` | string | Emit this string if output becomes empty after filtering |

### Trusting project filters

For security, project-level filter files must be explicitly trusted before gotk applies them:

```sh
gotk trust
# or for a specific file:
gotk trust .gotk/filters.toml
```

The SHA-256 hash is stored in `~/.config/gotk/trusted.json`. If the file changes, run `gotk trust` again.

---

## Security bypass

Commands containing security-related keywords are **never filtered**. Full output is always passed through:

```
audit, vuln, vulnerability, security, snyk, trivy, cve, scan,
owasp, grype, semgrep, gosec, bandit
```

```sh
# These always produce 100% unfiltered output:
gotk npm audit
gotk trivy image myapp:latest
gotk snyk test
```

---

## Loop protection

If the exact same command runs **3 or more times within 5 minutes**, gotk disables filtering automatically. This prevents the AI from losing context when it is retrying a failed command.

```
[gotk] filtro desativado: mesmo comando executado 3+ vezes (loop guard)
```

---

## Raw log

Every execution writes raw (unfiltered) output to `gotk_raw.log` in the current directory. The default `"failures"` mode only keeps the file if the command exits non-zero; on success the temp file is deleted.

```sh
# Override log directory
export GOTK_LOG_DIR=/tmp/gotk-logs
gotk cargo test
```

---

## Environment variables

| Variable | Description |
|---|---|
| `GOTK_LOG_DIR` | Directory for `gotk_raw.log` (default: current directory) |
| `GOTK_NO_FILTER` | Set to `1` to disable all filtering |
| `TIKTOKEN_CACHE_DIR` | Cache directory for the BPE vocab file (default: `~/.cache/tiktoken`) |

---

## CLI reference

```
gotk <command> [args...]             Run command with output compression
gotk stats                           Show token savings history
gotk stats --clear                   Clear the history
gotk rewrite <command>               Rewrite command, exit 0/1/2
gotk hook [--agent <agent>]          PreToolUse hook (reads JSON from stdin)
gotk init [--agent <agent>]          Install hook (default: claude)
gotk init --show                     Show installation status
gotk init --uninstall                Remove hook
gotk init --dry-run                  Simulate installation without writing
gotk trust [file]                    Trust a TOML/JSON filter file
gotk --version                       Print version
gotk --help                          Print help
```

### gotk rewrite exit codes

| Code | Verdict | Meaning |
|---|---|---|
| `0` | Allow | Command rewritten as `gotk <cmd>`, output on stdout |
| `1` | Passthrough | No registered profile, run without wrapping |
| `2` | Deny | Destructive/interactive command, block execution |

---

## Token counting

`gotk stats` uses `cl100k_base` BPE (the same encoding as GPT-4, a close approximation for Claude) via [`pkoukk/tiktoken-go`](https://github.com/pkoukk/tiktoken-go). The vocabulary file (~1.7 MB) is downloaded once to `~/.cache/tiktoken` and cached permanently.

If the download fails, gotk falls back to a `bytes/4` heuristic and marks records as `exact_tokens: false` in `~/.local/share/gotk/history.jsonl`.

---

## Comparison with RTK

| Feature | RTK (Rust) | gotk (Go) |
|---|---|---|
| Proxy mode | ✅ | ✅ |
| TAP / NDJSON / JUnit XML parsers | ✅ | ✅ |
| TOML filter format | ✅ | ✅ (compatible) |
| 80+ command profiles | ✅ | ✅ |
| Error block preservation | ✅ | ✅ |
| Deduplication | ✅ | ✅ |
| Success collapsing | ✅ | ✅ |
| Grouper (errors by file) | ✅ | ✅ |
| Stack frame truncation | ✅ | ✅ |
| Security bypass | ✅ | ✅ |
| Loop guard (3x protection) | ✅ | ✅ |
| Tee logger (failure mode) | ✅ | ✅ |
| Multi-agent install | ✅ | ✅ |
| Token savings history | SQLite | JSONL |
| Token counting | N/A | cl100k_base BPE |
| MCP integration | ✅ | planned |

---

## Contributing

Issues and pull requests are welcome. Run the test suite with:

```sh
go test ./...
```

---

## License

MIT
