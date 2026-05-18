package registry

// Rule defines one entry in the command registry.
type Rule struct {
	// Prefixes lists every executable name that activates this rule.
	Prefixes []string
	// Category is the display name shown in `gotk stats` and `gotk list`.
	Category string
	// Interactive subcommand prefixes that should NOT be proxied
	// (e.g. "npm start", "docker exec", "go run").
	Interactive []string
	// SavingsPct is the typical token reduction for this command family.
	SavingsPct float64
}

// rules is the canonical list of every command gotk can proxy.
// Mirrors RTK's src/discover/rules.rs (881 lines), organised by category.
var rules = []Rule{

	// ─── Git ────────────────────────────────────────────────────────────────
	{
		Prefixes:   []string{"git", "yadm"},
		Category:   "Git",
		SavingsPct: 70,
	},
	{
		Prefixes:   []string{"gh"},
		Category:   "GitHub CLI",
		SavingsPct: 60,
	},
	{
		Prefixes:   []string{"glab"},
		Category:   "GitLab CLI",
		SavingsPct: 60,
	},
	{
		Prefixes:   []string{"gt"},
		Category:   "Graphite",
		SavingsPct: 55,
	},

	// ─── Rust / Cargo ───────────────────────────────────────────────────────
	{
		Prefixes:    []string{"cargo"},
		Category:    "Cargo",
		Interactive: []string{"run"},
		SavingsPct:  80,
	},
	{
		Prefixes:   []string{"rustc"},
		Category:   "Rust",
		SavingsPct: 60,
	},

	// ─── Go ─────────────────────────────────────────────────────────────────
	{
		Prefixes:    []string{"go"},
		Category:    "Go",
		Interactive: []string{"run"},
		SavingsPct:  75,
	},
	{
		Prefixes:   []string{"golangci-lint"},
		Category:   "Go",
		SavingsPct: 65,
	},
	{
		Prefixes:   []string{"gofmt", "goimports", "staticcheck"},
		Category:   "Go",
		SavingsPct: 50,
	},

	// ─── JavaScript / TypeScript ────────────────────────────────────────────
	{
		Prefixes:    []string{"npm"},
		Category:    "npm",
		Interactive: []string{"start", "run dev", "run watch", "run serve"},
		SavingsPct:  55,
	},
	{
		Prefixes:   []string{"npx"},
		Category:   "npm",
		SavingsPct: 50,
	},
	{
		Prefixes:    []string{"pnpm"},
		Category:    "pnpm",
		Interactive: []string{"start", "dev", "watch"},
		SavingsPct:  55,
	},
	{
		Prefixes:    []string{"yarn"},
		Category:    "Yarn",
		Interactive: []string{"start", "dev"},
		SavingsPct:  55,
	},
	{
		Prefixes:    []string{"bun"},
		Category:    "Bun",
		Interactive: []string{"run dev", "run start", "run watch"},
		SavingsPct:  55,
	},
	{
		Prefixes:   []string{"tsc"},
		Category:   "TypeScript",
		SavingsPct: 70,
	},
	{
		Prefixes:   []string{"jest"},
		Category:   "Jest",
		SavingsPct: 85,
	},
	{
		Prefixes:   []string{"vitest"},
		Category:   "Vitest",
		SavingsPct: 85,
	},
	{
		Prefixes:   []string{"prettier"},
		Category:   "Prettier",
		SavingsPct: 60,
	},
	{
		Prefixes:   []string{"eslint"},
		Category:   "ESLint",
		SavingsPct: 65,
	},
	{
		Prefixes:   []string{"biome"},
		Category:   "Biome",
		SavingsPct: 65,
	},
	{
		Prefixes:   []string{"oxc", "oxlint"},
		Category:   "OXC",
		SavingsPct: 65,
	},
	{
		Prefixes:   []string{"next"},
		Category:   "Next.js",
		SavingsPct: 60,
	},
	{
		Prefixes:   []string{"playwright"},
		Category:   "Playwright",
		SavingsPct: 70,
	},
	{
		Prefixes:   []string{"cypress"},
		Category:   "Cypress",
		SavingsPct: 70,
	},
	{
		Prefixes:   []string{"node"},
		Category:   "Node.js",
		SavingsPct: 40,
	},
	{
		Prefixes:   []string{"deno"},
		Category:   "Deno",
		SavingsPct: 55,
	},

	// ─── Python ─────────────────────────────────────────────────────────────
	{
		Prefixes:   []string{"pytest"},
		Category:   "pytest",
		SavingsPct: 80,
	},
	{
		Prefixes:    []string{"python", "python3"},
		Category:    "Python",
		Interactive: []string{"-c", "-i", "-m"},
		SavingsPct:  50,
	},
	{
		Prefixes:   []string{"mypy"},
		Category:   "Python",
		SavingsPct: 65,
	},
	{
		Prefixes:   []string{"ruff"},
		Category:   "Python",
		SavingsPct: 60,
	},
	{
		Prefixes:   []string{"flake8", "pylint", "pyright"},
		Category:   "Python",
		SavingsPct: 60,
	},
	{
		Prefixes:    []string{"pip", "pip3"},
		Category:    "Python",
		Interactive: []string{},
		SavingsPct:  45,
	},
	{
		Prefixes:    []string{"poetry"},
		Category:    "Poetry",
		Interactive: []string{"shell", "run"},
		SavingsPct:  50,
	},
	{
		Prefixes:    []string{"uv"},
		Category:    "uv",
		Interactive: []string{"run"},
		SavingsPct:  50,
	},
	{
		Prefixes:   []string{"pdm"},
		Category:   "PDM",
		SavingsPct: 50,
	},

	// ─── Ruby ───────────────────────────────────────────────────────────────
	{
		Prefixes:   []string{"rake"},
		Category:   "Ruby",
		SavingsPct: 70,
	},
	{
		Prefixes:   []string{"rspec"},
		Category:   "Ruby",
		SavingsPct: 80,
	},
	{
		Prefixes:   []string{"rubocop"},
		Category:   "Ruby",
		SavingsPct: 65,
	},
	{
		Prefixes:    []string{"bundle"},
		Category:    "Bundler",
		Interactive: []string{"exec", "open"},
		SavingsPct:  50,
	},
	{
		Prefixes:   []string{"ruby"},
		Category:   "Ruby",
		SavingsPct: 40,
	},
	{
		Prefixes:   []string{"gem"},
		Category:   "RubyGems",
		SavingsPct: 45,
	},

	// ─── .NET ───────────────────────────────────────────────────────────────
	{
		Prefixes:    []string{"dotnet"},
		Category:    ".NET",
		Interactive: []string{"run", "watch"},
		SavingsPct:  65,
	},

	// ─── JVM ────────────────────────────────────────────────────────────────
	{
		Prefixes:   []string{"gradle", "gradlew", "./gradlew"},
		Category:   "Gradle",
		SavingsPct: 70,
	},
	{
		Prefixes:   []string{"mvn", "mvnw", "./mvnw"},
		Category:   "Maven",
		SavingsPct: 70,
	},
	{
		Prefixes:   []string{"java"},
		Category:   "Java",
		SavingsPct: 40,
	},
	{
		Prefixes:   []string{"kotlin"},
		Category:   "Kotlin",
		SavingsPct: 40,
	},

	// ─── Swift ──────────────────────────────────────────────────────────────
	{
		Prefixes:    []string{"swift"},
		Category:    "Swift",
		Interactive: []string{"run"},
		SavingsPct:  60,
	},
	{
		Prefixes:   []string{"xcodebuild"},
		Category:   "Xcode",
		SavingsPct: 75,
	},

	// ─── PHP ────────────────────────────────────────────────────────────────
	{
		Prefixes:    []string{"composer"},
		Category:    "PHP",
		Interactive: []string{"shell"},
		SavingsPct:  50,
	},
	{
		Prefixes:   []string{"php"},
		Category:   "PHP",
		SavingsPct: 40,
	},
	{
		Prefixes:   []string{"phpunit"},
		Category:   "PHP",
		SavingsPct: 75,
	},
	{
		Prefixes:   []string{"phpstan", "psalm"},
		Category:   "PHP",
		SavingsPct: 60,
	},

	// ─── Files ──────────────────────────────────────────────────────────────
	{
		Prefixes:   []string{"cat", "bat"},
		Category:   "Files",
		SavingsPct: 30,
	},
	{
		Prefixes:   []string{"head", "tail"},
		Category:   "Files",
		SavingsPct: 30,
	},
	{
		Prefixes:   []string{"grep", "rg", "ag", "ack"},
		Category:   "Files",
		SavingsPct: 50,
	},
	{
		Prefixes:   []string{"ls"},
		Category:   "Files",
		SavingsPct: 90,
	},
	{
		Prefixes:   []string{"find", "fd"},
		Category:   "Files",
		SavingsPct: 78,
	},
	{
		Prefixes:   []string{"tree"},
		Category:   "Files",
		SavingsPct: 50,
	},
	{
		Prefixes:   []string{"diff"},
		Category:   "Files",
		SavingsPct: 30,
	},
	{
		Prefixes:   []string{"wc"},
		Category:   "Files",
		SavingsPct: 20,
	},

	// ─── Build ──────────────────────────────────────────────────────────────
	{
		Prefixes:   []string{"make"},
		Category:   "Make",
		SavingsPct: 55,
	},
	{
		Prefixes:   []string{"cmake"},
		Category:   "CMake",
		SavingsPct: 55,
	},
	{
		Prefixes:   []string{"ninja"},
		Category:   "Ninja",
		SavingsPct: 50,
	},
	{
		Prefixes:   []string{"bazel", "bazelisk"},
		Category:   "Bazel",
		SavingsPct: 65,
	},
	{
		Prefixes:   []string{"meson"},
		Category:   "Meson",
		SavingsPct: 55,
	},

	// ─── Docker / Containers ────────────────────────────────────────────────
	{
		Prefixes:    []string{"docker"},
		Category:    "Docker",
		Interactive: []string{"exec", "run", "attach", "shell"},
		SavingsPct:  65,
	},
	{
		Prefixes:    []string{"docker-compose"},
		Category:    "Docker Compose",
		Interactive: []string{"run", "exec", "up", "logs -f"},
		SavingsPct:  60,
	},
	{
		Prefixes:    []string{"podman"},
		Category:    "Podman",
		Interactive: []string{"exec", "run"},
		SavingsPct:  65,
	},

	// ─── Kubernetes ─────────────────────────────────────────────────────────
	{
		Prefixes:    []string{"kubectl"},
		Category:    "Kubernetes",
		Interactive: []string{"exec", "port-forward", "proxy"},
		SavingsPct:  70,
	},
	{
		Prefixes:   []string{"helm"},
		Category:   "Helm",
		SavingsPct: 60,
	},
	{
		Prefixes:   []string{"kustomize"},
		Category:   "Kubernetes",
		SavingsPct: 55,
	},

	// ─── Cloud / Infrastructure ─────────────────────────────────────────────
	{
		Prefixes:   []string{"aws"},
		Category:   "AWS",
		SavingsPct: 55,
	},
	{
		Prefixes:   []string{"gcloud"},
		Category:   "GCP",
		SavingsPct: 55,
	},
	{
		Prefixes:   []string{"az"},
		Category:   "Azure",
		SavingsPct: 55,
	},
	{
		Prefixes:   []string{"terraform", "tofu"},
		Category:   "Terraform",
		SavingsPct: 60,
	},
	{
		Prefixes:   []string{"ansible", "ansible-playbook"},
		Category:   "Ansible",
		SavingsPct: 60,
	},
	{
		Prefixes:   []string{"pulumi"},
		Category:   "Pulumi",
		SavingsPct: 55,
	},

	// ─── Database ───────────────────────────────────────────────────────────
	{
		Prefixes:    []string{"psql"},
		Category:    "PostgreSQL",
		Interactive: []string{},
		SavingsPct:  50,
	},
	{
		Prefixes:    []string{"mysql", "mariadb"},
		Category:    "MySQL",
		Interactive: []string{},
		SavingsPct:  50,
	},
	{
		Prefixes:    []string{"sqlite3"},
		Category:    "SQLite",
		Interactive: []string{},
		SavingsPct:  50,
	},
	{
		Prefixes:   []string{"prisma"},
		Category:   "Prisma",
		SavingsPct: 65,
	},
	{
		Prefixes:   []string{"flyway", "liquibase"},
		Category:   "Database",
		SavingsPct: 60,
	},

	// ─── Package Managers ────────────────────────────────────────────────────
	{
		Prefixes:    []string{"brew"},
		Category:    "Homebrew",
		Interactive: []string{},
		SavingsPct:  45,
	},
	{
		Prefixes:   []string{"apt", "apt-get"},
		Category:   "APT",
		SavingsPct: 50,
	},
	{
		Prefixes:   []string{"dnf", "yum"},
		Category:   "DNF/YUM",
		SavingsPct: 50,
	},

	// ─── Network ────────────────────────────────────────────────────────────
	{
		Prefixes:   []string{"curl"},
		Category:   "Network",
		SavingsPct: 40,
	},
	{
		Prefixes:   []string{"wget"},
		Category:   "Network",
		SavingsPct: 40,
	},

	// ─── System ─────────────────────────────────────────────────────────────
	{
		Prefixes:   []string{"ps"},
		Category:   "System",
		SavingsPct: 50,
	},
	{
		Prefixes:   []string{"df", "du"},
		Category:   "System",
		SavingsPct: 55,
	},
	{
		Prefixes:    []string{"systemctl"},
		Category:    "System",
		Interactive: []string{},
		SavingsPct:  50,
	},
	{
		Prefixes:   []string{"journalctl"},
		Category:   "System",
		SavingsPct: 60,
	},
	{
		Prefixes:   []string{"lsof", "netstat", "ss"},
		Category:   "System",
		SavingsPct: 55,
	},

	// ─── Linters / Quality ───────────────────────────────────────────────────
	{
		Prefixes:   []string{"shellcheck"},
		Category:   "Lint",
		SavingsPct: 60,
	},
	{
		Prefixes:   []string{"yamllint"},
		Category:   "Lint",
		SavingsPct: 60,
	},
	{
		Prefixes:   []string{"hadolint"},
		Category:   "Lint",
		SavingsPct: 65,
	},
	{
		Prefixes:   []string{"markdownlint"},
		Category:   "Lint",
		SavingsPct: 55,
	},
	{
		Prefixes:   []string{"pre-commit"},
		Category:   "Lint",
		SavingsPct: 60,
	},
	{
		Prefixes:   []string{"semgrep"},
		Category:   "Security",
		SavingsPct: 65,
	},

	// ─── Mix / Elixir ───────────────────────────────────────────────────────
	{
		Prefixes:    []string{"mix"},
		Category:    "Elixir",
		Interactive: []string{"phx.server", "iex"},
		SavingsPct:  65,
	},

	// ─── Misc Dev Tools ─────────────────────────────────────────────────────
	{
		Prefixes:   []string{"quarto"},
		Category:   "Quarto",
		SavingsPct: 55,
	},
	{
		Prefixes:   []string{"pio"},
		Category:   "PlatformIO",
		SavingsPct: 60,
	},
	{
		Prefixes:   []string{"nx"},
		Category:   "Nx",
		SavingsPct: 65,
	},
	{
		Prefixes:   []string{"turbo"},
		Category:   "Turborepo",
		SavingsPct: 60,
	},
	{
		Prefixes:   []string{"lerna"},
		Category:   "Lerna",
		SavingsPct: 55,
	},
}

// ignoredCommands is the set of shell built-ins and utilities that gotk
// should never attempt to proxy (mirrors RTK's ignore list).
var ignoredCommands = map[string]bool{
	"cd": true, "echo": true, "printf": true, "export": true,
	"source": true, ".": true, "mkdir": true, "rm": true,
	"mv": true, "cp": true, "chmod": true, "chown": true,
	"touch": true, "which": true, "type": true, "test": true,
	"true": true, "false": true, "sleep": true, "wait": true,
	"kill": true, "set": true, "unset": true, "sort": true,
	"uniq": true, "tr": true, "cut": true, "awk": true,
	"sed": true, "pwd": true, "bash": true, "sh": true,
	"zsh": true, "fish": true, "then": true, "else": true,
	"do": true, "for": true, "while": true, "if": true,
	"case": true, "gotk": true, "rtk": true, "env": true,
	"exec": true, "eval": true, "read": true, "exit": true,
	"return": true,
}
