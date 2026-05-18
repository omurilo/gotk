package filter

import "strings"

// ToolProfile contains per-tool line classification overrides.
// A nil function means "use the global default".
type ToolProfile struct {
	// SuppressOnly returns true for lines that should be silently dropped
	// before any other classification (progress noise, decorative headers, etc.).
	SuppressOnly func(line string) bool

	// IsSuccess returns true for lines that indicate a passing/ok result.
	// These are collapsed into a summary counter.
	IsSuccess func(line string) bool

	// IsError returns true for lines that start or continue an error block.
	// Overrides the global heuristics when non-nil.
	IsError func(line string) bool
}

// DetectProfile returns the ToolProfile matching the given command name.
// Falls back to nil (all defaults) for unknown tools.
func DetectProfile(cmd string) *ToolProfile {
	switch cmd {
	case "go":
		return goProfile
	case "cargo":
		return cargoProfile
	case "pytest", "python", "python3":
		return pytestProfile
	case "git", "gh":
		return gitProfile
	case "npm", "npx", "yarn", "pnpm", "bun":
		return nodeProfile
	case "docker", "docker-compose":
		return dockerProfile
	case "gradle", "gradlew", "./gradlew":
		return gradleProfile
	case "mvn", "mvnw", "./mvnw":
		return mavenProfile
	case "kubectl", "k9s":
		return kubectlProfile
	case "aws":
		return awsProfile
	case "tsc":
		return tscProfile
	case "jest", "vitest":
		return jestProfile
	case "eslint", "biome", "oxc":
		return eslintProfile
	case "prisma":
		return prismaProfile
	case "terraform", "tofu":
		return terraformProfile
	case "make", "gmake":
		return makeProfile
	case "bundle", "rake", "ruby", "gem":
		return rubyProfile
	case "dotnet":
		return dotnetProfile
	case "grep", "rg", "ag", "fzf":
		return grepProfile
	default:
		return nil
	}
}

// ---------- Go ----------

var goProfile = &ToolProfile{
	SuppressOnly: func(line string) bool {
		return strings.HasPrefix(line, "=== RUN") ||
			strings.HasPrefix(line, "=== CONT") ||
			strings.HasPrefix(line, "=== PAUSE")
	},
	IsSuccess: func(line string) bool {
		return strings.HasPrefix(line, "--- PASS:") ||
			strings.HasPrefix(line, "    --- PASS:") ||
			strings.HasPrefix(line, "ok  \t") ||
			strings.HasPrefix(line, "ok \t") ||
			strings.HasPrefix(line, "?   \t") ||
			strings.HasPrefix(line, "? \t")
	},
	IsError: func(line string) bool {
		return strings.HasPrefix(line, "--- FAIL:") ||
			strings.HasPrefix(line, "    --- FAIL:") ||
			strings.HasPrefix(line, "FAIL\t") ||
			strings.HasPrefix(line, "FAIL ") ||
			line == "FAIL" ||
			strings.HasPrefix(line, "panic:") ||
			strings.HasPrefix(line, "goroutine ") ||
			strings.HasPrefix(line, "runtime error:") ||
			strings.HasPrefix(line, "build failed")
	},
}

// ---------- Cargo / Rust ----------

var cargoProfile = &ToolProfile{
	SuppressOnly: func(line string) bool {
		t := strings.TrimSpace(line)
		return strings.HasPrefix(t, "Compiling ") ||
			strings.HasPrefix(t, "Checking ") ||
			strings.HasPrefix(t, "Downloading ") ||
			strings.HasPrefix(t, "Updating ") ||
			strings.HasPrefix(t, "Fetching ") ||
			strings.HasPrefix(t, "Blocking ") ||
			t == "." || t == ""
	},
	IsSuccess: func(line string) bool {
		t := strings.TrimSpace(line)
		return strings.HasPrefix(t, "Finished ") ||
			strings.HasPrefix(t, "test result: ok.") ||
			(strings.HasPrefix(t, "test ") && strings.Contains(t, " ... ok"))
	},
	IsError: func(line string) bool {
		t := strings.TrimSpace(line)
		return strings.HasPrefix(t, "error[") ||
			strings.HasPrefix(t, "error:") ||
			strings.HasPrefix(t, "FAILED") ||
			(strings.HasPrefix(t, "test ") && strings.Contains(t, " ... FAILED")) ||
			strings.HasPrefix(t, "thread '") ||
			strings.HasPrefix(t, "note: run with")
	},
}

// ---------- pytest ----------

var pytestProfile = &ToolProfile{
	SuppressOnly: func(line string) bool {
		return (strings.HasPrefix(line, "===") && strings.Contains(line, "===") &&
			!strings.Contains(line, "FAILED") && !strings.Contains(line, "ERROR") &&
			!strings.Contains(line, "passed") && !strings.Contains(line, "error")) ||
			strings.HasPrefix(line, "collecting ") ||
			strings.HasPrefix(line, "collected ") ||
			(len(line) > 0 && isOnlyDots(line))
	},
	IsSuccess: func(line string) bool {
		return strings.HasPrefix(line, "PASSED") ||
			(strings.Contains(line, " passed") && strings.HasPrefix(line, "==="))
	},
	IsError: func(line string) bool {
		return strings.HasPrefix(line, "FAILED") ||
			strings.HasPrefix(line, "ERROR") ||
			strings.HasPrefix(line, "E   ") ||
			strings.HasPrefix(line, "Traceback (most recent call last)") ||
			strings.HasPrefix(line, "AssertionError") ||
			strings.HasPrefix(line, "_ _ _") ||
			strings.HasPrefix(line, "ERRORS")
	},
}

func isOnlyDots(s string) bool {
	for _, c := range s {
		if c != '.' && c != 'F' && c != 'E' && c != 's' && c != 'x' && c != 'X' && c != ' ' {
			return false
		}
	}
	return true
}

// ---------- git ----------

var gitProfile = &ToolProfile{
	SuppressOnly: func(line string) bool {
		return line == "no changes added to commit (use \"git add\" and/or \"git commit -a\")" ||
			strings.HasPrefix(line, "  (use \"git") ||
			strings.HasPrefix(line, "\t(use \"git") ||
			strings.HasPrefix(line, "hint:")
	},
	IsSuccess: nil,
	IsError: func(line string) bool {
		return strings.HasPrefix(line, "error:") ||
			strings.HasPrefix(line, "fatal:") ||
			strings.HasPrefix(line, "CONFLICT")
	},
}

// ---------- npm / yarn / pnpm / bun ----------

var nodeProfile = &ToolProfile{
	SuppressOnly: func(line string) bool {
		lower := strings.ToLower(line)
		return strings.HasPrefix(lower, "npm warn") ||
			strings.HasPrefix(lower, "npm notice") ||
			strings.HasPrefix(lower, "npm info") ||
			strings.HasPrefix(lower, "yarn warn") ||
			strings.HasPrefix(line, "  WARN ") ||
			strings.HasPrefix(line, "  notice ") ||
			strings.HasPrefix(line, "  http fetch") ||
			strings.HasPrefix(line, "  http request") ||
			strings.Contains(line, "No repository field") ||
			strings.Contains(line, "No license field") ||
			strings.HasPrefix(line, "bun install") && strings.Contains(line, "Resolving")
	},
	IsSuccess: func(line string) bool {
		return (strings.Contains(line, "added") && strings.Contains(line, "packages")) ||
			strings.Contains(line, "Done in ") ||
			strings.HasPrefix(line, "success ") ||
			strings.HasPrefix(line, "✓")
	},
	IsError: nil,
}

// ---------- Docker ----------

var dockerProfile = &ToolProfile{
	SuppressOnly: func(line string) bool {
		t := strings.TrimSpace(line)
		return strings.HasPrefix(t, "---> Using cache") ||
			strings.HasPrefix(t, "--->") ||
			strings.HasPrefix(t, "Sending build context") ||
			strings.HasPrefix(t, "CACHED ") ||
			(len(t) == 12 && isHex(t))
	},
	IsSuccess: func(line string) bool {
		return strings.HasPrefix(line, "Successfully built") ||
			strings.HasPrefix(line, "Successfully tagged") ||
			strings.HasPrefix(line, "FINISHED")
	},
	IsError: nil,
}

func isHex(s string) bool {
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}

// ---------- Gradle ----------

var gradleProfile = &ToolProfile{
	SuppressOnly: func(line string) bool {
		return strings.HasPrefix(line, "> Configure project") ||
			strings.HasPrefix(line, "Incremental java compilation") ||
			(strings.HasPrefix(line, "Note: ") && strings.Contains(line, "uses unchecked")) ||
			(strings.HasPrefix(line, "Note: ") && strings.Contains(line, "Recompile with"))
	},
	IsSuccess: func(line string) bool {
		return (strings.Contains(line, " tests, ") && strings.Contains(line, " failures, 0")) ||
			strings.HasPrefix(line, "BUILD SUCCESSFUL")
	},
	IsError: func(line string) bool {
		return strings.HasPrefix(line, "BUILD FAILED") ||
			strings.HasPrefix(line, "FAILURE:") ||
			strings.HasPrefix(line, "* What went wrong:") ||
			strings.HasPrefix(line, "FAILED")
	},
}

// ---------- Maven ----------

var mavenProfile = &ToolProfile{
	SuppressOnly: func(line string) bool {
		return strings.HasPrefix(line, "[INFO] ---") ||
			strings.HasPrefix(line, "[INFO] Building") ||
			strings.HasPrefix(line, "[INFO] Scanning") ||
			strings.HasPrefix(line, "[INFO] ----------------------------")
	},
	IsSuccess: func(line string) bool {
		return strings.Contains(line, "BUILD SUCCESS") ||
			(strings.Contains(line, "Tests run: ") && strings.Contains(line, "Failures: 0, Errors: 0"))
	},
	IsError: func(line string) bool {
		return strings.Contains(line, "BUILD FAILURE") ||
			strings.HasPrefix(line, "[ERROR]")
	},
}

// ---------- kubectl ----------

var kubectlProfile = &ToolProfile{
	SuppressOnly: func(line string) bool {
		// Suppress resource metadata noise in `kubectl get` wide output
		return strings.HasPrefix(line, "Warning:") && strings.Contains(line, "deprecated") ||
			strings.HasPrefix(line, "I") && len(line) > 20 && line[0] == 'I' // glog info lines
	},
	IsSuccess: func(line string) bool {
		return strings.HasSuffix(line, "configured") ||
			strings.HasSuffix(line, "created") ||
			strings.HasSuffix(line, "unchanged") ||
			strings.HasSuffix(line, "deleted")
	},
	IsError: func(line string) bool {
		return strings.HasPrefix(line, "Error from server") ||
			strings.HasPrefix(line, "error:") ||
			strings.HasPrefix(line, "Error:") ||
			strings.Contains(line, "CrashLoopBackOff") ||
			strings.Contains(line, "ImagePullBackOff") ||
			strings.Contains(line, "OOMKilled")
	},
}

// ---------- AWS CLI ----------

var awsProfile = &ToolProfile{
	SuppressOnly: func(line string) bool {
		// Suppress pagination tokens and excessive metadata in JSON output
		return strings.Contains(line, "NextToken") ||
			strings.Contains(line, "NextPageToken") ||
			strings.Contains(line, "\"Marker\":")
	},
	IsSuccess: nil,
	IsError: func(line string) bool {
		return strings.HasPrefix(line, "An error occurred") ||
			strings.HasPrefix(line, "ERROR") ||
			strings.Contains(line, "AccessDenied") ||
			strings.Contains(line, "NoCredentialsError") ||
			strings.Contains(line, "ValidationError")
	},
}

// ---------- TypeScript (tsc) ----------

var tscProfile = &ToolProfile{
	SuppressOnly: func(line string) bool {
		return line == "" ||
			strings.HasPrefix(line, "Version ") ||
			strings.HasPrefix(line, "TSFILE:")
	},
	IsSuccess: func(line string) bool {
		return line == "Found 0 errors." ||
			strings.HasPrefix(line, "Found 0 errors. Watching")
	},
	IsError: func(line string) bool {
		return strings.Contains(line, ": error TS") ||
			strings.Contains(line, " errors") && !strings.Contains(line, "0 errors") ||
			strings.HasPrefix(line, "error TS")
	},
}

// ---------- Jest / Vitest ----------

var jestProfile = &ToolProfile{
	SuppressOnly: func(line string) bool {
		return strings.HasPrefix(line, "  console.") ||
			strings.Contains(line, " at Object.") && strings.Contains(line, "node_modules") ||
			strings.HasPrefix(line, "  at node_modules/") ||
			strings.HasPrefix(line, "●")
	},
	IsSuccess: func(line string) bool {
		return strings.HasPrefix(line, "✓ ") ||
			strings.HasPrefix(line, "✔ ") ||
			strings.HasPrefix(line, "  ✓ ") ||
			strings.HasPrefix(line, "PASS ") ||
			(strings.Contains(line, "passed") && strings.Contains(line, "Tests:"))
	},
	IsError: func(line string) bool {
		return strings.HasPrefix(line, "FAIL ") ||
			strings.HasPrefix(line, "✕ ") ||
			strings.HasPrefix(line, "  ✕ ") ||
			strings.HasPrefix(line, "  × ") ||
			strings.Contains(line, "● ")
	},
}

// ---------- ESLint / Biome / oxc ----------

var eslintProfile = &ToolProfile{
	SuppressOnly: func(line string) bool {
		return strings.TrimSpace(line) == "" ||
			strings.HasPrefix(line, "  (") // source context
	},
	IsSuccess: func(line string) bool {
		return strings.Contains(line, "0 problems") ||
			strings.Contains(line, "0 errors, 0 warnings") ||
			strings.HasPrefix(line, "✔ No lint")
	},
	IsError: func(line string) bool {
		return strings.Contains(line, " error ") ||
			strings.Contains(line, " errors") && !strings.Contains(line, "0 errors") ||
			strings.HasPrefix(line, "  error ")
	},
}

// ---------- Prisma ----------

var prismaProfile = &ToolProfile{
	SuppressOnly: func(line string) bool {
		return strings.HasPrefix(line, "Prisma schema loaded") ||
			strings.HasPrefix(line, "Running seed command") ||
			strings.HasPrefix(line, "Environment variables loaded")
	},
	IsSuccess: func(line string) bool {
		return strings.Contains(line, "migrations applied") ||
			strings.Contains(line, "Your database is now in sync") ||
			strings.HasPrefix(line, "✔")
	},
	IsError: func(line string) bool {
		return strings.HasPrefix(line, "Error:") ||
			strings.Contains(line, "migration failed") ||
			strings.Contains(line, "P1") || // Prisma error codes
			strings.Contains(line, "P2") ||
			strings.Contains(line, "P3")
	},
}

// ---------- Terraform / OpenTofu ----------

var terraformProfile = &ToolProfile{
	SuppressOnly: func(line string) bool {
		return strings.HasPrefix(line, "  Reading...") ||
			strings.HasPrefix(line, "  Refreshing state...") ||
			strings.HasPrefix(line, "data.") && strings.Contains(line, ": Reading...")
	},
	IsSuccess: func(line string) bool {
		return strings.HasPrefix(line, "Apply complete!") ||
			strings.HasPrefix(line, "Plan:") ||
			strings.HasPrefix(line, "No changes.")
	},
	IsError: func(line string) bool {
		return strings.HasPrefix(line, "Error:") ||
			strings.HasPrefix(line, "│ Error") ||
			strings.Contains(line, "error occurred")
	},
}

// ---------- Make ----------

var makeProfile = &ToolProfile{
	SuppressOnly: func(line string) bool {
		return strings.HasPrefix(line, "make[") && strings.Contains(line, "Entering directory") ||
			strings.HasPrefix(line, "make[") && strings.Contains(line, "Leaving directory")
	},
	IsSuccess: nil,
	IsError: func(line string) bool {
		return strings.HasPrefix(line, "make:") && strings.Contains(line, "Error") ||
			strings.HasPrefix(line, "make[") && strings.Contains(line, "Error") ||
			strings.Contains(line, "recipe for target") && strings.Contains(line, "failed")
	},
}

// ---------- Ruby (bundle/rake/gem) ----------

var rubyProfile = &ToolProfile{
	SuppressOnly: func(line string) bool {
		return strings.HasPrefix(line, "Fetching ") ||
			strings.HasPrefix(line, "Installing ") ||
			strings.HasPrefix(line, "Resolving dependencies...")
	},
	IsSuccess: func(line string) bool {
		return strings.HasPrefix(line, "Bundle complete!") ||
			strings.Contains(line, " examples, 0 failures") ||
			strings.Contains(line, "passed, 0 failures")
	},
	IsError: func(line string) bool {
		return strings.HasPrefix(line, "Bundler::") ||
			strings.Contains(line, "FAILED") ||
			strings.Contains(line, " failures") && !strings.Contains(line, "0 failures") ||
			strings.HasPrefix(line, "rspec") && strings.Contains(line, "failure")
	},
}

// ---------- .NET ----------

var dotnetProfile = &ToolProfile{
	SuppressOnly: func(line string) bool {
		return strings.HasPrefix(line, "  Determining projects to restore") ||
			strings.HasPrefix(line, "  Restored ")
	},
	IsSuccess: func(line string) bool {
		return strings.Contains(line, "Build succeeded") ||
			strings.Contains(line, "Test Run Successful") ||
			(strings.Contains(line, "Passed!") && strings.Contains(line, "Failed:     0"))
	},
	IsError: func(line string) bool {
		return strings.Contains(line, "Build FAILED") ||
			strings.HasPrefix(line, "Error") ||
			strings.Contains(line, "Failed:") && !strings.Contains(line, "Failed:     0")
	},
}

// ---------- grep / rg / ag ----------

var grepProfile = &ToolProfile{
	SuppressOnly: nil,
	IsSuccess:    nil,
	IsError: func(line string) bool {
		return strings.HasPrefix(line, "grep:") ||
			strings.HasPrefix(line, "rg:") ||
			strings.Contains(line, ": Permission denied")
	},
}
