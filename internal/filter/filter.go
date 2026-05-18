package filter

import (
	"fmt"
	"regexp"
	"strings"
)

// filterState tracks the current line-processing mode.
type filterState int

const (
	stateNormal           filterState = iota
	stateErrorBlock                   // inside an error block — emit everything, buffer for grouping
	stateSuccessCollecting            // collecting consecutive success lines
)

// Filter processes command output line-by-line in real-time.
// It collapses noise while preserving every error-related line intact.
// Error blocks are passed through the Grouper before emission.
type Filter struct {
	bypassed bool
	profile  *ToolProfile
	grouper  Grouper

	state filterState

	// dedup window
	dedupLast  string
	dedupCount int

	// error block tracking
	errorBuf        []string // raw lines buffered during error block
	errorEmptyLines int

	// success collector
	successCount  int
	successSample string
}

// New constructs a Filter for the given command.
// bypass disables all filtering unconditionally.
func New(cmd string, bypass bool) *Filter {
	return &Filter{
		bypassed: bypass,
		profile:  DetectProfile(cmd),
	}
}

// Process transforms one raw line and returns 0, 1, or more output lines.
func (f *Filter) Process(line string) []string {
	if f.bypassed {
		return []string{line}
	}

	stripped := stripANSI(line)

	switch f.state {
	case stateErrorBlock:
		return f.inErrorBlock(stripped, line)
	case stateSuccessCollecting:
		return f.inSuccessCollecting(stripped, line)
	default:
		return f.inNormal(stripped, line)
	}
}

// Flush emits any pending buffered state at end-of-stream.
func (f *Filter) Flush() []string {
	var out []string

	// Close any open error block
	if f.state == stateErrorBlock && len(f.errorBuf) > 0 {
		out = append(out, f.grouper.Group(f.errorBuf)...)
		f.errorBuf = f.errorBuf[:0]
		f.state = stateNormal
	}

	if f.state == stateSuccessCollecting && f.successCount > 0 {
		out = append(out, f.buildSuccessSummary())
		f.successCount = 0
	}

	if flush := f.flushDedup(); flush != "" {
		out = append(out, flush)
	}
	return out
}

// ---------- state machine handlers ----------

func (f *Filter) inNormal(stripped, original string) []string {
	// Profile-level suppress (pure noise)
	if f.profile != nil && f.profile.SuppressOnly != nil && f.profile.SuppressOnly(stripped) {
		return nil
	}

	// Error block entry
	if f.isErrorStart(stripped) {
		var out []string
		if flush := f.flushDedup(); flush != "" {
			out = append(out, flush)
		}
		if f.successCount > 0 {
			out = append(out, f.buildSuccessSummary())
			f.successCount = 0
		}
		f.state = stateErrorBlock
		f.errorEmptyLines = 0
		f.errorBuf = append(f.errorBuf, original)
		return out
	}

	// Success line → collect
	if f.isSuccessLine(stripped) {
		if flush := f.flushDedup(); flush != "" {
			_ = flush // absorbed; success block supersedes dedup
		}
		if f.state != stateSuccessCollecting {
			f.state = stateSuccessCollecting
			f.successCount = 0
		}
		f.successCount++
		f.successSample = stripped
		return nil
	}

	// Flush success block when we exit it
	var out []string
	if f.successCount > 0 {
		out = append(out, f.buildSuccessSummary())
		f.successCount = 0
		f.state = stateNormal
	}

	// Empty lines don't participate in dedup
	if stripped == "" {
		if flush := f.flushDedup(); flush != "" {
			out = append(out, flush)
		}
		out = append(out, original)
		return out
	}

	// Deduplication
	normalized := normalizeNumbers(stripped)
	if normalized == f.dedupLast && f.dedupLast != "" {
		f.dedupCount++
		return out
	}

	if flush := f.flushDedup(); flush != "" {
		out = append(out, flush)
	}
	f.dedupLast = normalized
	f.dedupCount = 1
	out = append(out, original)
	return out
}

func (f *Filter) inErrorBlock(stripped, original string) []string {
	if stripped == "" {
		f.errorEmptyLines++
		f.errorBuf = append(f.errorBuf, original)

		// Two consecutive empty lines → exit error block, emit grouped
		if f.errorEmptyLines >= 2 {
			grouped := f.grouper.Group(f.errorBuf)
			f.errorBuf = f.errorBuf[:0]
			f.state = stateNormal
			f.errorEmptyLines = 0
			f.dedupLast = ""
			f.dedupCount = 0
			return grouped
		}
		return nil
	}

	f.errorEmptyLines = 0
	f.errorBuf = append(f.errorBuf, original)
	return nil
}

func (f *Filter) inSuccessCollecting(stripped, original string) []string {
	if f.isSuccessLine(stripped) {
		f.successCount++
		f.successSample = stripped
		return nil
	}

	// Exit success block
	out := []string{f.buildSuccessSummary()}
	f.successCount = 0
	f.state = stateNormal
	out = append(out, f.inNormal(stripped, original)...)
	return out
}

// ---------- classification helpers ----------

var globalErrorPatterns = []string{
	"error:", "ERROR:", "error[",
	"FAILED", "FAILURE",
	"Exception", "exception",
	"panic:", "PANIC",
	"fatal:", "FATAL",
	"Traceback (most recent call last)",
	"ASSERTION FAILED", "AssertionError",
	"undefined:", "cannot find module",
	"permission denied",
	"segmentation fault",
	"SIGSEGV", "SIGABRT",
}

func (f *Filter) isErrorStart(line string) bool {
	if f.profile != nil && f.profile.IsError != nil {
		return f.profile.IsError(line)
	}
	for _, p := range globalErrorPatterns {
		if strings.Contains(line, p) {
			return true
		}
	}
	// Go goroutine dump
	if strings.HasPrefix(line, "goroutine ") && strings.Contains(line, " [") {
		return true
	}
	return false
}

var globalSuccessPatterns = []string{
	"--- PASS:", "PASS\t", " passed", "all tests passed",
	"Build succeeded", "BUILD SUCCESS",
	"✓ ", "✔ ", "Done ",
}

func (f *Filter) isSuccessLine(line string) bool {
	if f.profile != nil && f.profile.IsSuccess != nil {
		return f.profile.IsSuccess(line)
	}
	for _, p := range globalSuccessPatterns {
		if strings.Contains(line, p) {
			return true
		}
	}
	return false
}

// ---------- dedup helpers ----------

func (f *Filter) flushDedup() string {
	if f.dedupCount > 1 {
		msg := fmt.Sprintf("[Repetido %d vezes]: %s", f.dedupCount, f.dedupLast)
		f.dedupLast = ""
		f.dedupCount = 0
		return msg
	}
	f.dedupLast = ""
	f.dedupCount = 0
	return ""
}

func (f *Filter) buildSuccessSummary() string {
	if f.successCount == 1 {
		return f.successSample
	}
	return fmt.Sprintf("[%d linhas OK omitidas — gotk]", f.successCount)
}

// ---------- utility ----------

var ansiEscape = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]|\x1b\][^\x07]*\x07`)
var numberRe = regexp.MustCompile(`\b\d+(\.\d+)?\b`)

func stripANSI(s string) string {
	return ansiEscape.ReplaceAllString(s, "")
}

// normalizeNumbers replaces digit sequences so "timeout after 30ms" and
// "timeout after 31ms" are treated as the same line for dedup purposes.
func normalizeNumbers(s string) string {
	return numberRe.ReplaceAllString(s, "N")
}
