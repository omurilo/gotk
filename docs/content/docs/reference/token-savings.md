---
title: "Token Savings"
description: "How gotk counts tokens, stores history, and reports savings with gotk stats."
order: 2
---

# Token Savings

## How token counting works

gotk uses the **cl100k_base BPE** encoding (the same tokenizer used by GPT-4, a close approximation for Claude Sonnet and Opus) via [`pkoukk/tiktoken-go`](https://github.com/pkoukk/tiktoken-go).

On first run, the BPE vocabulary file (~1.7 MB) is downloaded from OpenAI's CDN and cached at:

```
$TIKTOKEN_CACHE_DIR   # if set
~/.cache/tiktoken     # default
```

Subsequent runs use the cached file — no network access needed.

If the download fails, gotk falls back to a **bytes ÷ 4** heuristic and marks affected records with `"exact_tokens": false`.

### Background pre-warm

The tokenizer is initialised in a background goroutine at startup (`init()`). By the time a command finishes, the encoder is almost always ready. This adds zero latency to the main execution path.

## History file

Every run is appended as one JSON object to:

```
~/.local/share/gotk/history.jsonl
```

Example record:

```json
{
  "ts": "2026-05-18T14:32:01Z",
  "cmd": "go",
  "args": "test ./...",
  "raw_bytes": 48230,
  "filtered_bytes": 1840,
  "raw_tokens": 12058,
  "filtered_tokens": 460,
  "saved_tokens": 11598,
  "savings_pct": 96.2,
  "exec_ms": 4130,
  "exact_tokens": true
}
```

Because the file is plain JSONL, you can query it directly with `jq`:

```bash
# Total tokens saved
jq -s '[.[].saved_tokens] | add' ~/.local/share/gotk/history.jsonl

# Average savings by command
jq -s 'group_by(.cmd) | map({cmd: .[0].cmd, avg: (map(.savings_pct) | add / length)})' \
  ~/.local/share/gotk/history.jsonl
```

## gotk stats

```bash
gotk stats
```

Prints an aggregated summary: total tokens saved, average savings %, top commands by savings, and the last 10 runs.

```bash
# Clear all history
gotk stats --clear
```

## Bypassed runs

When the filter is bypassed (security command, loop guard, or `GOTK_NO_FILTER=1`), the run is still recorded with `"bypassed": true` and zero token savings. This ensures the history reflects actual usage without skewing averages.
