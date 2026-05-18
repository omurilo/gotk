package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Mode controls when raw output is persisted to disk.
type Mode string

const (
	ModeAlways   Mode = "always"
	ModeFailures Mode = "failures" // only persist if exit code != 0
	ModeNever    Mode = "never"
)

// TeeLogger writes raw (unfiltered) command output to a log file.
// In ModeFailures it writes to a temp file and either renames it to
// gotk_raw.log (on failure) or deletes it (on success).
// It is safe for concurrent use.
type TeeLogger struct {
	mode    Mode
	file    *os.File
	tmpPath string // only set in ModeFailures
	mu      sync.Mutex
}

// New creates a TeeLogger using the given mode.
// GOTK_LOG_DIR controls the destination directory (default: cwd).
func New(mode Mode) (*TeeLogger, error) {
	if mode == ModeNever {
		return &TeeLogger{mode: ModeNever}, nil
	}

	logDir := os.Getenv("GOTK_LOG_DIR")
	if logDir == "" {
		var err error
		logDir, err = os.Getwd()
		if err != nil {
			logDir = os.TempDir()
		}
	}

	tl := &TeeLogger{mode: mode}

	if mode == ModeFailures {
		// Write to a temp file; rename/delete in Close based on exit code.
		tmp, err := os.CreateTemp(logDir, ".gotk_raw_*.log")
		if err != nil {
			return nil, fmt.Errorf("create temp log: %w", err)
		}
		tl.file = tmp
		tl.tmpPath = tmp.Name()
	} else {
		// ModeAlways: write directly to gotk_raw.log (append)
		path := filepath.Join(logDir, "gotk_raw.log")
		f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return nil, fmt.Errorf("open log %s: %w", path, err)
		}
		tl.file = f
	}

	tl.writeMarker(fmt.Sprintf("=== gotk session start %s ===", time.Now().Format(time.RFC3339)))
	return tl, nil
}

// Write appends a single raw line to the log (thread-safe).
func (t *TeeLogger) Write(line string) {
	if t.mode == ModeNever || t.file == nil {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	fmt.Fprintln(t.file, line)
}

// Close finalises the log. exitCode is the child process exit code.
// In ModeFailures: renames the temp file to gotk_raw.log on failure,
// deletes it on success.
func (t *TeeLogger) Close(exitCode int) error {
	if t.mode == ModeNever || t.file == nil {
		return nil
	}
	t.mu.Lock()
	defer t.mu.Unlock()

	fmt.Fprintf(t.file, "=== gotk session end (exit %d) ===\n\n", exitCode)

	if err := t.file.Close(); err != nil {
		return err
	}

	if t.mode == ModeFailures {
		if exitCode != 0 {
			dest := filepath.Join(filepath.Dir(t.tmpPath), "gotk_raw.log")
			if err := os.Rename(t.tmpPath, dest); err != nil {
				return err
			}
			fmt.Fprintf(os.Stderr, "[gotk] log raw salvo em %s\n", dest)
		} else {
			_ = os.Remove(t.tmpPath)
		}
	}

	return nil
}

func (t *TeeLogger) writeMarker(s string) {
	if t.file == nil {
		return
	}
	fmt.Fprintln(t.file, s)
}
