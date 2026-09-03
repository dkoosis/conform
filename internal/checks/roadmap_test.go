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
// NORTH_STAR.md is not a retired name and never a rename target. dk edits it in
// the kg, a repo may carry it as a Publish-To reflection, and decision
// 9b4cbc91016f settles the name as final. Renaming it away would delete the
// published copy and orphan the publish target — the next publish re-creates
// it, and conform would fight the publisher forever.
func TestCheckRoadmap_NorthStarIsNeverRenamedAway(t *testing.T) {
	t.Parallel()

	files := goodRepo()
	delete(files, checks.RoadmapFile)
	files[checks.NorthStarFile] = "★ ship the thing, for dk\n"
	dir := writeRepo(t, files)

	f := findingFor(t, checks.Run(dir), checks.RuleRoadmap)
	if strings.Contains(f.Repair, "git mv") {
		t.Errorf("repair must never move %s — it is the source, not a stale copy; got %q", checks.NorthStarFile, f.Repair)
	}
	if !strings.Contains(f.Repair, "★") {
		t.Errorf("repair should say to copy the ★ line across, got %q", f.Repair)
	}
}

// The two files coexist: the kg's page owns direction, the roadmap mirrors its
// ★ line over an epic list. A repo holding both is conforming.
func TestCheckRoadmap_NorthStarAndRoadmapCoexist(t *testing.T) {
	t.Parallel()

	files := goodRepo()
	files[checks.NorthStarFile] = "★ ship the thing, for dk\n"
	dir := writeRepo(t, files)

	for _, f := range checks.Run(dir) {
		if f.Rule == checks.RuleRoadmap {
			t.Errorf("a repo holding both files should raise no roadmap finding, got %v", f)
		}
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
	if !strings.Contains(string(body), "## Epics") {
		t.Errorf("skeleton should prompt for the epic inventory, got:\n%s", body)
	}
	if strings.Contains(string(body), "Milestones") {
		t.Errorf("milestone is a banned size word — the ordered list is epics, got:\n%s", body)
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
// --fix writes the roadmap even when NORTH_STAR.md is present. The old rule
// declined here, on the theory that two files meant two destinations; they are
// one destination and one epic inventory, so declining just left the repo red.
func TestFix_WritesRoadmapBesideNorthStar(t *testing.T) {
	t.Parallel()

	files := goodRepo()
	delete(files, checks.RoadmapFile)
	files[checks.NorthStarFile] = "★ ship the thing, for dk\n"
	dir := writeRepo(t, files)

	done, err := checks.Fix(dir)
	if err != nil {
		t.Fatalf("Fix() error: %v", err)
	}
	if len(done) != 1 || !strings.Contains(done[0], checks.RoadmapFile) {
		t.Fatalf("Fix() = %v, want one line naming %s", done, checks.RoadmapFile)
	}
	if _, err := os.Stat(filepath.Join(dir, checks.NorthStarFile)); err != nil {
		t.Errorf("%s must survive untouched: %v", checks.NorthStarFile, err)
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

// The two renderers differ on exactly one thing, and it is the thing the
// checker greps for. --fix drops a page into a repo that may be unattended, so
// its skeleton must stay red; init promises a repo that passes unedited, so its
// page must carry a ★ line. Swapping them silently would either strand every
// scaffolded repo on the roadmap rule or let --fix report green on a page
// nobody wrote.
func TestRoadmapRenderers_DifferOnTheStarLine(t *testing.T) {
	repo := "widget"

	fix := checks.RoadmapSkeleton(repo)
	if findsStarLine(fix) {
		t.Error("RoadmapSkeleton must not satisfy the ★ check — --fix leaves the page red until a human writes it")
	}
	if checkRoadmapFindings(t, fix) == 0 {
		t.Error("a repo holding only the --fix skeleton should still fail the roadmap rule")
	}

	scaffold := checks.RoadmapScaffold(repo)
	if !findsStarLine(scaffold) {
		t.Error("RoadmapScaffold must satisfy the ★ check — conform init promises a repo that passes unedited")
	}
	if n := checkRoadmapFindings(t, scaffold); n != 0 {
		t.Errorf("a repo holding the init page should pass the roadmap rule, got %d finding(s)", n)
	}

	// Everything below the destination block is the same page.
	if !strings.HasSuffix(fix, roadmapTail) || !strings.HasSuffix(scaffold, roadmapTail) {
		t.Error("the two renderers should share one body below the destination block")
	}
}

const roadmapTail = "## Resources\n\n- <one hop to every resource the project has>\n"

// findsStarLine mirrors the checker's own anchoring: line start, no indent.
func findsStarLine(body string) bool {
	for line := range strings.SplitSeq(body, "\n") {
		if strings.HasPrefix(line, "★") {
			return true
		}
	}
	return false
}

// checkRoadmapFindings writes body as a repo's ROADMAP.md and counts the
// roadmap-rule findings the real checker returns for it.
func checkRoadmapFindings(t *testing.T, body string) int {
	t.Helper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, filepath.Dir(checks.RoadmapFile)), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, checks.RoadmapFile), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, f := range checks.Run(dir) {
		if f.Rule == checks.RuleRoadmap {
			n++
		}
	}
	return n
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
