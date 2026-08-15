package checks_test

import (
	"strings"
	"testing"

	"github.com/dkoosis/conform/internal/checks"
)

func TestCIGate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		yml     string // "" = no file
		wantMsg string // "" = expect clean
	}{
		{
			name: "conforming: make check plus make race",
			yml:  goodCheckYML,
		},
		{
			name:    "missing workflow",
			wantMsg: "no check workflow",
		},
		{
			name:    "no make check step",
			yml:     strings.Replace(goodCheckYML, "run: make check", "run: make lint", 1),
			wantMsg: "no step runs",
		},
		{
			name:    "gate re-implemented in YAML",
			yml:     strings.Replace(goodCheckYML, "run: make race", "run: go test -race ./...", 1),
			wantMsg: "re-implements the gate",
		},
		{
			name:    "golangci-lint run in YAML",
			yml:     strings.Replace(goodCheckYML, "run: make race", "run: golangci-lint run ./...", 1),
			wantMsg: "re-implements the gate",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			files := map[string]string{}
			if tt.yml != "" {
				files[".github/workflows/check.yml"] = tt.yml
			}
			dir := writeRepo(t, files)

			findings := checks.CheckCIGate(dir)
			assertOneOrClean(t, findings, checks.RuleCIGate, tt.wantMsg)
		})
	}
}

func TestCodexShape(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		yml     string // "" = no file
		wantMsg string
	}{
		{
			name: "conforming: issue_comment trigger",
			yml:  goodCodexYML,
		},
		{
			name: "absent workflow is fine — only the shape is contractual",
		},
		{
			name:    "auto-fire pull_request trigger",
			yml:     strings.Replace(goodCodexYML, "issue_comment:\n    types: [created]", "pull_request:", 1),
			wantMsg: "surprise OpenAI spend",
		},
		{
			name:    "pull_request alongside issue_comment",
			yml:     strings.Replace(goodCodexYML, "on:\n", "on:\n  pull_request:\n", 1),
			wantMsg: "surprise OpenAI spend",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			files := map[string]string{}
			if tt.yml != "" {
				files[".github/workflows/codex-review.yml"] = tt.yml
			}
			dir := writeRepo(t, files)

			findings := checks.CheckCodexShape(dir)
			assertOneOrClean(t, findings, checks.RuleCodexShape, tt.wantMsg)
		})
	}
}

// assertOneOrClean expects zero findings when wantMsg is empty, else at
// least one finding of rule containing wantMsg (with a repair).
func assertOneOrClean(t *testing.T, findings []checks.Finding, rule, wantMsg string) {
	t.Helper()
	if wantMsg == "" {
		if len(findings) != 0 {
			t.Errorf("want clean, got %v", findings)
		}
		return
	}
	for _, f := range findings {
		if f.Rule == rule && strings.Contains(f.Msg, wantMsg) {
			if f.Repair == "" {
				t.Error("finding carries no repair command")
			}
			return
		}
	}
	t.Errorf("no %s finding containing %q in %v", rule, wantMsg, findings)
}
