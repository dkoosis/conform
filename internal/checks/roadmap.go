package checks

import (
	"os"
	"path/filepath"
	"strings"
)

// RoadmapFile is the repo's direction home: one destination sentence and the
// ordered milestones, each naming its bd epic. Beside the code, not in the kg
// — a repo that has to be read from somewhere else has no direction of its
// own (home/rules/sdlc.md §The standard).
const RoadmapFile = "ROADMAP.md"

// starPrefix marks the destination sentence. Every renderer that shows "where
// is this project going" greps for exactly this, at the start of a line.
const starPrefix = "★"

// retiredDirectionFiles are the names ROADMAP.md replaced. Their presence is
// not itself a finding — a repo may keep NORTH_STAR.md as a pointer during the
// migration — but when ROADMAP.md is absent and one of these is present, the
// repair is a rename, and saying so is worth more than "create a file".
var retiredDirectionFiles = []string{"NORTH_STAR.md"}

// checkRoadmap verifies the repo carries a ROADMAP.md with a ★ destination
// line (roadmap).
//
// Two failures, deliberately distinct. A missing file means the repo has no
// direction home at all. A file without a ★ line is worse in one specific way:
// every reader — the session-open banner, /journey, this checker — greps for
// that line, so the file exists and answers nothing, which reads as "the
// renderer is broken" rather than "the page is unfinished".
//
// What is NOT checked: milestones, epic ids, ordering, progress. Progress
// never belongs here — it derives from the bd DAG at read time — and a
// milestone list is a judgment a checker cannot make. This rule guards the
// artifact's existence and its one machine-read line; the rest is dk's.
func checkRoadmap(dir string) []Finding {
	data, err := os.ReadFile(filepath.Join(dir, RoadmapFile))
	if err != nil {
		return []Finding{{
			File:   RoadmapFile,
			Rule:   RuleRoadmap,
			Msg:    "no direction home — nothing in the repo says where this project is going",
			Repair: roadmapRepair(dir),
		}}
	}
	if !hasStarLine(string(data)) {
		return []Finding{{
			File:   RoadmapFile,
			Rule:   RuleRoadmap,
			Msg:    "no ★ line — every reader greps for it, so the file exists and answers nothing",
			Repair: `add a line starting "★ " with the destination in one sentence`,
		}}
	}
	return nil
}

// hasStarLine reports whether any line starts with the ★ marker. Leading
// whitespace does not count: readers anchor the grep at the line start, so a
// star that only a human can see is not the line they are looking for.
func hasStarLine(body string) bool {
	for line := range strings.SplitSeq(body, "\n") {
		if strings.HasPrefix(line, starPrefix) {
			return true
		}
	}
	return false
}

// roadmapRepair names the cheapest true fix. A repo still carrying one of the
// retired names is a rename, not a blank page — telling it to "create
// ROADMAP.md" would invite a second, emptier destination beside the real one.
func roadmapRepair(dir string) string {
	for _, rel := range retiredDirectionFiles {
		if _, err := os.Stat(filepath.Join(dir, rel)); err == nil {
			return "git mv " + rel + " " + RoadmapFile + " (then `conform --fix` to check its shape)"
		}
	}
	return "conform --fix (writes a " + RoadmapFile + " skeleton to fill in)"
}

// RoadmapSkeleton renders a starting ROADMAP.md for repo, named by its
// directory. Every line is a prompt: the ★ placeholder is deliberately
// unusable as-is, so a repo that runs --fix and stops still fails the ★ check
// and says why, rather than passing with a file nobody wrote.
//
// One renderer, two callers: `conform --fix` repairs an existing repo with it,
// and `conform init` emits it into a new one. That sharing is the whole reason
// to grow our own rather than reach for a template tool — the skeleton a
// scaffold emits and the shape the checker demands cannot drift when they are
// the same function.
func RoadmapSkeleton(repo string) string {
	return "# " + repo + `

<!-- Replace this comment with the destination, as one line starting "★ ".
     Until you do, ` + "`conform`" + ` stays red on the roadmap rule — a skeleton
     that passed the gate would read as direction and carry none. -->

<A paragraph on what this repo is. This file is the destination: the ★ line,
the ordered milestones, and a link to every resource the project has. Start
here and you can reach the rest. Hand-written — agents do not edit it
unprompted.>

## Milestones

Ordered. Each names its bd epic; progress is never written here — it derives
at read time from the bd DAG joined against these ids.

1. <milestone> → <bd-epic-id>

## Non-goals

- <what this project deliberately will not do>

## Resources

- <one hop to every resource the project has>
`
}
