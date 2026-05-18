package tomlfilter

import (
	"testing"
)

const testTOML = `
[filters.make]
description = "Compact make output"
match_command = "^make\\b"
strip_lines_matching = [
  "^make\\[\\d+\\]:",
  "^\\s*$",
  "^Nothing to be done",
]
max_lines = 50
on_empty = "make: ok"

[filters.gradle]
description = "Compact Gradle output"
match_command = "^(gradle|gradlew|\\./)gradlew?\\b"
strip_ansi = true
strip_lines_matching = [
  "^\\s*$",
  "^> Task :.*UP-TO-DATE$",
]
truncate_lines_at = 150
max_lines = 50
on_empty = "gradle: ok"

[filters.rsync]
description = "Compact rsync"
match_command = "^rsync\\b"
strip_lines_matching = ["^\\s*$"]
match_output = [
  { pattern = "total size is", message = "ok (synced)", unless = "error|failed" },
]
max_lines = 20

[filters.tail-test]
description = "tail_lines test"
match_command = "^tail-test\\b"
tail_lines = 2
`

func TestParse(t *testing.T) {
	reg := Parse(testTOML)
	if reg.Len() != 4 {
		t.Fatalf("expected 4 filters, got %d", reg.Len())
	}
}

func TestMakeFilter(t *testing.T) {
	reg := Parse(testTOML)
	lines := []string{
		"make[1]: Entering directory '/home/user'",
		"gcc -O2 foo.c",
		"",
		"make[1]: Leaving directory '/home/user'",
	}
	out, ok := reg.Apply("make", lines)
	if !ok {
		t.Fatal("expected match for 'make'")
	}
	if len(out) != 1 || out[0] != "gcc -O2 foo.c" {
		t.Errorf("unexpected output: %v", out)
	}
}

func TestMakeOnEmpty(t *testing.T) {
	reg := Parse(testTOML)
	lines := []string{
		"make[1]: Entering directory '/home/user'",
		"make[1]: Leaving directory '/home/user'",
	}
	out, ok := reg.Apply("make", lines)
	if !ok {
		t.Fatal("expected match")
	}
	if len(out) != 1 || out[0] != "make: ok" {
		t.Errorf("expected on_empty fallback, got: %v", out)
	}
}

func TestMatchOutput(t *testing.T) {
	reg := Parse(testTOML)
	lines := []string{
		"sending incremental file list",
		"file1.txt",
		"",
		"sent 1234 bytes",
		"total size is 98765  speedup is 77.31",
	}
	out, ok := reg.Apply("rsync", lines)
	if !ok {
		t.Fatal("expected match")
	}
	if len(out) != 1 || out[0] != "ok (synced)" {
		t.Errorf("expected match_output short-circuit, got: %v", out)
	}
}

func TestMatchOutputUnless(t *testing.T) {
	reg := Parse(testTOML)
	// match_output should NOT fire when unless pattern matches
	lines := []string{
		"rsync: error in protocol",
		"total size is 1000",
	}
	out, ok := reg.Apply("rsync", lines)
	if !ok {
		t.Fatal("expected match")
	}
	// Should NOT short-circuit — unless "error" was in output
	if len(out) == 1 && out[0] == "ok (synced)" {
		t.Error("match_output should not fire when unless pattern matches")
	}
}

func TestTailLines(t *testing.T) {
	reg := Parse(testTOML)
	lines := []string{"line1", "line2", "line3", "line4", "line5"}
	out, ok := reg.Apply("tail-test", lines)
	if !ok {
		t.Fatal("expected match")
	}
	if len(out) != 2 || out[0] != "line4" || out[1] != "line5" {
		t.Errorf("expected last 2 lines, got: %v", out)
	}
}

func TestNoMatch(t *testing.T) {
	reg := Parse(testTOML)
	_, ok := reg.Apply("unknown-command", []string{"line"})
	if ok {
		t.Error("expected no match for unknown command")
	}
}

func TestStripAnsi(t *testing.T) {
	reg := Parse(testTOML)
	lines := []string{
		"\x1b[32m> Task :app:compileJava UP-TO-DATE\x1b[0m",
		"\x1b[33m> Task :app:test\x1b[0m",
	}
	// RTK gradle regex matches full command string like "./gradlew build"
	out, ok := reg.Apply("./gradlew build", lines)
	if !ok {
		t.Fatal("expected match for ./gradlew build")
	}
	// UP-TO-DATE line should be stripped after ANSI removal
	if len(out) != 1 || out[0] != "> Task :app:test" {
		t.Errorf("unexpected output: %v", out)
	}
}

func TestFullCmdMatch(t *testing.T) {
	reg := Parse(testTOML)
	// make filter matches full command string
	if !reg.HasMatch("make clean all") {
		t.Error("expected match for 'make clean all'")
	}
	// also matches bare executable name
	if !reg.HasMatch("make") {
		t.Error("expected match for bare 'make'")
	}
}
