package checks_test

import (
	"context"
	"strings"
	"testing"

	"github.com/dkoosis/conform/internal/checks"
)

// TestRun_ConformingRepo: a repo satisfying the whole Surface-1 contract
// produces zero findings. (Uses t.Setenv via fakeBD — not parallel.)
func TestRun_ConformingRepo(t *testing.T) {
	fakeBD(t, "/vault/plans", "cfm", "git+https://github.com/x/y.git")
	dir := writeRepo(t, goodRepo())

	findings := checks.Run(context.Background(), dir)
	if len(findings) != 0 {
		t.Fatalf("Run() on a conforming repo returned %d finding(s):\n%v", len(findings), findings)
	}
}

// TestRun_MissingValuesFile: no conform.json is itself a finding, and the
// remaining checks still run under the default tool profile.
func TestRun_MissingValuesFile(t *testing.T) {
	fakeBD(t, "/vault/plans", "cfm", "git+https://github.com/x/y.git")
	files := goodRepo()
	delete(files, "conform.json")
	dir := writeRepo(t, files)

	findings := checks.Run(context.Background(), dir)
	rules := rulesOf(findings)
	if rules[checks.RuleValuesFile] != 1 {
		t.Fatalf("want exactly one %s finding, got rules %v", checks.RuleValuesFile, rules)
	}
	if len(findings) != 1 {
		t.Errorf("other checks should pass on the otherwise-good repo, got %v", findings)
	}
}

// TestRun_InvalidValuesFile: the finding carries the values package's
// path-and-field error text.
func TestRun_InvalidValuesFile(t *testing.T) {
	fakeBD(t, "/vault/plans", "cfm", "git+https://github.com/x/y.git")
	files := goodRepo()
	files["conform.json"] = `{"profile": "tool", "exceptions": [{"rule": "x"}]}`
	dir := writeRepo(t, files)

	findings := checks.Run(context.Background(), dir)
	var msg string
	for _, f := range findings {
		if f.Rule == checks.RuleValuesFile {
			msg = f.Msg
		}
	}
	if msg == "" {
		t.Fatalf("want a %s finding, got %v", checks.RuleValuesFile, findings)
	}
	for _, part := range []string{"conform.json", `rule "x"`, "reason"} {
		if !strings.Contains(msg, part) {
			t.Errorf("finding message %q should name %q", msg, part)
		}
	}
}

// TestRun_ExceptionSuppressesFinding: a declared, reasoned exception is the
// one sanctioned suppression.
func TestRun_ExceptionSuppressesFinding(t *testing.T) {
	fakeBD(t, "/vault/plans", "cfm", "git+https://github.com/x/y.git")
	files := goodRepo()
	// Break the codex shape…
	files[".github/workflows/codex-review.yml"] = strings.Replace(
		goodCodexYML, "issue_comment:\n    types: [created]", "pull_request:", 1)
	// …and declare the exception.
	files["conform.json"] = `{"profile": "tool", "exceptions": [
		{"rule": "codex-workflow", "reason": "this repo pays for auto-review deliberately"}]}`
	dir := writeRepo(t, files)

	findings := checks.Run(context.Background(), dir)
	if n := rulesOf(findings)[checks.RuleCodexShape]; n != 0 {
		t.Errorf("excepted rule still produced %d finding(s): %v", n, findings)
	}

	// Same break without the exception must fail.
	files["conform.json"] = goodValues
	dir = writeRepo(t, files)
	if n := rulesOf(checks.Run(context.Background(), dir))[checks.RuleCodexShape]; n == 0 {
		t.Error("codex-workflow break produced no finding without the exception")
	}
}

// TestRun_LibProfile: libs are the contract minus deploy — a deploy verb is
// a finding, its absence is not.
func TestRun_LibProfile(t *testing.T) {
	fakeBD(t, "/vault/plans", "afl", "git+https://github.com/x/y.git")
	files := goodRepo()
	files["conform.json"] = `{"profile": "lib", "exceptions": []}`

	// Lib with deploy: finding.
	dir := writeRepo(t, files)
	if n := rulesOf(checks.Run(context.Background(), dir))[checks.RuleMakefileVerb]; n == 0 {
		t.Error("lib profile with a deploy verb produced no makefile-verbs finding")
	}

	// Lib without deploy: clean.
	files["Makefile"] = strings.Replace(goodMakefile, "deploy: build ## Install locally\n", "", 1)
	dir = writeRepo(t, files)
	if findings := checks.Run(context.Background(), dir); len(findings) != 0 {
		t.Errorf("deploy-less lib should be clean, got %v", findings)
	}
}

// TestFindingString: the printed form names file, rule, and repair — the
// contract every failure must honor.
func TestFindingString(t *testing.T) {
	t.Parallel()

	f := checks.Finding{File: "Makefile", Rule: "makefile-verbs", Msg: "m", Repair: "r"}
	s := f.String()
	for _, part := range []string{"Makefile", "makefile-verbs", "m", "fix: r"} {
		if !strings.Contains(s, part) {
			t.Errorf("String() = %q, missing %q", s, part)
		}
	}
}
