// Package tomlfilter implements RTK-compatible TOML-based command output filters.
//
// Filter files use the same schema as RTK's src/filters/*.toml so existing
// RTK project filters work without modification.
//
// Load order (first match wins per command):
//
//	.gotk/filters.toml       — project-local (gotk)
//	.rtk/filters.toml        — project-local (RTK compatibility)
//	~/.config/gotk/filters.toml — user-global
package tomlfilter

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/BurntSushi/toml"
)

// ─── schema ──────────────────────────────────────────────────────────────────

type tomlDoc struct {
	Filters map[string]FilterDef       `toml:"filters"`
	Tests   map[string][]InlineTest    `toml:"tests"`
}

// InlineTest is one [[tests.<filter-name>]] entry from an RTK filter file.
type InlineTest struct {
	Name     string `toml:"name"`
	Input    string `toml:"input"`
	Expected string `toml:"expected"`
}

// FilterDef mirrors RTK's TOML filter structure.
type FilterDef struct {
	Description        string            `toml:"description"`
	MatchCommand       string            `toml:"match_command"`
	StripAnsi          bool              `toml:"strip_ansi"`
	FilterStderr       bool              `toml:"filter_stderr"`
	StripLinesMatching []string          `toml:"strip_lines_matching"`
	KeepLinesMatching  []string          `toml:"keep_lines_matching"`
	Replace            []ReplaceRule     `toml:"replace"`
	MatchOutput        []MatchOutputRule `toml:"match_output"`
	TruncateLinesAt    int               `toml:"truncate_lines_at"`
	MaxLines           int               `toml:"max_lines"`
	TailLines          int               `toml:"tail_lines"`
	OnEmpty            string            `toml:"on_empty"`
}

// ReplaceRule is a regex substitution applied to each line.
type ReplaceRule struct {
	Pattern     string `toml:"pattern"`
	Replacement string `toml:"replacement"`
}

// MatchOutputRule short-circuits processing: if the full output matches
// Pattern (and does NOT match Unless), emit Message instead of all lines.
type MatchOutputRule struct {
	Pattern string `toml:"pattern"`
	Message string `toml:"message"`
	Unless  string `toml:"unless"`
}

// ─── compiled filter ─────────────────────────────────────────────────────────

type compiledFilter struct {
	name     string
	def      FilterDef
	matchCmd *regexp.Regexp

	stripLines  []*regexp.Regexp
	keepLines   []*regexp.Regexp
	replaceFrom []*regexp.Regexp
	matchOut    []compiledMatchOutput
}

type compiledMatchOutput struct {
	pattern *regexp.Regexp
	unless  *regexp.Regexp
	message string
}

func compile(name string, def FilterDef) (*compiledFilter, error) {
	cf := &compiledFilter{name: name, def: def}

	if def.MatchCommand != "" {
		re, err := regexp.Compile(def.MatchCommand)
		if err != nil {
			return nil, err
		}
		cf.matchCmd = re
	}

	for _, s := range def.StripLinesMatching {
		re, err := regexp.Compile(s)
		if err != nil {
			return nil, err
		}
		cf.stripLines = append(cf.stripLines, re)
	}

	for _, s := range def.KeepLinesMatching {
		re, err := regexp.Compile(s)
		if err != nil {
			return nil, err
		}
		cf.keepLines = append(cf.keepLines, re)
	}

	for _, r := range def.Replace {
		re, err := regexp.Compile(r.Pattern)
		if err != nil {
			return nil, err
		}
		cf.replaceFrom = append(cf.replaceFrom, re)
	}

	for _, mo := range def.MatchOutput {
		cmo := compiledMatchOutput{message: mo.Message}
		if mo.Pattern != "" {
			re, err := regexp.Compile(mo.Pattern)
			if err != nil {
				return nil, err
			}
			cmo.pattern = re
		}
		if mo.Unless != "" {
			re, err := regexp.Compile(mo.Unless)
			if err != nil {
				return nil, err
			}
			cmo.unless = re
		}
		cf.matchOut = append(cf.matchOut, cmo)
	}

	return cf, nil
}

// matches reports whether this filter handles the given command string.
func (cf *compiledFilter) matches(cmd string) bool {
	if cf.matchCmd == nil {
		return false
	}
	return cf.matchCmd.MatchString(cmd)
}

// ─── Registry ────────────────────────────────────────────────────────────────

// Registry holds compiled filters loaded from one or more TOML sources.
// First-match-wins semantics: when multiple registries are merged, earlier
// entries take priority.
type Registry struct {
	filters []*compiledFilter
}

// Empty returns a Registry with no filters.
func Empty() *Registry { return &Registry{} }

// Parse parses a TOML string and returns a Registry.
// Invalid regex patterns in a filter definition cause that filter to be
// silently skipped (same behaviour as RTK).
func Parse(content string) *Registry {
	reg, _ := parseDoc(content)
	return reg
}

// ParseWithTests parses a TOML string and returns the Registry plus inline
// tests keyed by filter name (used by the RTK inline test runner).
func ParseWithTests(content string) (*Registry, map[string][]InlineTest) {
	return parseDoc(content)
}

func parseDoc(content string) (*Registry, map[string][]InlineTest) {
	var doc tomlDoc
	if _, err := toml.Decode(content, &doc); err != nil {
		return Empty(), nil
	}
	reg := &Registry{}
	for name, def := range doc.Filters {
		cf, err := compile(name, def)
		if err != nil {
			continue
		}
		reg.filters = append(reg.filters, cf)
	}
	return reg, doc.Tests
}

// LoadFile loads filters from a TOML file.
// Returns an empty Registry if the file does not exist or cannot be parsed.
func LoadFile(path string) *Registry {
	data, err := os.ReadFile(path)
	if err != nil {
		return Empty()
	}
	return Parse(string(data))
}

// Merge combines registries into one; earlier registries have higher priority.
func Merge(regs ...*Registry) *Registry {
	merged := &Registry{}
	for _, r := range regs {
		merged.filters = append(merged.filters, r.filters...)
	}
	return merged
}

// LoadAll builds the standard multi-source registry with first-match-wins
// priority:
//
//  1. .gotk/filters.toml       — project-local (gotk)
//  2. .rtk/filters.toml        — project-local (RTK compatibility)
//  3. ~/.config/gotk/filters.toml — user-global
//  4. embedded built-in filters (59 RTK filters, read-only)
func LoadAll() *Registry {
	return Merge(
		LoadFile(".gotk/filters.toml"),
		LoadFile(".rtk/filters.toml"),
		LoadFile(globalFiltersPath()),
		loadBuiltins(),
	)
}

// Len returns the number of loaded filter definitions.
func (r *Registry) Len() int { return len(r.filters) }

// HasMatch reports whether any filter matches the command.
// fullCmd should be "executable args..." (full command string), mirroring how
// RTK matches match_command. If only the executable is provided, matching still
// works for simple regexes like "^make\\b".
func (r *Registry) HasMatch(fullCmd string) bool {
	return r.find(fullCmd) != nil
}

func (r *Registry) find(fullCmd string) *compiledFilter {
	for _, cf := range r.filters {
		if cf.matches(fullCmd) {
			return cf
		}
	}
	return nil
}

// ─── Apply ───────────────────────────────────────────────────────────────────

// Apply runs the matching filter's 8-stage pipeline against lines.
// Returns (output lines, true) when a filter matched, or (nil, false) when no
// filter matches cmd.
func (r *Registry) Apply(cmd string, lines []string) ([]string, bool) {
	cf := r.find(cmd)
	if cf == nil {
		return nil, false
	}
	return applyPipeline(cf, lines), true
}

func applyPipeline(cf *compiledFilter, lines []string) []string {
	def := cf.def

	// Stage 1 — strip ANSI
	if def.StripAnsi {
		for i, l := range lines {
			lines[i] = stripANSI(l)
		}
	}

	// Stage 2 — replace
	if len(cf.replaceFrom) > 0 {
		for i, l := range lines {
			for j, re := range cf.replaceFrom {
				l = re.ReplaceAllString(l, def.Replace[j].Replacement)
			}
			lines[i] = l
		}
	}

	// Stage 3 — match_output (short-circuit on full-output pattern)
	if len(cf.matchOut) > 0 {
		full := strings.Join(lines, "\n")
		for _, mo := range cf.matchOut {
			if mo.pattern != nil && !mo.pattern.MatchString(full) {
				continue
			}
			if mo.unless != nil && mo.unless.MatchString(full) {
				continue
			}
			if mo.message != "" {
				return []string{mo.message}
			}
			return nil
		}
	}

	// Stage 4 — strip_lines_matching
	if len(cf.stripLines) > 0 {
		out := lines[:0]
		for _, l := range lines {
			if !matchesAny(l, cf.stripLines) {
				out = append(out, l)
			}
		}
		lines = out
	}

	// Stage 5 — keep_lines_matching
	if len(cf.keepLines) > 0 {
		out := lines[:0]
		for _, l := range lines {
			if matchesAny(l, cf.keepLines) {
				out = append(out, l)
			}
		}
		lines = out
	}

	// Stage 6 — truncate_lines_at
	if def.TruncateLinesAt > 0 {
		for i, l := range lines {
			if len(l) > def.TruncateLinesAt {
				lines[i] = l[:def.TruncateLinesAt]
			}
		}
	}

	// Stage 7 — max_lines (before tail so tail applies to full filtered set)
	if def.MaxLines > 0 && len(lines) > def.MaxLines {
		lines = lines[:def.MaxLines]
	}

	// Stage 8 — tail_lines
	if def.TailLines > 0 && len(lines) > def.TailLines {
		lines = lines[len(lines)-def.TailLines:]
	}

	// Stage 9 — on_empty fallback
	if len(lines) == 0 && def.OnEmpty != "" {
		return []string{def.OnEmpty}
	}

	return lines
}

// ─── helpers ─────────────────────────────────────────────────────────────────

func matchesAny(line string, patterns []*regexp.Regexp) bool {
	for _, re := range patterns {
		if re.MatchString(line) {
			return true
		}
	}
	return false
}

var ansiRe = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

func stripANSI(s string) string {
	return ansiRe.ReplaceAllString(s, "")
}

func globalFiltersPath() string {
	if dir := os.Getenv("XDG_CONFIG_HOME"); dir != "" {
		return filepath.Join(dir, "gotk", "filters.toml")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "gotk", "filters.toml")
}
