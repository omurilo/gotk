package format

import (
	"encoding/xml"
	"fmt"
	"strings"
)

// junitParser accumulates JUnit XML output (which cannot be parsed line-by-line)
// and emits a compressed summary on Flush.
//
// Supports both <testsuites> (multiple suites) and bare <testsuite> root elements.
type junitParser struct {
	buf strings.Builder
}

// XML structures — only the fields gotk needs.

type xmlTestSuites struct {
	Suites []xmlTestSuite `xml:"testsuite"`
}

type xmlTestSuite struct {
	Name     string        `xml:"name,attr"`
	Tests    int           `xml:"tests,attr"`
	Failures int           `xml:"failures,attr"`
	Errors   int           `xml:"errors,attr"`
	Time     float64       `xml:"time,attr"`
	Cases    []xmlTestCase `xml:"testcase"`
}

type xmlTestCase struct {
	Name      string      `xml:"name,attr"`
	ClassName string      `xml:"classname,attr"`
	Time      float64     `xml:"time,attr"`
	Failure   *xmlMessage `xml:"failure"`
	Error     *xmlMessage `xml:"error"`
	Skipped   *struct{}   `xml:"skipped"`
}

type xmlMessage struct {
	Message string `xml:"message,attr"`
	Body    string `xml:",chardata"`
}

func newJUnitParser() *junitParser { return &junitParser{} }

func (p *junitParser) Kind() Kind { return KindJUnit }

// Process buffers every line of XML — parsing happens in Flush.
func (p *junitParser) Process(line string) []string {
	p.buf.WriteString(line)
	p.buf.WriteByte('\n')
	return nil
}

func (p *junitParser) Flush() []string {
	raw := p.buf.String()
	if strings.TrimSpace(raw) == "" {
		return nil
	}

	suites, err := parseJUnit(raw)
	if err != nil {
		// Parse failure: fall back to raw output
		lines := strings.Split(raw, "\n")
		return lines
	}

	return formatSuites(suites)
}

func parseJUnit(raw string) ([]xmlTestSuite, error) {
	// Try <testsuites> wrapper first.
	var ts xmlTestSuites
	if err := xml.Unmarshal([]byte(raw), &ts); err == nil && len(ts.Suites) > 0 {
		return ts.Suites, nil
	}
	// Try bare <testsuite>.
	var suite xmlTestSuite
	if err := xml.Unmarshal([]byte(raw), &suite); err != nil {
		return nil, err
	}
	return []xmlTestSuite{suite}, nil
}

func formatSuites(suites []xmlTestSuite) []string {
	var out []string

	for _, s := range suites {
		total := s.Tests
		bad := s.Failures + s.Errors
		passed := total - bad

		timeStr := ""
		if s.Time > 0 {
			timeStr = fmt.Sprintf(" (%.3fs)", s.Time)
		}

		if bad == 0 {
			out = append(out, fmt.Sprintf("[JUnit] %s: %d/%d passaram%s", s.Name, passed, total, timeStr))
			continue
		}

		out = append(out, fmt.Sprintf("[JUnit] %s: %d/%d passaram%s, %d FALHARAM:", s.Name, passed, total, timeStr, bad))

		for _, tc := range s.Cases {
			var msg *xmlMessage
			if tc.Failure != nil {
				msg = tc.Failure
			} else if tc.Error != nil {
				msg = tc.Error
			}
			if msg == nil {
				continue
			}

			label := tc.Name
			if tc.ClassName != "" && tc.ClassName != tc.Name {
				label = tc.ClassName + "." + tc.Name
			}
			out = append(out, fmt.Sprintf("  FAIL: %s", label))

			if msg.Message != "" {
				out = append(out, "    "+msg.Message)
			}
			// Show first 3 lines of the failure body (stack trace truncation)
			body := strings.TrimSpace(msg.Body)
			if body != "" {
				lines := strings.SplitN(body, "\n", 4)
				for i, l := range lines {
					if i == 3 {
						out = append(out, "    [... mais linhas omitidas — gotk]")
						break
					}
					out = append(out, "    "+strings.TrimSpace(l))
				}
			}
		}
	}

	return out
}
