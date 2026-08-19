package checks_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dkoosis/conform/internal/checks"
)

// goodRoadmap is the shape the rule demands: a ★ line at the start of a line.
const goodRoadmap = "# repo\n\n★ ship the thing, for dk\n\n## Milestones\n\n1. first → bd-1\n"

// TestCheckRoadmap_Missing: no ROADMAP.md is a finding, and the repair names
// the flag that writes one.
func TestCheckRoadmap_Missing(t *testing.T) {
	t.Parallel()

	files := goodRepo()
	delete(files, checks.RoadmapFile)
	dir := writeRepo(t, files)

	findings := checks.Run(dir)
	f := findingFor(t, findings, checks.RuleRoadmap)
	if f.File != checks.RoadmapFile {
		t.Errorf("File = %q, want %q", f.File, checks.RoadmapFile)
	}
	if !strings.Contains(f.Repair, "--fix") {
		t.Errorf("repair should name the flag that writes one, got %q", f.Repair)
	}
}

// TestCheckRoadmap_Present: a ROADMAP.md with a ★ line clears the rule.
func TestCheckRoadmap_Present(t *testing.T) {
	t.Parallel()

	files := goodRepo()
	files[checks.RoadmapFile] = goodRoadmap
	dir := writeRepo(t, files)

	for _, f := range checks.Run(dir) {
		if f.Rule == checks.RuleRoadmap {
			t.Fatalf("unexpected finding: %v", f)
		}
	}
}

// TestCheckRoadmap_NoStarLine: the file exists but answers nothing. Every
// reader greps for the ★ line, so this is its own failure, not a pass.
func TestCheckRoadmap_NoStarLine(t *testing.T) {
	t.Parallel()

	files := goodRepo()
	files[checks.RoadmapFile] = "# repo\n\nprose, no destination sentence\n\n## Milestones\n"
	dir := writeRepo(t, files)

	f := findingFor(t, checks.Run(dir), checks.RuleRoadmap)
	if !strings.Contains(f.Msg, "★") {
		t.Errorf("message should name the missing line, got %q", f.Msg)
	}
}

// TestCheckRoadmap_IndentedStarIsNotTheLine: readers anchor the grep at the
// line start, so a ★ only a human can see does not count.
func TestCheckRoadmap_IndentedStarIsNotTheLine(t *testing.T) {
	t.Parallel()

	files := goodRepo()
	files[checks.RoadmapFile] = "# repo\n\n  ★ indented, so no renderer finds it\n"
	dir := writeRepo(t, files)

	findingFor(t, checks.Run(dir), checks.RuleRoadmap)
}

// TestCheckRoadmap_RetiredFilePresent: a repo still on NORTH_STAR.md is a
// rename, not a blank page — a fresh skeleton beside the old one would leave
// two destinations and no rule about which wins.
func TestCheckRoadmap_RetiredFilePresent(t *testing.T) {
	t.Parallel()

	files := goodRepo()
	delete(files, checks.RoadmapFile)
	files["NORTH_STAR.md"] = "★ the old pointer\n"
	dir := writeRepo(t, files)

	f := findingFor(t, checks.Run(dir), checks.RuleRoadmap)
	if !strings.Contains(f.Repair, "git mv") {
		t.Errorf("repair should preserve history with a rename, got %q", f.Repair)
	}
}

// TestFix_CreatesRoadmap: --fix writes a skeleton, and the skeleton it writes
// still fails the ★ check — a repo that runs --fix and stops is told so,
// rather than passing with a file nobody wrote.
func TestFix_CreatesRoadmap(t *testing.T) {
	t.Parallel()

	files := goodRepo()
	delete(files, checks.RoadmapFile)
	dir := writeRepo(t, files)

	done, err := checks.Fix(dir)
	if err != nil {
		t.Fatalf("Fix() error: %v", err)
	}
	if len(done) != 1 || !strings.Contains(done[0], checks.RoadmapFile) {
		t.Fatalf("Fix() = %v, want one line naming %s", done, checks.RoadmapFile)
	}
	body, err := os.ReadFile(filepath.Join(dir, checks.RoadmapFile))
	if err != nil {
		t.Fatalf("skeleton not written: %v", err)
	}
	if !strings.Contains(string(body), "## Milestones") {
		t.Errorf("skeleton should prompt for milestones, got:\n%s", body)
	}
	f := findingFor(t, checks.Run(dir), checks.RuleRoadmap)
	if !strings.Contains(f.Msg, "★") {
		t.Errorf("an unfilled skeleton must still fail on the ★ line, got %q", f.Msg)
	}
}

// TestFix_IsIdempotentAndNeverOverwrites: running twice is safe, and a file a
// human already wrote is never touched.
func TestFix_IsIdempotentAndNeverOverwrites(t *testing.T) {
	t.Parallel()

	files := goodRepo()
	files[checks.RoadmapFile] = goodRoadmap
	dir := writeRepo(t, files)

	done, err := checks.Fix(dir)
	if err != nil {
		t.Fatalf("Fix() error: %v", err)
	}
	if len(done) != 0 {
		t.Errorf("Fix() on a repo with a ROADMAP.md should do nothing, got %v", done)
	}
	body, err := os.ReadFile(filepath.Join(dir, checks.RoadmapFile))
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != goodRoadmap {
		t.Errorf("Fix() rewrote a hand-written file:\n%s", body)
	}
}

// TestFix_DeclinesWhenARenameIsTheRepair: NORTH_STAR.md present means the fix
// is `git mv`, which keeps history. Writing a skeleton beside it is worse than
// writing nothing.
func TestFix_DeclinesWhenARenameIsTheRepair(t *testing.T) {
	t.Parallel()

	files := goodRepo()
	delete(files, checks.RoadmapFile)
	files["NORTH_STAR.md"] = "★ the old pointer\n"
	dir := writeRepo(t, files)

	done, err := checks.Fix(dir)
	if err != nil {
		t.Fatalf("Fix() error: %v", err)
	}
	if len(done) != 0 {
		t.Errorf("Fix() should decline in favour of a rename, got %v", done)
	}
	if _, err := os.Stat(filepath.Join(dir, checks.RoadmapFile)); !os.IsNotExist(err) {
		t.Error("Fix() wrote a second destination beside the retired one")
	}
}

// TestRoadmapSkeleton_NamesTheRepo: the heading is the repo's own name, so a
// scaffolded page does not open with someone else's title.
func TestRoadmapSkeleton_NamesTheRepo(t *testing.T) {
	t.Parallel()

	got := checks.RoadmapSkeleton("widget")
	if !strings.HasPrefix(got, "# widget\n") {
		t.Errorf("skeleton should open with the repo name, got %q", firstLine(got))
	}
}

// findingFor returns the single finding for rule, failing the test otherwise.
func findingFor(t *testing.T, findings []checks.Finding, rule string) checks.Finding {
	t.Helper()
	var hits []checks.Finding
	for _, f := range findings {
		if f.Rule == rule {
			hits = append(hits, f)
		}
	}
	if len(hits) != 1 {
		t.Fatalf("want exactly one %s finding, got %d (all findings: %v)", rule, len(hits), findings)
	}
	return hits[0]
}

func firstLine(s string) string {
	head, _, _ := strings.Cut(s, "\n")
	return head
}
