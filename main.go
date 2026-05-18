package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/murilo-alves/gotk/internal/config"
	"github.com/murilo-alves/gotk/internal/proxy"
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
		if err := proxy.RunHook(); err != nil {
			fmt.Fprintf(os.Stderr, "gotk hook: %v\n", err)
			os.Exit(1)
		}

	case "init":
		if err := proxy.Install(); err != nil {
			fmt.Fprintf(os.Stderr, "gotk init: %v\n", err)
			os.Exit(1)
		}

	case "trust":
		if err := config.Trust(); err != nil {
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
agentes de IA (Claude Code, Cursor, etc.).

USO:
  gotk <comando> [args...]   Executa o comando com filtro de output
  gotk stats                 Exibe histórico de tokens economizados
  gotk stats --clear         Limpa o histórico
  gotk hook                  Modo hook: reescreve comandos Bash (JSON via stdin)
  gotk init                  Instala o hook global no Claude Code
  gotk trust                 Confia em .gotk/filters.json no diretório atual
  gotk --version             Imprime versão
  gotk --help                Imprime esta ajuda

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
`, version)
}
