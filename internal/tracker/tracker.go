// Package tracker persists per-execution token savings to a JSONL history file
// at ~/.local/share/gotk/history.jsonl, one JSON object per line.
// This gives the same analytics as RTK's SQLite history.db without any
// external dependency — the file is directly queryable with jq.
package tracker

import (
	"bufio"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// Run is one recorded command execution.
type Run struct {
	Timestamp      time.Time `json:"ts"`
	Command        string    `json:"cmd"`
	Args           string    `json:"args,omitempty"`
	RawBytes       int       `json:"raw_bytes"`
	FilteredBytes  int       `json:"filtered_bytes"`
	RawTokens      int       `json:"raw_tokens"`
	FilteredTokens int       `json:"filtered_tokens"`
	SavedTokens    int       `json:"saved_tokens"`
	SavingsPct     float64   `json:"savings_pct"`
	ExecMs         int64     `json:"exec_ms"`
	Bypassed       bool      `json:"bypassed,omitempty"`
	ExactTokens    bool      `json:"exact_tokens,omitempty"` // true = BPE, false = bytes/4 heuristic
}

// RecordInput carries all fields needed to persist one run.
type RecordInput struct {
	Command        string
	Args           string
	RawBytes       int
	FilteredBytes  int
	RawTokens      int
	FilteredTokens int
	ExecMs         int64
	Bypassed       bool
	ExactTokens    bool
}

// Tracker appends Run records to a JSONL history file.
type Tracker struct {
	path string
	mu   sync.Mutex
}

// Open returns a Tracker backed by the history file, creating the directory
// if needed. Errors here are non-fatal for the caller.
func Open() (*Tracker, error) {
	path := historyPath()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, fmt.Errorf("create history dir: %w", err)
	}
	return &Tracker{path: path}, nil
}

// Record appends one run to the history file.
func (t *Tracker) Record(in RecordInput) error {
	saved := max(in.RawTokens-in.FilteredTokens, 0)
	pct := 0.0
	if in.RawTokens > 0 {
		pct = math.Round(float64(saved)/float64(in.RawTokens)*1000) / 10
	}

	r := Run{
		Timestamp:      time.Now().UTC(),
		Command:        in.Command,
		Args:           in.Args,
		RawBytes:       in.RawBytes,
		FilteredBytes:  in.FilteredBytes,
		RawTokens:      in.RawTokens,
		FilteredTokens: in.FilteredTokens,
		SavedTokens:    saved,
		SavingsPct:     pct,
		ExecMs:         in.ExecMs,
		Bypassed:       in.Bypassed,
		ExactTokens:    in.ExactTokens,
	}

	data, err := json.Marshal(r)
	if err != nil {
		return err
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	f, err := os.OpenFile(t.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("open history: %w", err)
	}
	defer f.Close()

	_, err = fmt.Fprintf(f, "%s\n", data)
	return err
}

// ---------- analytics ----------

// CommandStat aggregates statistics for one command across all recorded runs.
type CommandStat struct {
	Command    string
	Runs       int
	TotalSaved int
	AvgPct     float64
	AvgExecMs  int64
}

// Summary holds aggregated statistics across all recorded runs.
type Summary struct {
	TotalRuns     int
	TotalSaved    int
	AvgSavingsPct float64
	ExactTokens   bool // true if all runs used BPE tokenizer
	TopCommands   []CommandStat
	RecentRuns    []Run
}

// GetSummary reads the history file and returns aggregated stats.
func (t *Tracker) GetSummary(topN, recentN int) (Summary, error) {
	runs, err := t.readAll()
	if err != nil {
		return Summary{}, err
	}
	if len(runs) == 0 {
		return Summary{}, nil
	}

	type agg struct {
		runs       int
		totalSaved int
		sumPct     float64
		sumExecMs  int64
	}
	cmdMap := make(map[string]*agg)
	totalSaved := 0
	sumPct := 0.0
	activeCnt := 0
	allExact := true

	for _, r := range runs {
		totalSaved += r.SavedTokens
		if !r.ExactTokens {
			allExact = false
		}
		if !r.Bypassed {
			sumPct += r.SavingsPct
			activeCnt++
		}
		a, ok := cmdMap[r.Command]
		if !ok {
			a = &agg{}
			cmdMap[r.Command] = a
		}
		a.runs++
		a.totalSaved += r.SavedTokens
		a.sumPct += r.SavingsPct
		a.sumExecMs += r.ExecMs
	}

	avgPct := 0.0
	if activeCnt > 0 {
		avgPct = math.Round(sumPct/float64(activeCnt)*10) / 10
	}

	cmds := make([]CommandStat, 0, len(cmdMap))
	for cmd, a := range cmdMap {
		cmds = append(cmds, CommandStat{
			Command:    cmd,
			Runs:       a.runs,
			TotalSaved: a.totalSaved,
			AvgPct:     math.Round(a.sumPct/float64(a.runs)*10) / 10,
			AvgExecMs:  a.sumExecMs / int64(a.runs),
		})
	}
	sort.Slice(cmds, func(i, j int) bool {
		return cmds[i].TotalSaved > cmds[j].TotalSaved
	})
	if len(cmds) > topN {
		cmds = cmds[:topN]
	}

	recent := runs
	if len(recent) > recentN {
		recent = recent[len(recent)-recentN:]
	}
	for i, j := 0, len(recent)-1; i < j; i, j = i+1, j-1 {
		recent[i], recent[j] = recent[j], recent[i]
	}

	return Summary{
		TotalRuns:     len(runs),
		TotalSaved:    totalSaved,
		AvgSavingsPct: avgPct,
		ExactTokens:   allExact,
		TopCommands:   cmds,
		RecentRuns:    recent,
	}, nil
}

// Clear removes all history. Returns the number of records deleted.
func (t *Tracker) Clear() (int, error) {
	runs, err := t.readAll()
	if err != nil {
		return 0, err
	}
	return len(runs), os.Remove(t.path)
}

// ---------- internal ----------

func (t *Tracker) readAll() ([]Run, error) {
	f, err := os.Open(t.path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var runs []Run
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var r Run
		if err := json.Unmarshal(line, &r); err != nil {
			continue
		}
		runs = append(runs, r)
	}
	return runs, scanner.Err()
}

func historyPath() string {
	if dir := os.Getenv("XDG_DATA_HOME"); dir != "" {
		return filepath.Join(dir, "gotk", "history.jsonl")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(os.TempDir(), "gotk_history.jsonl")
	}
	return filepath.Join(home, ".local", "share", "gotk", "history.jsonl")
}
