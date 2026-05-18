package filter

import (
	"fmt"
	"regexp"
	"strings"
)

// Grouper consolidates compiler/linter error lines that reference the same
// source file into a grouped block, reducing token usage when many errors
// come from the same file.
//
// Example input (TypeScript compiler):
//
//	src/main.ts(10,5): error TS2322: Type 'string' is not assignable to type 'number'.
//	src/main.ts(25,3): error TS2304: Cannot find name 'foo'.
//	src/util.ts(7,1): error TS2304: Cannot find name 'bar'.
//
// Example output:
//
//	src/main.ts — 2 erros:
//	  (10,5): error TS2322: Type 'string' is not assignable to type 'number'.
//	  (25,3): error TS2304: Cannot find name 'foo'.
//	src/util.ts — 1 erro:
//	  (7,1): error TS2304: Cannot find name 'bar'.
type Grouper struct{}

// fileRef captures the file portion of an error line.
// Supports:
//   - path/to/file.go:10:5: message  (Go, Rust, GCC)
//   - path/to/file.ts(10,5): message (TypeScript)
//   - path/to/file.py, line 10       (Python)
var fileRefRe = regexp.MustCompile(
	`^([^:\s][^:\n]*?)` + // file path (non-greedy, no leading space)
		`(?::(\d+)|\((\d+),\d+\))` + // :line  or  (line,col)
		`[:\s]`,
)

type errorLine struct {
	file    string
	rest    string // everything after the file:line prefix
	raw     string // original line
}

// Group takes a slice of error-block lines and returns them reorganised by
// source file. Lines without a file:line pattern are kept in their original
// position. If there is only one unique file referenced, the original slice is
// returned unchanged.
func (g *Grouper) Group(lines []string) []string {
	if len(lines) == 0 {
		return lines
	}

	type fileEntry struct {
		name   string
		errors []errorLine
	}

	grouped := make(map[string]*fileEntry)
	var fileOrder []string
	var ungrouped []string

	for _, raw := range lines {
		m := fileRefRe.FindStringSubmatch(raw)
		if m == nil {
			ungrouped = append(ungrouped, raw)
			continue
		}
		file := m[1]
		rest := strings.TrimPrefix(raw, file)
		if _, seen := grouped[file]; !seen {
			grouped[file] = &fileEntry{name: file}
			fileOrder = append(fileOrder, file)
		}
		grouped[file].errors = append(grouped[file].errors, errorLine{
			file: file,
			rest: rest,
			raw:  raw,
		})
	}

	// No grouping benefit if only one file or everything is ungrouped
	if len(grouped) <= 1 {
		return lines
	}

	var out []string
	out = append(out, ungrouped...)

	for _, name := range fileOrder {
		entry := grouped[name]
		if len(entry.errors) == 1 {
			out = append(out, entry.errors[0].raw)
			continue
		}
		label := "erros"
		if len(entry.errors) == 1 {
			label = "erro"
		}
		out = append(out, fmt.Sprintf("%s — %d %s:", name, len(entry.errors), label))
		for _, e := range entry.errors {
			out = append(out, "  "+strings.TrimLeft(e.rest, ":"))
		}
	}

	return out
}
