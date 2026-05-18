package tomlfilter

import (
	"embed"
	"io/fs"
)

// builtinFS embeds all 59 RTK-compatible filter files from filters/*.toml.
//
//go:embed filters/*.toml
var builtinFS embed.FS

// loadBuiltins parses every embedded *.toml file and merges them into a single
// Registry. Files are loaded in alphabetical order (same as RTK's build.rs).
// Invalid or unparseable files are silently skipped.
func loadBuiltins() *Registry {
	entries, err := fs.Glob(builtinFS, "filters/*.toml")
	if err != nil {
		return Empty()
	}
	regs := make([]*Registry, 0, len(entries))
	for _, path := range entries {
		data, err := builtinFS.ReadFile(path)
		if err != nil {
			continue
		}
		regs = append(regs, Parse(string(data)))
	}
	return Merge(regs...)
}
