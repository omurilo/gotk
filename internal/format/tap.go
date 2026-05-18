package format

import (
	"fmt"
	"regexp"
	"strings"
)

// tapParser processes TAP (Test Anything Protocol) output.
// It collects pass/fail counts and emits failures with their diagnostic lines.
//
// Example TAP input:
//
//	TAP version 13
//	1..5
//	ok 1 - first test
//	not ok 2 - second test
//	  ---
//	  message: 'expected 1 to equal 2'
//	  ---
//	ok 3 - third test
type tapParser struct {
	total       int
	passed      int
	failed      int
	skipped     int
	failures    []tapFailure // buffered failures with diagnostics
	currentFail *tapFailure  // active failure being collected
	inDiag      bool         // inside a YAML diagnostic block (--- ... ---)
}

type tapFailure struct {
	desc string
	diag []string
}

var (
	tapOk        = regexp.MustCompile(`^ok\s+(\d+)\s*-?\s*(.*)`)
	tapNotOk     = regexp.MustCompile(`^not ok\s+(\d+)\s*-?\s*(.*)`)
	tapPlan      = regexp.MustCompile(`^1\.\.(\d+)`)
	tapDirective = regexp.MustCompile(`(?i)#\s*(skip|todo)`)
)

func newTAPParser() *tapParser { return &tapParser{} }

func (p *tapParser) Kind() Kind { return KindTAP }

func (p *tapParser) Process(line string) []string {
	trimmed := strings.TrimSpace(line)

	// Plan line: 1..N
	if m := tapPlan.FindStringSubmatch(trimmed); m != nil {
		_, _ = fmt.Sscanf(m[1], "%d", &p.total)
		return nil
	}

	// YAML diagnostic block markers
	if trimmed == "---" {
		p.inDiag = !p.inDiag
		return nil
	}
	if p.inDiag {
		if p.currentFail != nil {
			p.currentFail.diag = append(p.currentFail.diag, "  "+trimmed)
		}
		return nil
	}

	// ok N - description
	if m := tapOk.FindStringSubmatch(trimmed); m != nil {
		desc := strings.TrimSpace(m[2])
		if tapDirective.MatchString(desc) {
			p.skipped++
		} else {
			p.passed++
		}
		p.currentFail = nil
		return nil
	}

	// not ok N - description
	if m := tapNotOk.FindStringSubmatch(trimmed); m != nil {
		desc := strings.TrimSpace(m[2])
		p.failed++
		p.currentFail = &tapFailure{desc: desc}
		p.failures = append(p.failures, *p.currentFail)
		// Store pointer to the last element so diag lines attach to it
		p.currentFail = &p.failures[len(p.failures)-1]
		return nil
	}

	// Skip version header and other metadata
	return nil
}

func (p *tapParser) Flush() []string {
	var out []string

	if p.failed == 0 {
		summary := fmt.Sprintf("[TAP] %d/%d testes passaram", p.passed, p.total)
		if p.skipped > 0 {
			summary += fmt.Sprintf(", %d ignorados", p.skipped)
		}
		out = append(out, summary)
		return out
	}

	out = append(out, fmt.Sprintf(
		"[TAP] %d/%d passaram, %d FALHARAM:", p.passed, p.total, p.failed,
	))
	for _, f := range p.failures {
		out = append(out, "  not ok — "+f.desc)
		out = append(out, f.diag...)
	}
	return out
}
