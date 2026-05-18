// Package registry classifies shell commands and rewrites them to run through
// gotk, mirroring RTK's src/discover/registry.rs and rules.rs.
package registry

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Verdict is the exit-code protocol used by `gotk rewrite`, mirroring RTK:
//
//	0 VerdictAllow       rewrite found, hook may auto-allow
//	1 VerdictPassthrough no gotk equivalent, pass command through unchanged
//	2 VerdictDeny        destructive command, hook defers to agent's deny
type Verdict int

const (
	VerdictAllow       Verdict = 0
	VerdictPassthrough Verdict = 1
	VerdictDeny        Verdict = 2
)

// ruleIndex maps every command prefix to its Rule (built at package init).
var ruleIndex map[string]*Rule

func init() {
	ruleIndex = make(map[string]*Rule, len(rules)*2)
	for i := range rules {
		for _, pfx := range rules[i].Prefixes {
			ruleIndex[pfx] = &rules[i]
		}
	}
}

// Rewrite normalises cmd, looks it up in the registry, and returns the
// gotk-wrapped command plus the verdict.
//
//   - VerdictAllow:       rewritten is "gotk <normalised-cmd>", exit code 0
//   - VerdictPassthrough: rewritten == cmd unchanged, exit code 1
//   - VerdictDeny:        rewritten == cmd unchanged, exit code 2
func Rewrite(cmd string) (rewritten string, verdict Verdict) {
	cmd = strings.TrimSpace(cmd)
	if cmd == "" {
		return cmd, VerdictPassthrough
	}

	// Never rewrite commands that contain shell metacharacters or heredocs.
	if containsShellMeta(cmd) {
		return cmd, VerdictPassthrough
	}

	// Normalise: strip sudo / env-vars / absolute path prefix
	strippedSudo, cmdAfterSudo := stripSudo(cmd)
	envPrefix, cmdAfterEnv := stripEnvVars(cmdAfterSudo)
	base, args := splitBaseArgs(cmdAfterEnv)

	// Absolute-path executables: /usr/bin/git → git
	base = filepath.Base(base)

	// Shell built-ins and gotk itself are never proxied.
	if ignoredCommands[base] {
		return cmd, VerdictPassthrough
	}

	// Destructive built-ins with no useful filter.
	if isDeny(base) {
		return cmd, VerdictDeny
	}

	rule, ok := ruleIndex[base]
	if !ok {
		return cmd, VerdictPassthrough
	}

	// Check whether the subcommand is marked as interactive.
	if isInteractiveCall(args, rule.Interactive) {
		return cmd, VerdictPassthrough
	}

	// Rebuild rewritten command preserving env-var prefix and dropping sudo.
	// sudo is intentionally dropped — gotk doesn't need elevated privileges.
	_ = strippedSudo
	normalised := base
	if args != "" {
		normalised = base + " " + args
	}
	rewrittenCmd := envPrefix + "gotk " + normalised
	return rewrittenCmd, VerdictAllow
}

// Lookup returns the Rule for base (the executable name, no path).
// Returns nil if the command is not in the registry.
func Lookup(base string) *Rule {
	return ruleIndex[filepath.Base(base)]
}

// ListAll returns all rules sorted in their defined order.
func ListAll() []Rule {
	return rules
}

// ---------- helpers ----------

// shellMetaRe matches characters that indicate a compound shell expression.
var shellMetaRe = regexp.MustCompile(`[|><;` + "`" + `]|\$\(|\&\&|\|\|`)

func containsShellMeta(cmd string) bool {
	return shellMetaRe.MatchString(cmd)
}

// envVarPrefixRe matches leading "VAR=val " tokens.
var envVarPrefixRe = regexp.MustCompile(`^(?:[A-Za-z_][A-Za-z0-9_]*=[^\s]*\s+)+`)

func stripEnvVars(cmd string) (prefix, rest string) {
	loc := envVarPrefixRe.FindStringIndex(cmd)
	if loc == nil {
		return "", cmd
	}
	return cmd[:loc[1]], cmd[loc[1]:]
}

// stripSudo removes a leading "sudo" (with optional flags) from the command.
func stripSudo(cmd string) (stripped bool, rest string) {
	if !strings.HasPrefix(cmd, "sudo") {
		return false, cmd
	}
	after := strings.TrimPrefix(cmd, "sudo")
	// Must be followed by a space or flag
	if after == "" || after[0] == ' ' || after[0] == '-' {
		fields := strings.Fields(after)
		// consume "-n", "-u user", "-E" etc.
		i := 0
		for i < len(fields) {
			f := fields[i]
			if f == "-n" || f == "-E" || f == "-H" || f == "-S" || f == "-i" || f == "-k" {
				i++
				continue
			}
			if f == "-u" || f == "-g" || f == "-C" || f == "-r" || f == "-t" {
				i += 2 // flag + argument
				continue
			}
			if strings.HasPrefix(f, "-") {
				i++
				continue
			}
			break
		}
		return true, strings.Join(fields[i:], " ")
	}
	return false, cmd
}

// splitBaseArgs returns (firstWord, restOfLine) from a command string.
func splitBaseArgs(cmd string) (base, args string) {
	i := strings.IndexByte(cmd, ' ')
	if i < 0 {
		return cmd, ""
	}
	return cmd[:i], strings.TrimSpace(cmd[i+1:])
}

// isInteractiveCall returns true when the args string begins with one of the
// interactive subcommand prefixes declared in the rule.
func isInteractiveCall(args string, interactive []string) bool {
	for _, prefix := range interactive {
		if args == prefix || strings.HasPrefix(args, prefix+" ") {
			return true
		}
	}
	// Heuristic: docker/podman run/exec with -it/-i flags
	lower := strings.ToLower(args)
	if strings.HasPrefix(lower, "run ") || strings.HasPrefix(lower, "exec ") {
		if strings.Contains(args, "-it") || strings.Contains(args, "-i ") ||
			strings.Contains(args, "--interactive") {
			return true
		}
	}
	return false
}

// denyCommands are commands gotk should not proxy because they are
// dangerous, binary, or produce no useful text output.
var denyCommands = map[string]bool{
	"dd": true, "mkfs": true, "fdisk": true, "parted": true,
	"shred": true, "wipefs": true,
}

func isDeny(base string) bool {
	return denyCommands[base]
}

// ForAgent returns the list of ignored commands (useful for hook scripts).
func IgnoredCommands() []string {
	out := make([]string, 0, len(ignoredCommands))
	for k := range ignoredCommands {
		out = append(out, k)
	}
	return out
}

// GotkBin returns the path to the running gotk binary, for use in hook
// scripts that embed the full path.
func GotkBin() string {
	if exe, err := os.Executable(); err == nil {
		return exe
	}
	return "gotk"
}
