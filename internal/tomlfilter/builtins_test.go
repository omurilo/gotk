package tomlfilter

import (
	"strings"
	"testing"
)

func TestBuiltinsLoad(t *testing.T) {
	reg := loadBuiltins()
	if reg.Len() < 50 {
		t.Fatalf("expected ≥50 built-in filters, got %d", reg.Len())
	}
	t.Logf("loaded %d built-in filters", reg.Len())
}

func TestBuiltinsNoParseErrors(t *testing.T) {
	// Every embedded file must parse without error; none should be silently dropped.
	entries, _ := builtinFS.ReadDir("filters")
	total := len(entries)
	reg := loadBuiltins()
	if reg.Len() != total {
		t.Errorf("expected %d filters (one per file), got %d — some files failed to parse", total, reg.Len())
	}
}

func TestBuiltinsMakeFilter(t *testing.T) {
	reg := loadBuiltins()
	lines := []string{
		"make[1]: Entering directory '/home/user'",
		"gcc -O2 foo.c",
		"",
		"make[1]: Leaving directory '/home/user'",
	}
	out, ok := reg.Apply("make", lines)
	if !ok {
		t.Fatal("built-in make filter not found")
	}
	if len(out) != 1 || out[0] != "gcc -O2 foo.c" {
		t.Errorf("make filter: unexpected output %v", out)
	}
}

func TestBuiltinsGradleFilter(t *testing.T) {
	reg := loadBuiltins()
	lines := []string{
		"> Configuring project :app",
		"> Task :app:compileJava UP-TO-DATE",
		"> Task :app:test",
		"",
		"3 tests completed, 1 failed",
		"",
		"BUILD FAILED in 12s",
	}
	out, ok := reg.Apply("./gradlew build", lines)
	if !ok {
		t.Fatal("built-in gradle filter not found")
	}
	joined := strings.Join(out, "\n")
	if !strings.Contains(joined, "BUILD FAILED") {
		t.Errorf("gradle filter: BUILD FAILED missing from output: %v", out)
	}
	for _, l := range out {
		if strings.Contains(l, "UP-TO-DATE") {
			t.Errorf("gradle filter: UP-TO-DATE line should have been stripped: %q", l)
		}
		if strings.Contains(l, "Configuring project") {
			t.Errorf("gradle filter: Configuring line should have been stripped: %q", l)
		}
	}
}

func TestBuiltinsTerraformPlan(t *testing.T) {
	reg := loadBuiltins()
	lines := []string{
		"Acquiring state lock. This may take a few moments...",
		"Refreshing state... [id=vpc-abc]",
		"",
		"Terraform will perform the following actions:",
		"",
		"  # aws_instance.web will be created",
		"  + resource \"aws_instance\" \"web\" {}",
		"",
		"Plan: 1 to add, 0 to change, 0 to destroy.",
	}
	out, ok := reg.Apply("terraform plan", lines)
	if !ok {
		t.Fatal("built-in terraform-plan filter not found")
	}
	joined := strings.Join(out, "\n")
	if strings.Contains(joined, "Refreshing state") {
		t.Error("terraform-plan: Refreshing state should be stripped")
	}
	if strings.Contains(joined, "Acquiring state lock") {
		t.Error("terraform-plan: Acquiring state lock should be stripped")
	}
	if !strings.Contains(joined, "Plan: 1 to add") {
		t.Error("terraform-plan: summary line missing from output")
	}
}

func TestBuiltinsRsyncMatchOutput(t *testing.T) {
	reg := loadBuiltins()
	lines := []string{
		"sending incremental file list",
		"./",
		"file1.txt",
		"",
		"sent 1,234 bytes  received 42 bytes",
		"total size is 98,765  speedup is 77.31",
	}
	out, ok := reg.Apply("rsync -avz src/ dst/", lines)
	if !ok {
		t.Fatal("built-in rsync filter not found")
	}
	if len(out) != 1 || out[0] != "ok (synced)" {
		t.Errorf("rsync match_output short-circuit failed: %v", out)
	}
}

func TestBuiltinsProjectOverridesBuiltin(t *testing.T) {
	// A project filter with same command should take priority over built-in.
	projectReg := Parse(`
[filters.make]
description = "Custom project make filter"
match_command = "^make\\b"
on_empty = "project: make ok"
strip_lines_matching = [".*"]
`)
	combined := Merge(projectReg, loadBuiltins())
	out, ok := combined.Apply("make", []string{"make[1]: some line"})
	if !ok {
		t.Fatal("expected match")
	}
	// Project filter strips everything and returns its own on_empty
	if len(out) != 1 || out[0] != "project: make ok" {
		t.Errorf("project filter should override builtin, got: %v", out)
	}
}
