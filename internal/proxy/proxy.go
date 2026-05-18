package proxy

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/murilo-alves/gotk/internal/config"
	"github.com/murilo-alves/gotk/internal/filter"
	"github.com/murilo-alves/gotk/internal/format"
	"github.com/murilo-alves/gotk/internal/logger"
	"github.com/murilo-alves/gotk/internal/registry"
	"github.com/murilo-alves/gotk/internal/state"
	"github.com/murilo-alves/gotk/internal/tokens"
	"github.com/murilo-alves/gotk/internal/tomlfilter"
	"github.com/murilo-alves/gotk/internal/tracker"
)

// securityKeywords cause the filter to be disabled entirely.
var securityKeywords = []string{
	"audit", "vuln", "vulnerability", "security",
	"snyk", "trivy", "cve", "scan", "owasp",
	"grype", "semgrep", "gosec", "bandit",
}

type lineEvent struct {
	text  string
	isErr bool
}

// Run executes cmd+args as a child process, filters output, tees raw to disk,
// and records token savings to the history file.
func Run(cmd string, args []string) (exitCode int, err error) {
	start := time.Now()
	cfg := config.Load()

	bypass := shouldBypass(cmd, args, cfg)

	if !bypass {
		if st, stErr := state.Load(); stErr == nil {
			_, loopBypass := st.Track(cmd, args)
			if loopBypass {
				bypass = true
				fmt.Fprintf(os.Stderr,
					"[gotk] filtro desativado: mesmo comando executado 3+ vezes (loop guard)\n")
			}
		}
	}
	if os.Getenv("GOTK_NO_FILTER") == "1" {
		bypass = true
	}

	// Open tracker (non-fatal).
	trk, _ := tracker.Open()

	teeMode := logger.Mode(cfg.Tee.Mode)
	if teeMode == "" {
		teeMode = logger.ModeFailures
	}
	tee, teeErr := logger.New(teeMode)
	if teeErr != nil {
		fmt.Fprintf(os.Stderr, "[gotk] aviso: log raw indisponível: %v\n", teeErr)
	}

	filterEngine := filter.New(cmd, bypass)
	truncEngine := filter.NewTruncator(cfg.Filter.MaxStackFrames, cfg.Filter.MaxGrepResults)

	// TOML filter registry (RTK-compatible .gotk/filters.toml / .rtk/filters.toml).
	// When a TOML filter matches the command, all output is buffered and processed
	// in bulk after the command finishes (required for tail_lines / match_output).
	tomlReg := tomlfilter.LoadAll()
	// Match against full command string (cmd + args) to match RTK's behaviour.
	fullCmd := cmd
	if len(args) > 0 {
		fullCmd = cmd + " " + strings.Join(args, " ")
	}
	useTomlFilter := !bypass && tomlReg.HasMatch(fullCmd)
	var tomlBuf []string

	// Hot-reloadable project rules (legacy regex JSON rules, still supported).
	hr := newHotRules(config.LoadProjectFilters(), bypass)
	stopWatcher := hr.startWatcher(".gotk/filters.json")
	defer close(stopWatcher)

	// Text accumulators for accurate BPE token counting at end-of-stream.
	// Using strings.Builder instead of byte counters so tiktoken gets the
	// actual text, not an approximation.
	var rawBuf, filtBuf strings.Builder

	child := exec.Command(cmd, args...)
	child.Stdin = os.Stdin

	stdoutPipe, err := child.StdoutPipe()
	if err != nil {
		return 1, fmt.Errorf("stdout pipe: %w", err)
	}
	stderrPipe, err := child.StderrPipe()
	if err != nil {
		return 1, fmt.Errorf("stderr pipe: %w", err)
	}

	if err := child.Start(); err != nil {
		return 1, fmt.Errorf("start %q: %w", cmd, err)
	}

	lines := make(chan lineEvent, 512)
	var wg sync.WaitGroup
	wg.Add(2)
	go readPipe(stdoutPipe, false, lines, &wg)
	go readPipe(stderrPipe, true, lines, &wg)
	go func() {
		wg.Wait()
		close(lines)
	}()

	// Format detection on first line.
	var fmtParser format.Parser
	fmtDetected := false

	emit := func(line string, isErr bool) {
		filtBuf.WriteString(line)
		filtBuf.WriteByte('\n')
		if isErr {
			fmt.Fprintln(os.Stderr, line)
		} else {
			fmt.Println(line)
		}
	}

	for ev := range lines {
		rawBuf.WriteString(ev.text)
		rawBuf.WriteByte('\n')
		if tee != nil {
			tee.Write(ev.text)
		}

		// TOML filter: buffer all lines, emit after command exits.
		if useTomlFilter {
			tomlBuf = append(tomlBuf, ev.text)
			continue
		}

		if !fmtDetected {
			k := format.Detect(ev.text)
			fmtParser = format.NewParser(k)
			fmtDetected = true
			if k != format.KindPlain {
				fmt.Fprintf(os.Stderr, "[gotk] formato %s detectado — saída comprimida\n", k)
			}
		}

		if fmtParser.Kind() != format.KindPlain {
			for _, out := range fmtParser.Process(ev.text) {
				emit(out, ev.isErr)
			}
			continue
		}

		// Plain pipeline: project rules → Filter → Truncator
		line := ev.text
		if !bypass {
			line = hr.apply(line)
			if line == "" {
				continue
			}
		}
		for _, filtered := range filterEngine.Process(line) {
			for _, truncated := range truncEngine.Process(filtered) {
				emit(truncated, ev.isErr)
			}
		}
	}

	// Flush TOML-buffered output through the filter pipeline.
	if useTomlFilter {
		if filtered, ok := tomlReg.Apply(fullCmd, tomlBuf); ok {
			for _, l := range filtered {
				emit(l, false)
			}
		}
	}

	// Flush structured parser
	if fmtParser != nil {
		for _, out := range fmtParser.Flush() {
			emit(out, false)
		}
	}
	// Flush filter + truncator tail
	for _, filtered := range filterEngine.Flush() {
		for _, truncated := range truncEngine.Process(filtered) {
			emit(truncated, false)
		}
	}
	for _, out := range truncEngine.Flush() {
		emit(out, false)
	}

	waitErr := child.Wait()
	exitCode = 0
	if waitErr != nil {
		var exitErr *exec.ExitError
		if errors.As(waitErr, &exitErr) {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = 1
		}
	}

	if tee != nil {
		_ = tee.Close(exitCode)
	}

	// Count tokens and record savings (non-fatal).
	if trk != nil {
		rawText := rawBuf.String()
		filtText := filtBuf.String()
		rawTok := tokens.Count(rawText)
		filtTok := tokens.Count(filtText)
		execMs := time.Since(start).Milliseconds()
		_ = trk.Record(tracker.RecordInput{
			Command:        cmd,
			Args:           strings.Join(args, " "),
			RawBytes:       len(rawText),
			FilteredBytes:  len(filtText),
			RawTokens:      rawTok,
			FilteredTokens: filtTok,
			ExecMs:         execMs,
			Bypassed:       bypass,
			ExactTokens:    tokens.IsExact(),
		})
	}

	return exitCode, nil
}

// ---------- pipe reader ----------

func readPipe(r io.Reader, isErr bool, out chan<- lineEvent, wg *sync.WaitGroup) {
	defer wg.Done()
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 4*1024*1024), 4*1024*1024)
	for scanner.Scan() {
		out <- lineEvent{text: scanner.Text(), isErr: isErr}
	}
}

// ---------- bypass detection ----------

func shouldBypass(cmd string, args []string, cfg config.Config) bool {
	all := strings.ToLower(cmd + " " + strings.Join(args, " "))
	for _, kw := range securityKeywords {
		if strings.Contains(all, kw) {
			fmt.Fprintf(os.Stderr, "[gotk] bypass de segurança: filtro desativado (%s)\n", kw)
			return true
		}
	}
	for _, excl := range cfg.Hooks.ExcludeCommands {
		if strings.EqualFold(cmd, excl) {
			return true
		}
	}
	return false
}

// ---------- hot-reloadable project rules ----------

type compiledRule struct {
	re     *regexp.Regexp
	action string
}

// hotRules holds project filter rules and reloads them from disk when the
// source file changes (polled every 250 ms).
type hotRules struct {
	mu      sync.RWMutex
	rules   []compiledRule
	bypass  bool
}

func newHotRules(pf *config.ProjectFilters, bypass bool) *hotRules {
	hr := &hotRules{bypass: bypass}
	if pf != nil {
		hr.rules = compileRules(pf.Rules)
	}
	return hr
}

// startWatcher launches a 250ms polling goroutine.
// Close the returned channel to stop it.
func (hr *hotRules) startWatcher(path string) chan struct{} {
	stop := make(chan struct{})
	if hr.bypass {
		return stop // no need to watch when filter is off
	}

	var lastMtime time.Time
	if info, err := os.Stat(path); err == nil {
		lastMtime = info.ModTime()
	}

	go func() {
		ticker := time.NewTicker(250 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				info, err := os.Stat(path)
				if err != nil {
					continue
				}
				if !info.ModTime().After(lastMtime) {
					continue
				}
				lastMtime = info.ModTime()
				pf := config.LoadProjectFilters()
				if pf == nil {
					continue
				}
				newRules := compileRules(pf.Rules)
				hr.mu.Lock()
				hr.rules = newRules
				hr.mu.Unlock()
				fmt.Fprintf(os.Stderr, "[gotk] filtros de projeto recarregados\n")
			}
		}
	}()
	return stop
}

// apply runs all project rules against a line.
// Returns empty string if the line should be suppressed.
func (hr *hotRules) apply(line string) string {
	hr.mu.RLock()
	rules := hr.rules
	hr.mu.RUnlock()
	return applyProjectRules(line, rules)
}

func compileRules(rules []config.ProjectRule) []compiledRule {
	out := make([]compiledRule, 0, len(rules))
	for _, r := range rules {
		re, err := regexp.Compile(r.Match)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[gotk] regra inválida %q: %v\n", r.Match, err)
			continue
		}
		out = append(out, compiledRule{re: re, action: r.Action})
	}
	return out
}

func applyProjectRules(line string, rules []compiledRule) string {
	for _, r := range rules {
		if r.re.MatchString(line) && r.action == "suppress" {
			return ""
		}
	}
	return line
}

// ---------- Hook mode ----------

type hookInput struct {
	SessionID      string         `json:"session_id"`
	TranscriptPath string         `json:"transcript_path"`
	ToolName       string         `json:"tool_name"`
	ToolInput      map[string]any `json:"tool_input"`
	// Cursor uses camelCase
	ToolName2 string         `json:"toolName"`
	ToolArgs  string         `json:"toolArgs"`
}

// RunHook reads a PreToolUse JSON from stdin (Claude Code, Cursor, or Gemini)
// and rewrites the Bash command to be proxied through gotk when appropriate.
// agent must be one of: "claude", "cursor", "gemini", "" (defaults to "claude").
func RunHook(agent string) error {
	if agent == "" {
		agent = "claude"
	}

	raw, err := io.ReadAll(os.Stdin)
	if err != nil {
		return fmt.Errorf("read stdin: %w", err)
	}

	var input hookInput
	if err := json.Unmarshal(raw, &input); err != nil {
		return nil
	}

	// Detect command based on agent/format
	var cmdStr string
	switch agent {
	case "cursor":
		// Cursor sends camelCase toolName + toolArgs (JSON-encoded string)
		tName := input.ToolName2
		if tName == "" {
			tName = input.ToolName
		}
		if !strings.EqualFold(tName, "bash") && !strings.EqualFold(tName, "Shell") {
			return nil
		}
		if input.ToolArgs != "" {
			var args map[string]any
			if json.Unmarshal([]byte(input.ToolArgs), &args) == nil {
				if c, ok := args["command"].(string); ok {
					cmdStr = c
				}
			}
		}
		if cmdStr == "" {
			if c, ok := input.ToolInput["command"].(string); ok {
				cmdStr = c
			}
		}
	default: // claude, gemini
		if !strings.EqualFold(input.ToolName, "Bash") &&
			!strings.EqualFold(input.ToolName, "run_shell_command") {
			return nil
		}
		if c, ok := input.ToolInput["command"].(string); ok {
			cmdStr = c
		}
	}

	if cmdStr == "" {
		return nil
	}
	if strings.HasPrefix(cmdStr, "gotk ") {
		return nil
	}

	// Check excludes from config
	cfg := config.Load()
	fields := strings.Fields(cmdStr)
	if len(fields) > 0 {
		for _, excl := range cfg.Hooks.ExcludeCommands {
			if strings.EqualFold(fields[0], excl) {
				return nil
			}
		}
	}

	rewritten, verdict := registry.Rewrite(cmdStr)
	if verdict != registry.VerdictAllow {
		return nil
	}

	// Output rewrite in agent-specific JSON format
	switch agent {
	case "cursor":
		// Cursor expects {"command": "<rewritten>"} on stdout
		out, _ := json.Marshal(map[string]string{"command": rewritten})
		fmt.Println(string(out))
	default: // claude code
		// hookSpecificOutput protocol: rewrite + auto-allow
		out, _ := json.Marshal(map[string]any{
			"hookSpecificOutput": map[string]any{
				"hookEventName":              "PreToolUse",
				"permissionDecision":         "allow",
				"permissionDecisionReason":   "gotk auto-rewrite",
				"updatedInput":               map[string]string{"command": rewritten},
			},
		})
		fmt.Println(string(out))
	}
	return nil
}
