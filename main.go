package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/murilo-alves/gotk/internal/config"
	"github.com/murilo-alves/gotk/internal/proxy"
	"github.com/murilo-alves/gotk/internal/registry"
	"github.com/murilo-alves/gotk/internal/tracker"
)

const version = "0.1.0"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "--version", "-v":
		fmt.Printf("gotk v%s\n", version)

	case "--help", "-h":
		printHelp()

	case "hook":
		// Usage: gotk hook [--agent <name>]
		// agent defaults to "claude"
		agent := flagValue(os.Args[2:], "--agent", "claude")
		if err := proxy.RunHook(agent); err != nil {
			fmt.Fprintf(os.Stderr, "gotk hook: %v\n", err)
			os.Exit(1)
		}

	case "init":
		// Usage: gotk init [--agent <name>] [--uninstall] [--show] [--dry-run]
		opts := proxy.InstallOptions{
			Agent:     flagValue(os.Args[2:], "--agent", "claude"),
			Uninstall: hasFlag(os.Args[2:], "--uninstall"),
			Show:      hasFlag(os.Args[2:], "--show"),
			DryRun:    hasFlag(os.Args[2:], "--dry-run"),
		}
		if err := proxy.Install(opts); err != nil {
			fmt.Fprintf(os.Stderr, "gotk init: %v\n", err)
			os.Exit(1)
		}

	case "rewrite":
		// Usage: gotk rewrite <command...>
		// Exits 0 (allow+rewrite), 1 (passthrough), or 2 (deny).
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: gotk rewrite <command>")
			os.Exit(1)
		}
		cmd := strings.Join(os.Args[2:], " ")
		rewritten, verdict := registry.Rewrite(cmd)
		fmt.Println(rewritten)
		os.Exit(int(verdict))

	case "trust":
		// Accept optional path: gotk trust [file]
		// Defaults: .gotk/filters.toml if present, else .gotk/filters.json
		path := ""
		if len(os.Args) >= 3 {
			path = os.Args[2]
		}
		if err := runTrust(path); err != nil {
			fmt.Fprintf(os.Stderr, "gotk trust: %v\n", err)
			os.Exit(1)
		}

	case "stats":
		if err := runStats(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "gotk stats: %v\n", err)
			os.Exit(1)
		}

	default:
		exitCode, err := proxy.Run(os.Args[1], os.Args[2:])
		if err != nil {
			fmt.Fprintf(os.Stderr, "gotk: %v\n", err)
			os.Exit(1)
		}
		os.Exit(exitCode)
	}
}

// runStats prints aggregated token savings from the history file.
func runStats(args []string) error {
	// --clear flag
	if len(args) > 0 && args[0] == "--clear" {
		trk, err := tracker.Open()
		if err != nil {
			return err
		}
		n, err := trk.Clear()
		if err != nil {
			return err
		}
		fmt.Printf("histórico limpo: %d registro(s) removido(s)\n", n)
		return nil
	}

	trk, err := tracker.Open()
	if err != nil {
		return err
	}

	sum, err := trk.GetSummary(10, 10)
	if err != nil {
		return err
	}

	if sum.TotalRuns == 0 {
		fmt.Println("nenhuma execução registrada ainda.")
		fmt.Println("use: gotk <comando> [args...]")
		return nil
	}

	tokenizerLabel := "cl100k_base BPE (tiktoken)"
	if !sum.ExactTokens {
		tokenizerLabel = "estimativa bytes/4"
	}
	fmt.Printf("gotk token savings — %d execuções registradas\n\n", sum.TotalRuns)
	fmt.Printf("Tokenizador:         %s\n", tokenizerLabel)
	fmt.Printf("Total economizado:   %s tokens\n", fmtInt(sum.TotalSaved))
	fmt.Printf("Economia média:      %.1f%%\n\n", sum.AvgSavingsPct)

	if len(sum.TopCommands) > 0 {
		fmt.Println("Top comandos por tokens economizados:")
		fmt.Printf("  %-30s  %5s  %8s  %20s  %8s\n",
			"Comando", "Runs", "Economia", "Tokens economizados", "Tempo médio")
		fmt.Println("  " + strings.Repeat("-", 78))
		for _, cs := range sum.TopCommands {
			avgTime := fmt.Sprintf("%.1fs", float64(cs.AvgExecMs)/1000)
			fmt.Printf("  %-30s  %5d  %7.1f%%  %20s  %8s\n",
				truncate(cs.Command, 30),
				cs.Runs,
				cs.AvgPct,
				fmtInt(cs.TotalSaved),
				avgTime,
			)
		}
		fmt.Println()
	}

	if len(sum.RecentRuns) > 0 {
		fmt.Println("Últimas execuções:")
		for _, r := range sum.RecentRuns {
			ts := r.Timestamp.Local().Format("2006-01-02 15:04:05")
			status := fmt.Sprintf("%.1f%% economizados", r.SavingsPct)
			if r.Bypassed {
				status = "filtro desativado"
			}
			fmt.Printf("  %s  %-28s  %-25s  %s tok  (%dms)\n",
				ts,
				truncate(r.Command+" "+r.Args, 28),
				status,
				fmtInt(r.SavedTokens),
				r.ExecMs,
			)
		}
	}

	return nil
}

// runTrust trusts a filter file. Defaults to .gotk/filters.toml if it exists,
// then falls back to .gotk/filters.json.
func runTrust(path string) error {
	if path == "" {
		if _, err := os.Stat(".gotk/filters.toml"); err == nil {
			path = ".gotk/filters.toml"
		} else {
			path = ".gotk/filters.json"
		}
	}
	if strings.HasSuffix(path, ".toml") {
		return config.TrustToml(path)
	}
	return config.Trust()
}

// ---------- flag helpers ----------

func flagValue(args []string, flag, def string) string {
	for i, a := range args {
		if a == flag && i+1 < len(args) {
			return args[i+1]
		}
		if v, ok := strings.CutPrefix(a, flag+"="); ok {
			return v
		}
	}
	return def
}

func hasFlag(args []string, flag string) bool {
	for _, a := range args {
		if a == flag {
			return true
		}
	}
	return false
}

// ---------- formatting helpers ----------

func fmtInt(n int) string {
	s := fmt.Sprintf("%d", n)
	out := make([]byte, 0, len(s)+len(s)/3)
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			out = append(out, ',')
		}
		out = append(out, byte(c))
	}
	return string(out)
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-1] + "…"
}

// ---------- help ----------

func printUsage() {
	fmt.Fprintf(os.Stderr,
		"uso: gotk <comando> [args...]\n"+
			"     gotk stats | hook | init | trust | --version | --help\n")
}

func printHelp() {
	fmt.Printf(`gotk v%s — Go Token Killer

Proxy que comprime saída de comandos em tempo real, reduzindo uso de tokens em
agentes de IA (Claude Code, Cursor, Gemini, etc.).

USO:
  gotk <comando> [args...]            Executa o comando com filtro de output
  gotk stats                          Exibe histórico de tokens economizados
  gotk stats --clear                  Limpa o histórico
  gotk rewrite <comando>              Reescreve comando (saída + exit 0/1/2)
  gotk hook [--agent <agente>]        Hook PreToolUse: reescreve JSON via stdin
  gotk init [--agent <agente>]        Instala hook para agente (padrão: claude)
  gotk init --show                    Exibe status de instalação
  gotk init --uninstall               Remove hook do agente
  gotk init --dry-run                 Simula instalação sem gravar
  gotk trust [arquivo]                Confia em arquivo de filtros TOML/JSON
  gotk --version                      Imprime versão
  gotk --help                         Imprime esta ajuda

FILTROS CUSTOMIZADOS (compatíveis com RTK):
  .gotk/filters.toml                  Filtros do projeto (gotk)
  .rtk/filters.toml                   Filtros RTK (compatibilidade direta)
  ~/.config/gotk/filters.toml         Filtros globais do usuário

  Exemplo de filtro:
    [filters.my-tool]
    description = "Compacta saída do my-tool"
    match_command = "^my-tool\\b"
    strip_ansi = true
    strip_lines_matching = ["^\\s*$", "^Downloading"]
    max_lines = 30
    on_empty = "my-tool: ok"

AGENTES SUPORTADOS (--agent):
  claude    Claude Code — ~/.claude/settings.json (padrão)
  cursor    Cursor — ~/.cursor/hooks.json
  gemini    Gemini CLI — ~/.gemini/hooks/ + settings.json
  windsurf  Windsurf Cascade — .windsurfrules (projeto)
  cline     Cline / Roo Code — .clinerules (projeto)
  all       Instala em todos os agentes acima

EXIT CODES (gotk rewrite):
  0   VerdictAllow       — reescrito como "gotk <cmd>"
  1   VerdictPassthrough — sem equivalente, passa sem alteração
  2   VerdictDeny        — comando destrutivo, nega execução

VARIÁVEIS DE AMBIENTE:
  GOTK_LOG_DIR    Diretório para gotk_raw.log (padrão: diretório atual)
  GOTK_NO_FILTER  Defina como "1" para desativar toda filtragem

BYPASS DE SEGURANÇA (automático):
  Comandos com: audit, vuln, security, snyk, trivy, cve, scan → filtro desativado

PROTEÇÃO CONTRA LOOP:
  O mesmo comando 3+ vezes em 5 min desativa o filtro (evita perda de contexto)

EXEMPLOS:
  gotk go test ./...
  gotk cargo build --release
  gotk pytest -v
  gotk git log --oneline -50
  gotk npm install
  gotk init --agent cursor
  gotk init --agent gemini
  gotk init --show
`, version)
}
