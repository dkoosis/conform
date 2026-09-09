package checks_test

import (
	"strings"
	"testing"

	"github.com/dkoosis/conform/internal/checks"
)

// stubAgents is the permitted body, copied from home/rules/standard-sdlc.md.
// It is quoted rather than derived so this test fails loudly if the rule ever
// starts rejecting the exact text the decision permits.
const stubAgents = `# Agent Instructions

Project instructions live in ` + "`.claude/rules/`" + ` — every ` + "`*.md`" + ` there, loaded
recursively. Read them. Nothing in this file is authoritative.

Task tracking is bd: run ` + "`bd prime`" + `.
`

// lardedAgents is canapay's shape, measured 2026-09-08: a pointer with a
// bd-injected managed block bolted underneath.
const lardedAgents = stubAgents + `
<!-- BEGIN BEADS INTEGRATION v:1 profile:minimal -->
## Task tracking

This repo uses bd. Run ` + "`bd ready`" + ` to see what is unblocked.
<!-- END BEADS INTEGRATION -->
`

// TestAgentsStub_StubPasses: the body the decision permits produces nothing.
// This is the green half the acceptance criteria ask for, and it is first
// because every red case below means nothing if the rule rejects the stub.
func TestAgentsStub_StubPasses(t *testing.T) {
	t.Parallel()
	files := goodRepo()
	files[checks.AgentsFile] = stubAgents
	dir := writeRepo(t, files)
	if got := checks.CheckAgentsStub(dir); len(got) != 0 {
		t.Fatalf("permitted stub: want no findings, got %+v", got)
	}
}

// TestAgentsStub_AbsentIsNotAFinding: ferret, loto and mnemd carry no
// AGENTS.md and are not wrong. The rule guards the file from becoming
// content, never its absence.
func TestAgentsStub_AbsentIsNotAFinding(t *testing.T) {
	t.Parallel()
	dir := writeRepo(t, goodRepo())
	if got := checks.CheckAgentsStub(dir); len(got) != 0 {
		t.Fatalf("no AGENTS.md: want no findings, got %+v", got)
	}
}

// TestAgentsStub_ManagedBlockIsAFinding: the case the rule exists for, with
// the repair asserted — deletion, never a fold into .claude/rules/, since
// folding preserves the content the decision says must not exist.
func TestAgentsStub_ManagedBlockIsAFinding(t *testing.T) {
	t.Parallel()
	files := goodRepo()
	files[checks.AgentsFile] = lardedAgents
	dir := writeRepo(t, files)
	got := checks.CheckAgentsStub(dir)
	if len(got) != 1 || got[0].File != checks.AgentsFile || got[0].Rule != checks.RuleAgentsStub {
		t.Fatalf("larded AGENTS.md: want one %s finding on %s, got %+v",
			checks.RuleAgentsStub, checks.AgentsFile, got)
	}
	if !strings.Contains(got[0].Repair, "delete") {
		t.Fatalf("managed block repair must say delete, got %q", got[0].Repair)
	}
	if strings.Contains(got[0].Repair, "fold") && !strings.Contains(got[0].Repair, "do not fold") {
		t.Fatalf("repair must not send the block into .claude/rules/, got %q", got[0].Repair)
	}
}

// TestAgentsStub_ShortMarkedBlockStillFires: a managed block under the line
// cap is still a finding. The two legs answer different questions — a marked
// block regenerates after a hand cleanup, a long body does not — so the
// short-block case is the one that proves the marker leg is doing work rather
// than riding on the cap.
func TestAgentsStub_ShortMarkedBlockStillFires(t *testing.T) {
	t.Parallel()
	files := goodRepo()
	files[checks.AgentsFile] = "# Agent Instructions\n\n<!-- BEGIN BEADS INTEGRATION v:1 -->\nbd\n<!-- END BEADS INTEGRATION -->\n"
	dir := writeRepo(t, files)
	got := checks.CheckAgentsStub(dir)
	if len(got) != 1 || got[0].Rule != checks.RuleAgentsStub {
		t.Fatalf("short marked block: want one %s finding, got %+v", checks.RuleAgentsStub, got)
	}
	if !strings.Contains(got[0].Msg, "managed block") {
		t.Fatalf("short marked block: want the managed-block message, got %q", got[0].Msg)
	}
}

// TestAgentsStub_LineCapFiresWithoutAMarker: the blind spot the check names —
// a tool that injects without markers is caught only by length. One body at
// the cap passes and one line more fails, so the boundary is asserted rather
// than assumed.
func TestAgentsStub_LineCapFiresWithoutAMarker(t *testing.T) {
	t.Parallel()
	body := func(n int) string {
		return strings.TrimRight(strings.Repeat("x\n", n), "\n") + "\n"
	}
	files := goodRepo()
	files[checks.AgentsFile] = body(checks.AgentsLineCap)
	if got := checks.CheckAgentsStub(writeRepo(t, files)); len(got) != 0 {
		t.Fatalf("body exactly at the %d-line cap: want no findings, got %+v",
			checks.AgentsLineCap, got)
	}

	files[checks.AgentsFile] = body(checks.AgentsLineCap + 1)
	got := checks.CheckAgentsStub(writeRepo(t, files))
	if len(got) != 1 || got[0].Rule != checks.RuleAgentsStub {
		t.Fatalf("body one line past the cap: want one %s finding, got %+v",
			checks.RuleAgentsStub, got)
	}
	if strings.Contains(got[0].Msg, "managed block") {
		t.Fatalf("unmarked long body reported as a managed block: %q", got[0].Msg)
	}
}

// TestAgentsStub_CapStaysAPointerSizedNumber: the cases above spend
// checks.AgentsLineCap rather than a literal so the test cannot drift from the
// rule — which means they also cannot notice the cap moving. This one holds
// the number itself: the permitted stub is seven lines, and a cap loose enough
// to admit a manual would make every case above vacuous.
func TestAgentsStub_CapStaysAPointerSizedNumber(t *testing.T) {
	t.Parallel()
	if checks.AgentsLineCap < strings.Count(stubAgents, "\n") || checks.AgentsLineCap > 30 {
		t.Fatalf("cap %d is not pointer-sized: it must hold the %d-line stub and refuse a manual",
			checks.AgentsLineCap, strings.Count(stubAgents, "\n"))
	}
}

// TestRun_LardedAgentsFailsTheGate: the rule is wired into Run, not just
// reachable from a test hook. Without this the check could pass its own unit
// cases while conform stayed green on a larded repo.
func TestRun_LardedAgentsFailsTheGate(t *testing.T) {
	t.Parallel()
	files := goodRepo()
	files[checks.AgentsFile] = lardedAgents
	dir := writeRepo(t, files)
	var hit bool
	for _, f := range checks.Run(dir) {
		if f.Rule == checks.RuleAgentsStub && f.File == checks.AgentsFile {
			hit = true
		}
	}
	if !hit {
		t.Fatalf("larded AGENTS.md: want an %s finding from Run, got %+v",
			checks.RuleAgentsStub, checks.Run(dir))
	}
}
