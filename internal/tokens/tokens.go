// Package tokens provides LLM token counting using the cl100k_base BPE
// encoding (the same encoding used by GPT-4 and a close approximation for
// Claude Sonnet/Opus). It falls back to the bytes/4 heuristic if the BPE
// vocab file cannot be downloaded or loaded.
//
// The first call to Count (or PreWarm) downloads the BPE vocab file from
// OpenAI's CDN (~1.7 MB) and caches it at:
//
//	$TIKTOKEN_CACHE_DIR  (if set)
//	$HOME/.cache/tiktoken  (default)
package tokens

import (
	"fmt"
	"os"
	"sync"

	tiktoken "github.com/pkoukk/tiktoken-go"
)

// Encoding is the BPE encoding loaded for token counting.
// cl100k_base is used by GPT-4 and closely approximates Claude's tokenizer.
const Encoding = "cl100k_base"

var (
	enc     *tiktoken.Tiktoken
	initErr error
	once    sync.Once
)

func init() {
	// Kick off background download so it's ready before the command finishes.
	go func() { once.Do(load) }()
}

func load() {
	enc, initErr = tiktoken.GetEncoding(Encoding)
	if initErr != nil {
		fmt.Fprintf(os.Stderr,
			"[gotk] tiktoken: %v — usando estimativa bytes/4\n", initErr)
	}
}

// Count returns the number of BPE tokens in text.
// Blocks until the tokenizer is initialized (usually already done by the
// background goroutine started in init).
func Count(text string) int {
	once.Do(load)
	if initErr != nil || enc == nil {
		return heuristic(text)
	}
	return len(enc.Encode(text, nil, nil))
}

// IsExact reports whether the BPE tokenizer is available.
// Returns false when falling back to the bytes/4 heuristic.
func IsExact() bool {
	once.Do(load)
	return enc != nil && initErr == nil
}

// heuristic returns a rough token estimate: 1 token ≈ 4 bytes.
func heuristic(text string) int {
	if len(text) == 0 {
		return 0
	}
	return len(text) / 4
}
