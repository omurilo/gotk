---
title: "Environment Variables"
description: "All environment variables recognised by gotk."
order: 3
---

# Environment Variables

| Variable | Default | Description |
|---|---|---|
| `GOTK_LOG_DIR` | current directory | Directory where `gotk_raw.log` is written. |
| `GOTK_NO_FILTER` | — | Set to `1` to disable all filtering for the current invocation. |
| `TIKTOKEN_CACHE_DIR` | `~/.cache/tiktoken` | Cache directory for the cl100k_base BPE vocabulary file. |

## GOTK_LOG_DIR

Overrides where the raw log file is written. Useful in CI or when the working directory is not writable:

```bash
export GOTK_LOG_DIR=/tmp/gotk-logs
gotk cargo test
```

The file is always named `gotk_raw.log` inside this directory.

## GOTK_NO_FILTER

Disables the filter engine for a single invocation without changing `~/.config/gotk/config.json`:

```bash
GOTK_NO_FILTER=1 gotk go test ./...
```

Equivalent to setting `tee.mode = "always"` and removing all filter rules simultaneously.

## TIKTOKEN_CACHE_DIR

Redirects the BPE vocabulary file download. Useful in air-gapped environments where you pre-populate the cache:

```bash
# Pre-populate on a machine with internet access
TIKTOKEN_CACHE_DIR=/shared/cache gotk go test ./...

# Use on air-gapped machine
export TIKTOKEN_CACHE_DIR=/shared/cache
gotk cargo build
```

If the vocab file is unavailable, gotk falls back to the bytes ÷ 4 heuristic and logs a warning.
