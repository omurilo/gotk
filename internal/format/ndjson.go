package format

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// ndJSONParser handles go test -json output (newline-delimited JSON events).
// All output is buffered; a clean summary is emitted on Flush.
//
// Event format:
//
//	{"Time":"...","Action":"run|pass|fail|skip|output","Package":"...","Test":"...","Elapsed":0.01,"Output":"..."}
type ndJSONParser struct {
	packages map[string]*pkgResult
	pkgOrder []string
}

type pkgResult struct {
	name    string
	passed  int
	failed  int
	skipped int
	elapsed float64
	// Only failing tests preserve their output lines.
	failOutput map[string][]string
	// pkg-level output (no Test field) for package-level failures
	pkgOutput  []string
	pkgFailed  bool
}

type goTestEvent struct {
	Time    time.Time `json:"Time"`
	Action  string    `json:"Action"`
	Package string    `json:"Package"`
	Test    string    `json:"Test"`
	Elapsed float64   `json:"Elapsed"`
	Output  string    `json:"Output"`
}

func newNDJSONParser() *ndJSONParser {
	return &ndJSONParser{packages: make(map[string]*pkgResult)}
}

func (p *ndJSONParser) Kind() Kind { return KindNDJSON }

func (p *ndJSONParser) pkg(name string) *pkgResult {
	if r, ok := p.packages[name]; ok {
		return r
	}
	r := &pkgResult{name: name, failOutput: make(map[string][]string)}
	p.packages[name] = r
	p.pkgOrder = append(p.pkgOrder, name)
	return r
}

func (p *ndJSONParser) Process(line string) []string {
	var ev goTestEvent
	if err := json.Unmarshal([]byte(line), &ev); err != nil {
		// Not a JSON event — pass through as-is (shouldn't happen in -json mode).
		return []string{line}
	}

	pkg := p.pkg(ev.Package)

	switch ev.Action {
	case "pass":
		if ev.Test != "" {
			pkg.passed++
		} else {
			pkg.elapsed = ev.Elapsed
		}

	case "fail":
		if ev.Test != "" {
			pkg.failed++
		} else {
			pkg.pkgFailed = true
			pkg.elapsed = ev.Elapsed
		}

	case "skip":
		if ev.Test != "" {
			pkg.skipped++
		}

	case "output":
		text := strings.TrimRight(ev.Output, "\n")
		if ev.Test != "" {
			// Buffer output only for tests that might fail.
			// We can't know yet, so buffer all and prune on "pass".
			pkg.failOutput[ev.Test] = append(pkg.failOutput[ev.Test], text)
		} else {
			pkg.pkgOutput = append(pkg.pkgOutput, text)
		}
	}

	// Prune passing test output to keep memory bounded.
	if ev.Action == "pass" && ev.Test != "" {
		delete(pkg.failOutput, ev.Test)
	}

	return nil // all output emitted on Flush
}

func (p *ndJSONParser) Flush() []string {
	var out []string

	for _, name := range p.pkgOrder {
		pkg := p.packages[name]
		total := pkg.passed + pkg.failed + pkg.skipped
		elapsed := ""
		if pkg.elapsed > 0 {
			elapsed = fmt.Sprintf(" (%.3fs)", pkg.elapsed)
		}

		if !pkg.pkgFailed && pkg.failed == 0 {
			out = append(out, fmt.Sprintf("ok   %s%s", name, elapsed))
			continue
		}

		out = append(out, fmt.Sprintf("FAIL %s%s — %d/%d falharam:", name, elapsed, pkg.failed, total))

		// Emit buffered output for failing tests
		for test, lines := range pkg.failOutput {
			out = append(out, fmt.Sprintf("  --- FAIL: %s", test))
			for _, l := range lines {
				if strings.TrimSpace(l) != "" {
					out = append(out, "    "+l)
				}
			}
		}

		// Package-level output (e.g., build errors)
		if pkg.pkgFailed {
			for _, l := range pkg.pkgOutput {
				if strings.TrimSpace(l) != "" {
					out = append(out, l)
				}
			}
		}
	}

	return out
}
