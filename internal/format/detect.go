package format

import (
	"encoding/json"
	"strings"
)

// Kind identifies the output format of a command.
type Kind int

const (
	KindPlain  Kind = iota // unstructured text — run through Filter
	KindTAP                // Test Anything Protocol
	KindNDJSON             // Newline-delimited JSON (go test -json)
	KindJUnit              // JUnit XML (buffered, parsed at end)
)

func (k Kind) String() string {
	switch k {
	case KindTAP:
		return "TAP"
	case KindNDJSON:
		return "NDJSON"
	case KindJUnit:
		return "JUnit XML"
	default:
		return "plain"
	}
}

// Parser transforms raw output lines into compressed, human-readable lines.
// Structured parsers (TAP, NDJSON, JUnit) bypass the Filter entirely.
type Parser interface {
	Kind() Kind
	// Process handles one raw line. Returns 0-N output lines.
	Process(line string) []string
	// Flush emits any buffered state at end-of-stream.
	Flush() []string
}

// Detect inspects the first non-empty line and returns the most likely Kind.
func Detect(firstLine string) Kind {
	trimmed := strings.TrimSpace(firstLine)
	if trimmed == "" {
		return KindPlain
	}
	// JUnit XML
	if strings.HasPrefix(trimmed, "<?xml") || strings.HasPrefix(trimmed, "<testsuites") ||
		strings.HasPrefix(trimmed, "<testsuite") {
		return KindJUnit
	}
	// TAP
	if strings.HasPrefix(trimmed, "TAP version") {
		return KindTAP
	}
	// NDJSON: line must be a JSON object with an "Action" key (go test -json)
	if strings.HasPrefix(trimmed, "{") {
		var obj map[string]any
		if json.Unmarshal([]byte(trimmed), &obj) == nil {
			if _, ok := obj["Action"]; ok {
				return KindNDJSON
			}
		}
	}
	return KindPlain
}

// NewParser creates the appropriate Parser for kind.
func NewParser(k Kind) Parser {
	switch k {
	case KindTAP:
		return newTAPParser()
	case KindNDJSON:
		return newNDJSONParser()
	case KindJUnit:
		return newJUnitParser()
	default:
		return &plainParser{}
	}
}

// plainParser is a no-op — plain text goes straight through.
type plainParser struct{}

func (p *plainParser) Kind() Kind                   { return KindPlain }
func (p *plainParser) Process(line string) []string { return []string{line} }
func (p *plainParser) Flush() []string              { return nil }
