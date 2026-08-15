package checks

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// The three bd config keys every fleet repo carries (ratified 2026-08-09):
// plan_dir points dispatch audit trails at the vault, the prefix names the
// repo's beads, and sync.remote gives bead history an off-machine recovery
// path.
var bdKeys = []struct {
	key    string
	repair string
	why    string
}{
	{
		key:    "custom.plan_dir",
		repair: "bd config set custom.plan_dir ~/Projects/dk/Project/<repo>/plans",
		why:    "plans and dispatch audit trails have no derivable home",
	},
	{
		key:    "issue_prefix",
		repair: "bd init --prefix <2-3 letters> (or bd config set issue_prefix <p>)",
		why:    "beads mint without a stable repo prefix",
	},
	{
		key:    "sync.remote",
		repair: "bd config set sync.remote git+https://github.com/<owner>/<repo>.git",
		why:    "bead history is local-only, no off-machine recovery",
	},
}

// bdTimeout bounds the single bd invocation. One `bd config list` starts one
// bd (~0.2s); concurrent per-key gets are NOT used — parallel bd processes
// contend on the embedded Dolt lock and time each other out.
const bdTimeout = 900 * time.Millisecond

// checkBDConfig verifies the three keys are present and non-empty
// (bd-config). The keys live in bd's store, not in a parseable repo file, so
// this is the one check that shells out.
func checkBDConfig(ctx context.Context, dir string) []Finding {
	if _, err := exec.LookPath("bd"); err != nil {
		return []Finding{{
			File:   ".beads",
			Rule:   RuleBDConfig,
			Msg:    "bd not on PATH — cannot verify plan_dir, prefix, or sync.remote",
			Repair: "install beads (bd) and run bd init --prefix <p>",
		}}
	}

	got, err := bdConfigList(ctx, dir)
	if err != nil {
		return []Finding{{
			File:   ".beads",
			Rule:   RuleBDConfig,
			Msg:    fmt.Sprintf("bd config list failed (%v) — no beads workspace here, or bd cannot open it", err),
			Repair: "bd init --prefix <p>, then set custom.plan_dir and sync.remote",
		}}
	}

	var findings []Finding
	for _, k := range bdKeys {
		if got[k.key] == "" {
			findings = append(findings, Finding{
				File:   ".beads",
				Rule:   RuleBDConfig,
				Msg:    fmt.Sprintf("bd config %s is unset — %s", k.key, k.why),
				Repair: k.repair,
			})
		}
	}
	return findings
}

// bdConfigList runs one `bd config list` and parses its "key = value" lines.
func bdConfigList(ctx context.Context, dir string) (map[string]string, error) {
	ctx, cancel := context.WithTimeout(ctx, bdTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "bd", "config", "list")
	cmd.Dir = dir
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		return nil, err
	}

	values := make(map[string]string)
	for line := range strings.SplitSeq(stdout.String(), "\n") {
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		values[strings.TrimSpace(key)] = strings.TrimSpace(val)
	}
	return values, nil
}
