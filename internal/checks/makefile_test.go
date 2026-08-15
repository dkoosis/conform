package checks_test

import (
	"strings"
	"testing"

	"github.com/dkoosis/conform/internal/checks"
	"github.com/dkoosis/conform/internal/values"
)

func TestMakefile_Violations(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		makefile string
		profile  values.Profile
		wantRule string
		wantMsg  string
	}{
		{
			name:     "missing deploy verb on tool profile",
			makefile: strings.Replace(goodMakefile, "deploy: build ## Install locally\n", "", 1),
			profile:  values.ProfileTool,
			wantRule: checks.RuleMakefileVerb,
			wantMsg:  `"deploy" missing`,
		},
		{
			name:     "deploy verb on lib profile",
			makefile: goodMakefile,
			profile:  values.ProfileLib,
			wantRule: checks.RuleMakefileVerb,
			wantMsg:  "lib profile carries a deploy verb",
		},
		{
			name:     "check does not compose lint",
			makefile: strings.Replace(goodMakefile, "check: vet lint test build", "check: vet test build", 1),
			profile:  values.ProfileTool,
			wantRule: checks.RuleMakefileVerb,
			wantMsg:  "lint",
		},
		{
			name:     "audit skips check",
			makefile: strings.Replace(goodMakefile, "audit: check race", "audit: race", 1),
			profile:  values.ProfileTool,
			wantRule: checks.RuleMakefileVerb,
			wantMsg:  "audit does not run check",
		},
		{
			name:     "undocumented target",
			makefile: goodMakefile + "fuzz:\n\tgo test -fuzz .\n",
			profile:  values.ProfileTool,
			wantRule: checks.RuleMakefileDocs,
			wantMsg:  "fuzz",
		},
		{
			name:     "no Makefile at all",
			makefile: "",
			profile:  values.ProfileTool,
			wantRule: checks.RuleMakefileVerb,
			wantMsg:  "no top-level Makefile",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			files := map[string]string{}
			if tt.makefile != "" {
				files["Makefile"] = tt.makefile
			}
			dir := writeRepo(t, files)

			findings := checks.CheckMakefile(dir, tt.profile)
			found := false
			for _, f := range findings {
				if f.Rule == tt.wantRule && strings.Contains(f.Msg, tt.wantMsg) {
					found = true
					if f.Repair == "" {
						t.Error("finding carries no repair command")
					}
				}
			}
			if !found {
				t.Errorf("no %s finding containing %q in %v", tt.wantRule, tt.wantMsg, findings)
			}
		})
	}
}

// TestMakefile_Conforming: the reference shape is clean under both the tool
// contract and (deploy removed) the lib contract, and variable assignments
// are not mistaken for targets.
func TestMakefile_Conforming(t *testing.T) {
	t.Parallel()

	dir := writeRepo(t, map[string]string{"Makefile": goodMakefile})
	if findings := checks.CheckMakefile(dir, values.ProfileTool); len(findings) != 0 {
		t.Errorf("conforming tool Makefile produced findings: %v", findings)
	}

	lib := strings.Replace(goodMakefile, "deploy: build ## Install locally\n", "", 1)
	dir = writeRepo(t, map[string]string{"Makefile": lib})
	if findings := checks.CheckMakefile(dir, values.ProfileLib); len(findings) != 0 {
		t.Errorf("conforming lib Makefile produced findings: %v", findings)
	}
}
