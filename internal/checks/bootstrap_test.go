package checks

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestBootstrapPlanIsLocalFirst: every remote step sits after every local
// one, so an interrupted run never leaves GitHub ahead of the working tree.
func TestBootstrapPlanIsLocalFirst(t *testing.T) {
	seenRemote := false
	for _, step := range BootstrapPlan(testSpec()) {
		if step.Remote {
			seenRemote = true
			continue
		}
		if seenRemote {
			t.Errorf("local step %q follows a remote one", step)
		}
	}
}

// TestBootstrapPlanSetsEveryTrackedBDKey: the live config the plan sets is
// the declaration the renderer wrote — same bdKeys slice, so --local's
// bd-config-live comparison starts equal.
func TestBootstrapPlanSetsEveryTrackedBDKey(t *testing.T) {
	plan := BootstrapPlan(testSpec())
	for _, k := range bdKeys {
		if bdInitOwnedKeys[k.key] {
			// `bd config set` rejects these; bd init establishes them.
			// Assert the plan does NOT try, rather than that it does.
			for _, step := range plan {
				if len(step.Argv) >= 4 && step.Argv[0] == "bd" && step.Argv[2] == "set" && step.Argv[3] == k.key {
					t.Errorf("plan runs `bd config set %s`, which bd rejects", k.key)
				}
			}
			continue
		}
		found := false
		for _, step := range plan {
			if len(step.Argv) >= 4 && step.Argv[0] == "bd" && step.Argv[2] == "set" && step.Argv[3] == k.key {
				found = true
				if got := step.Argv[4]; got != bdConfigValue(k.key, withDefaults(testSpec())) {
					t.Errorf("bd config set %s = %q, does not match the tracked declaration", k.key, got)
				}
			}
		}
		if !found {
			t.Errorf("plan never sets %q — the live config will not match %s", k.key, bdConfigFile)
		}
	}
}

// TestBootstrapPlanWiresHooksPath: the plan runs checkHooksPath's own repair.
func TestBootstrapPlanWiresHooksPath(t *testing.T) {
	want := "git config core.hooksPath " + hooksDir
	for _, step := range BootstrapPlan(testSpec()) {
		if step.String() == want {
			return
		}
	}
	t.Fatalf("plan does not run %q", want)
}

// TestRemoteStepsAreMarkedRemote: every gh step is flagged, so the default
// path cannot execute one by accident.
func TestRemoteStepsAreMarkedRemote(t *testing.T) {
	for _, step := range BootstrapPlan(testSpec()) {
		if step.Argv[0] == "gh" && !step.Remote {
			t.Errorf("gh step not marked remote: %s", step)
		}
		if step.Argv[0] != "gh" && step.Remote {
			t.Errorf("non-gh step marked remote: %s", step)
		}
	}
}

// TestBootstrapPlanCoversEveryFleetLabel guards the same sharing as the
// scaffold proof: the labels the plan creates come from the slice --fleet
// checks for.
func TestBootstrapPlanCoversEveryFleetLabel(t *testing.T) {
	old := fleetLabels
	t.Cleanup(func() { fleetLabels = old })
	fleetLabels = append(append([]string{}, old...), "sentinel-label")

	plan := BootstrapPlan(testSpec())
	for _, want := range fleetLabels {
		found := false
		for _, step := range plan {
			if step.Argv[0] == "gh" && len(step.Argv) > 3 && step.Argv[1] == "label" && step.Argv[3] == want {
				found = true
			}
		}
		if !found {
			t.Errorf("plan never creates label %q", want)
		}
	}
}

// TestProtectionPayloadCarriesTheFleetAmendment: strict:false and the same
// required context --fleet demands.
func TestProtectionPayloadCarriesTheFleetAmendment(t *testing.T) {
	payload := ProtectionPayload()
	if !strings.Contains(payload, `"strict":false`) {
		t.Errorf("payload is not strict:false — parallel agent PRs pay a re-run tax: %s", payload)
	}
	if !strings.Contains(payload, `"`+requiredCheckContext+`"`) {
		t.Errorf("payload omits the required context %q: %s", requiredCheckContext, payload)
	}
	if !strings.Contains(payload, `"enforce_admins":true`) {
		t.Errorf("payload leaves an owner-shaped hole: %s", payload)
	}
}

// TestBootstrapDryRunTouchesNothing: the dry run must not create a git dir,
// a bd store, or anything else in the target.
func TestBootstrapDryRunTouchesNothing(t *testing.T) {
	dir := t.TempDir()
	out, err := os.CreateTemp(t.TempDir(), "out")
	if err != nil {
		t.Fatal(err)
	}
	defer out.Close()

	if err := Bootstrap(context.Background(), dir, testSpec(), BootstrapOpts{DryRun: true, Out: out}); err != nil {
		t.Fatalf("Bootstrap: %v", err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("dry run wrote %d entries into the target dir", len(entries))
	}
	data, err := os.ReadFile(out.Name())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "would run: git init") {
		t.Errorf("dry run did not report the plan: %s", data)
	}
}

// TestBootstrapSkipsRemoteByDefault: without WithRemote, no gh step runs —
// the whole reason remote work is a flag and not a default.
func TestBootstrapSkipsRemoteByDefault(t *testing.T) {
	dir := t.TempDir()
	out, err := os.CreateTemp(t.TempDir(), "out")
	if err != nil {
		t.Fatal(err)
	}
	defer out.Close()

	// Put a fake gh on PATH that fails loudly if it is ever executed.
	bin := t.TempDir()
	tripwire := filepath.Join(bin, "gh")
	if err := os.WriteFile(tripwire, []byte("#!/bin/sh\necho 'gh executed' >&2\nexit 1\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin) // bd and git also vanish; their steps report and skip

	if err := Bootstrap(context.Background(), dir, testSpec(), BootstrapOpts{Out: out}); err != nil {
		t.Fatalf("Bootstrap: %v", err)
	}
	data, err := os.ReadFile(out.Name())
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "✓ gh ") {
		t.Errorf("a gh step ran without WithRemote: %s", data)
	}
	if !strings.Contains(string(data), "skipped (remote): gh") {
		t.Errorf("remote steps were not reported as skipped: %s", data)
	}
}

// withDefaults is the spec as Bootstrap and Scaffold both see it.
func withDefaults(spec ScaffoldSpec) ScaffoldSpec {
	spec.applyDefaults()
	return spec
}

// TestBootstrapInitCarriesThePrefix: the prefix bd refuses through `config
// set` must arrive through `bd init --prefix`, or it arrives nowhere.
func TestBootstrapInitCarriesThePrefix(t *testing.T) {
	spec := withDefaults(testSpec())
	for _, step := range BootstrapPlan(spec) {
		if step.Argv[0] == "bd" && step.Argv[1] == "init" {
			if !strings.Contains(step.String(), "--prefix "+spec.Prefix) {
				t.Errorf("bd init does not carry the prefix: %s", step)
			}
			return
		}
	}
	t.Fatal("plan has no bd init step")
}
