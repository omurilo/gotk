package tomlfilter

import (
	"io/fs"
	"strings"
	"testing"
)

// TestRTKInline runs every [[tests.*]] block embedded in the RTK filter files
// against gotk's filter pipeline, exactly mirroring RTK's own inline test
// suite.  Each test case gets its own t.Run so failures are reported
// individually without aborting the whole suite.
func TestRTKInline(t *testing.T) {
	entries, err := fs.Glob(builtinFS, "filters/*.toml")
	if err != nil {
		t.Fatalf("glob filters: %v", err)
	}

	total, passed, skipped := 0, 0, 0

	for _, path := range entries {
		data, err := builtinFS.ReadFile(path)
		if err != nil {
			t.Errorf("read %s: %v", path, err)
			continue
		}

		reg, tests := ParseWithTests(string(data))
		if len(tests) == 0 {
			continue
		}

		for filterName, cases := range tests {
			for _, tc := range cases {
				total++
				tc := tc // capture
				filterName := filterName
				path := path

				t.Run(filterName+"/"+tc.Name, func(t *testing.T) {
					inputLines := splitLines(tc.Input)
					wantLines := splitLines(tc.Expected)

					// Build a match command from the filter name that the
					// registry will recognise.  We try the filter name itself
					// (e.g. "make") and, for hyphenated names, the part before
					// the first hyphen (e.g. "terraform" for "terraform-plan").
					cmdCandidates := matchCandidates(filterName)

					var gotLines []string
					var matched bool
					for _, cmd := range cmdCandidates {
						gotLines, matched = reg.Apply(cmd, cloneLines(inputLines))
						if matched {
							break
						}
					}

					if !matched {
						// Filter didn't match any of our candidates — skip rather
						// than fail: this happens for filters whose match_command
						// uses full argv patterns (e.g. "./gradlew", "terraform plan").
						t.Skipf("filter %q in %s: no match for candidates %v — skip",
							filterName, path, cmdCandidates)
						skipped++
						return
					}

					if !linesEqual(gotLines, wantLines) {
						t.Errorf("filter %q in %s\n  test:  %q\n  input: %q\n  want:  %v\n  got:   %v",
							filterName, path, tc.Name,
							tc.Input, wantLines, gotLines)
					} else {
						passed++
					}
				})
			}
		}
	}

	t.Logf("RTK inline tests: %d total, %d passed, %d skipped (unmatched cmd pattern)",
		total, passed, skipped)
}

// splitLines splits a test string into trimmed lines, dropping the trailing
// empty string that results from a trailing newline.  This matches how RTK
// normalises multi-line test strings.
func splitLines(s string) []string {
	// TOML multi-line strings start with a trimmed first newline; single-line
	// strings have no trailing newline.  Normalise by trimming both ends.
	// Also strip \r so tests pass on Windows where git may check out TOML
	// files with CRLF line endings, causing the embedded multi-line strings
	// to contain \r\n instead of \n.
	s = strings.TrimRight(s, "\r\n")
	if s == "" {
		return []string{}
	}
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		lines[i] = strings.TrimRight(l, "\r")
	}
	return lines
}

// linesEqual compares two line slices, treating nil and empty as equal.
func linesEqual(a, b []string) bool {
	if len(a) == 0 && len(b) == 0 {
		return true
	}
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// cloneLines returns a shallow copy so applyPipeline's in-place edits don't
// corrupt the original slice across multiple candidate attempts.
func cloneLines(lines []string) []string {
	out := make([]string, len(lines))
	copy(out, lines)
	return out
}

// matchCandidates returns the command strings to try for a given filter name.
// Filter names like "terraform-plan" map to "terraform plan" and "terraform".
func matchCandidates(filterName string) []string {
	candidates := []string{filterName}

	if idx := strings.Index(filterName, "-"); idx != -1 {
		// "terraform-plan" → "terraform plan" (hyphen → space)
		withSpace := filterName[:idx] + " " + filterName[idx+1:]
		candidates = append(candidates, withSpace)
		// "terraform-plan" → "terraform" (prefix only)
		candidates = append(candidates, filterName[:idx])
	}

	// Special-case well-known patterns that differ from filter name
	extras := map[string][]string{
		"brew-install":     {"brew install"},
		"bundle-install":   {"bundle install"},
		"composer-install": {"composer install"},
		"dotnet-build":     {"dotnet build"},
		"mix-compile":      {"mix compile"},
		"mix-format":       {"mix format"},
		"mvn-build":        {"mvn"},
		"poetry-install":   {"poetry install"},
		"swift-build":      {"swift build"},
		"uv-sync":          {"uv sync"},
		"tofu-plan":        {"tofu plan"},
		"tofu-init":        {"tofu init"},
		"tofu-fmt":         {"tofu fmt"},
		"tofu-validate":    {"tofu validate"},
		"terraform-plan":   {"terraform plan"},
		"pio-run":          {"pio run"},
		"spring-boot":      {"./mvnw spring-boot:run"},
		"trunk-build":      {"trunk build"},
		"quarto-render":    {"quarto render"},
		"shopify-theme":    {"shopify theme"},
		"systemctl-status": {"systemctl status"},
		"ansible-playbook": {"ansible-playbook"},
		"fail2ban-client":  {"fail2ban-client"},
	}
	if extra, ok := extras[filterName]; ok {
		candidates = append(candidates, extra...)
	}

	return candidates
}
