package filter

import (
	"fmt"
	"regexp"
	"strings"
)

// Truncator limits stack trace depth and grep result count in filtered output.
// It runs as a post-processor after the main Filter.
type Truncator struct {
	maxStack int
	maxGrep  int

	// stack trace state
	inStack       bool
	frameCount    int
	omittedFrames int

	// grep result state
	grepCount      int
	omittedGrep    int
	lastWasGrep    bool

	// ls -l stripping state
	lsMode bool
}

// NewTruncator creates a Truncator with the given limits (0 = unlimited).
func NewTruncator(maxStack, maxGrep int) *Truncator {
	return &Truncator{maxStack: maxStack, maxGrep: maxGrep}
}

// Process applies truncation rules to a single line.
func (t *Truncator) Process(line string) []string {
	stripped := stripANSI(line)

	// --- ls -l stripping ---
	if lsLongRe.MatchString(stripped) {
		t.lsMode = true
		// Keep only the filename (last field after timestamp)
		return []string{extractLsName(stripped)}
	}
	t.lsMode = false

	// --- Grep result capping ---
	isGrep := grepResultRe.MatchString(stripped)
	if isGrep {
		t.grepCount++
		t.lastWasGrep = true
		if t.maxGrep > 0 && t.grepCount > t.maxGrep {
			t.omittedGrep++
			return nil
		}
		return []string{line}
	}
	if t.lastWasGrep && !isGrep {
		t.lastWasGrep = false
		if t.omittedGrep > 0 {
			msg := fmt.Sprintf("[... %d resultados de grep omitidos — gotk]", t.omittedGrep)
			t.omittedGrep = 0
			t.grepCount = 0
			return []string{msg, line}
		}
		t.grepCount = 0
	}

	// --- Stack frame capping ---
	if isStackFrame(stripped) {
		if !t.inStack {
			t.inStack = true
			t.frameCount = 0
			t.omittedFrames = 0
		}
		t.frameCount++
		if t.maxStack > 0 && t.frameCount > t.maxStack {
			t.omittedFrames++
			return nil
		}
		return []string{line}
	}
	if t.inStack && !isStackFrame(stripped) {
		t.inStack = false
		if t.omittedFrames > 0 {
			msg := fmt.Sprintf("[... %d frames omitidos — gotk]", t.omittedFrames)
			t.omittedFrames = 0
			return []string{msg, line}
		}
	}

	return []string{line}
}

// Flush emits any pending "N omitted" messages.
func (t *Truncator) Flush() []string {
	var out []string
	if t.omittedGrep > 0 {
		out = append(out, fmt.Sprintf("[... %d resultados de grep omitidos — gotk]", t.omittedGrep))
	}
	if t.omittedFrames > 0 {
		out = append(out, fmt.Sprintf("[... %d frames omitidos — gotk]", t.omittedFrames))
	}
	return out
}

// ---------- pattern matchers ----------

var (
	// Go: \tgithub.com/..., \truntime/...
	// Java: \tat com.example...
	// Python: File "...", line N
	// JS/Node: "    at Object." etc.
	// Rust: "   N: " followed by path
	stackFrameRe = regexp.MustCompile(
		`(?:` +
			`^\s+at\s` + // JS/Java
			`|^\s+\tat\s` + // Java (double tab)
			`|^\t\S` + // Go (tab + non-space)
			`|^\s*File ".*", line \d+` + // Python
			`|^\s*\d+:\s+\S` + // Rust
			`|^goroutine\s+\d+\s+\[` + // Go goroutine header
			`|^\s+\.\.\.(\s+\d+\s+more)` + // Java truncated trace
			`)`,
	)

	// grep output: filename:lineno:content  or  filename:content
	grepResultRe = regexp.MustCompile(`^[^:]+:\d+:`)

	// ls -l: permission string like "-rw-r--r--"
	lsLongRe = regexp.MustCompile(`^[dlcbps-][rwx-]{9}\s`)
)

func isStackFrame(line string) bool {
	return stackFrameRe.MatchString(line)
}

// extractLsName returns just the filename from an `ls -l` line.
func extractLsName(line string) string {
	// ls -l format: "permissions links owner group size date name"
	// Split by whitespace, take everything from field 8 onward.
	fields := strings.Fields(line)
	if len(fields) >= 9 {
		return strings.Join(fields[8:], " ")
	}
	return line
}
