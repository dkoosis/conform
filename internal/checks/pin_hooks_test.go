package checks_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dkoosis/conform/internal/checks"
)

func TestLintPin(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		files   map[string]string
		wantMsg string
	}{
		{
			name: "one pin in project.conf, workflow reads the key",
			files: map[string]string{
				".sandbox/project.conf":       goodProjectConf,
				".github/workflows/check.yml": goodCheckYML,
			},
		},
		{
			name:    "no pin anywhere",
			files:   map[string]string{".sandbox/project.conf": "PROJECT_NAME=x\n"},
			wantMsg: "not pinned",
		},
		{
			name: "second pin hardcoded in the workflow",
			files: map[string]string{
				".sandbox/project.conf": goodProjectConf,
				".github/workflows/check.yml": strings.Replace(goodCheckYML,
					`install-golangci "$GOLANGCI_LINT_VERSION"`,
					"install-golangci v2.12.2 # golangci-lint", 1),
			},
			wantMsg: "second golangci-lint pin",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dir := writeRepo(t, tt.files)
			assertOneOrClean(t, checks.CheckLintPin(dir), checks.RuleLintPin, tt.wantMsg)
		})
	}
}

func TestRetiredFiles(t *testing.T) {
	t.Parallel()

	dir := writeRepo(t, map[string]string{
		"scripts/check-pack-drift.sh": "#!/bin/sh\nexit 0\n",
	})
	assertOneOrClean(t, checks.CheckRetiredFiles(dir), checks.RuleRetiredFiles, "retired fleet-wide")

	clean := writeRepo(t, map[string]string{"scripts/lint-locked": "#!/bin/sh\n"})
	assertOneOrClean(t, checks.CheckRetiredFiles(clean), checks.RuleRetiredFiles, "")
}

func TestHooksShape(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		files   map[string]string
		wantMsg string
	}{
		{
			name: "shape B: tracked executable pre-commit and pre-push",
			files: map[string]string{
				".githooks/pre-commit": "#!/bin/sh\nexit 0\n",
				".githooks/pre-push":   "#!/bin/sh\nexit 0\n",
			},
		},
		{
			name:    "no .githooks directory",
			files:   map[string]string{},
			wantMsg: "shape B is the one accepted hook shape",
		},
		{
			name: "pre-push missing",
			files: map[string]string{
				".githooks/pre-commit": "#!/bin/sh\nexit 0\n",
			},
			wantMsg: "pre-push hook missing",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dir := writeRepo(t, tt.files)
			assertOneOrClean(t, checks.CheckHooksShape(dir), checks.RuleHooksShape, tt.wantMsg)
		})
	}
}

func TestHooksShape_NotExecutable(t *testing.T) {
	t.Parallel()

	dir := writeRepo(t, map[string]string{
		".githooks/pre-commit": "#!/bin/sh\nexit 0\n",
		".githooks/pre-push":   "#!/bin/sh\nexit 0\n",
	})
	if err := os.Chmod(filepath.Join(dir, ".githooks", "pre-push"), 0o644); err != nil {
		t.Fatal(err)
	}
	assertOneOrClean(t, checks.CheckHooksShape(dir), checks.RuleHooksShape, "not executable")
}

// TestBDConfig covers the shelled check via a stubbed bd (not parallel:
// fakeBD uses t.Setenv).
func TestBDConfig(t *testing.T) {
	dir := writeRepo(t, map[string]string{})

	t.Run("all keys set", func(t *testing.T) {
		fakeBD(t, "/vault/plans", "cfm", "git+https://github.com/x/y.git")
		assertOneOrClean(t, checks.CheckBDConfig(context.Background(), dir), checks.RuleBDConfig, "")
	})

	t.Run("sync.remote unset", func(t *testing.T) {
		fakeBD(t, "/vault/plans", "cfm", "")
		assertOneOrClean(t, checks.CheckBDConfig(context.Background(), dir), checks.RuleBDConfig, "sync.remote is unset")
	})

	t.Run("bd not on PATH", func(t *testing.T) {
		t.Setenv("PATH", t.TempDir())
		assertOneOrClean(t, checks.CheckBDConfig(context.Background(), dir), checks.RuleBDConfig, "bd not on PATH")
	})
}
