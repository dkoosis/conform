package checks

import (
	"fmt"
	"os"
	"path/filepath"
)

// Fix applies the Surface-1 repairs conform can make without judgment, and
// returns one line per action taken. A clean repo returns nothing.
//
// The bar for living here is narrow on purpose: the repair must be the ONLY
// correct one, and it must never destroy work. Creating a file that is absent
// qualifies. Rewriting one that exists does not — conform would be guessing at
// content a human wrote, and a checker that edits your prose stops being
// trusted long before it stops being right. Everything else stays a Repair
// string on the finding, for a person to run.
//
// Fix never overwrites, so running it twice is safe and the second run is
// silent.
func Fix(dir string) ([]string, error) {
	var done []string

	created, err := fixRoadmap(dir)
	if err != nil {
		return done, err
	}
	if created != "" {
		done = append(done, created)
	}

	return done, nil
}

// fixRoadmap writes a ROADMAP.md skeleton when the repo has no direction home.
//
// It deliberately declines when a retired direction file is still present: the
// right move there is `git mv`, which keeps the file's history, and a fresh
// skeleton beside the old page would leave the repo with two destinations and
// no rule about which one wins. The finding already names that repair.
func fixRoadmap(dir string) (string, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	path := filepath.Join(abs, RoadmapFile)
	if _, err := os.Stat(path); err == nil {
		return "", nil // present — its ★ line is the checker's business, not ours
	} else if !os.IsNotExist(err) {
		return "", err
	}
	for _, rel := range retiredDirectionFiles {
		if _, err := os.Stat(filepath.Join(abs, rel)); err == nil {
			return "", nil
		}
	}

	body := RoadmapSkeleton(filepath.Base(abs))
	// 0o600: the file is world-readable the moment git tracks it, so a wider
	// mode here buys nothing and trips gosec. #nosec is a worse answer than a
	// mode that is simply correct.
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		return "", err
	}
	return fmt.Sprintf("created %s — fill in the ★ line and the milestones", RoadmapFile), nil
}
