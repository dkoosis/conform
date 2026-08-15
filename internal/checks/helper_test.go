package checks_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/dkoosis/conform/internal/checks"
)

// goodMakefile satisfies the four-verb contract for the tool profile.
const goodMakefile = `.DEFAULT_GOAL := check
GOLANGCILINT := bash scripts/lint-locked

check: vet lint test build ## Fast validation: vet + lint + test + build
audit: check race ## Exhaustive validation
deploy: build ## Install locally
help: ## Show this help
vet: ## Run go vet
lint: ## Run golangci-lint
test: ## Run tests
build: ## Compile
race: ## Race detector
`

// goodGolangci carries the baseline floor and nothing that trips it.
const goodGolangci = `version: "2"
linters:
  default: standard
  enable:
    - nilerr
    - rowserrcheck
    - noctx
    - staticcheck
    - nolintlint
  settings:
    nolintlint:
      require-explanation: true
      require-specific: true
`

const goodCheckYML = `name: check
on:
  pull_request:
jobs:
  check:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: golangci-lint (pinned in .sandbox/project.conf)
        run: |
          . ./.sandbox/project.conf
          install-golangci "$GOLANGCI_LINT_VERSION"
      - name: check
        run: make check
      - name: race
        run: make race
`

const goodCodexYML = `name: codex-review
on:
  issue_comment:
    types: [created]
jobs:
  codex:
    runs-on: ubuntu-latest
    steps:
      - run: echo review
`

const goodProjectConf = `PROJECT_NAME=x
GOLANGCI_LINT_VERSION=v2.12.2
`

const goodValues = `{"profile": "tool", "exceptions": []}` + "\n"

// goodRepo is a complete conforming tool repo, minus bd state (tests stub bd
// on PATH via fakeBD).
func goodRepo() map[string]string {
	return map[string]string{
		"conform.json":                       goodValues,
		"Makefile":                           goodMakefile,
		".golangci.yml":                      goodGolangci,
		".sandbox/project.conf":              goodProjectConf,
		".github/workflows/check.yml":        goodCheckYML,
		".github/workflows/codex-review.yml": goodCodexYML,
		".githooks/pre-commit":               "#!/bin/sh\nexit 0\n",
		".githooks/pre-push":                 "#!/bin/sh\nexit 0\n",
	}
}

// writeRepo materializes files (path → content) under a fresh temp dir.
// Paths under .githooks/ are written executable.
func writeRepo(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for rel, content := range files {
		abs := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			t.Fatal(err)
		}
		mode := os.FileMode(0o644)
		if filepath.Dir(rel) == ".githooks" {
			mode = 0o755
		}
		if err := os.WriteFile(abs, []byte(content), mode); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

// fakeBD puts a stub bd on PATH answering `bd config list`, so tests
// exercise the bd-config check without a beads workspace. Pass "" to leave a
// key unset.
func fakeBD(t *testing.T, planDir, prefix, remote string) {
	t.Helper()
	bin := t.TempDir()
	script := `#!/bin/sh
echo "Configuration:"
echo "  custom.plan_dir = ` + planDir + `"
echo "  issue_prefix = ` + prefix + `"
echo "  sync.remote = ` + remote + `"
`
	if err := os.WriteFile(filepath.Join(bin, "bd"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin)
}

// rulesOf collects the distinct rule ids present in findings.
func rulesOf(findings []checks.Finding) map[string]int {
	rules := make(map[string]int)
	for _, f := range findings {
		rules[f.Rule]++
	}
	return rules
}
