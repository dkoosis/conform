package checks_test

import (
	"strings"
	"testing"

	"github.com/dkoosis/conform-to-sdlc/internal/checks"
)

// TestCheckVocabulary_Missing: no .claude/rules/vocabulary.md is a finding,
// and the message names the path.
func TestCheckVocabulary_Missing(t *testing.T) {
	t.Parallel()

	files := goodRepo()
	delete(files, checks.VocabularyFile)
	dir := writeRepo(t, files)

	f := findingFor(t, checks.Run(dir), checks.RuleVocabulary)
	if f.File != checks.VocabularyFile {
		t.Errorf("File = %q, want %q", f.File, checks.VocabularyFile)
	}
	if !strings.Contains(f.Msg, checks.VocabularyFile) {
		t.Errorf("Msg = %q, should name the path %q", f.Msg, checks.VocabularyFile)
	}
}

// TestCheckVocabulary_EmptyFilePasses: the rule checks presence only — an
// empty file clears it, since the check never judges content.
func TestCheckVocabulary_EmptyFilePasses(t *testing.T) {
	t.Parallel()

	files := goodRepo()
	files[checks.VocabularyFile] = ""
	dir := writeRepo(t, files)

	for _, f := range checks.Run(dir) {
		if f.Rule == checks.RuleVocabulary {
			t.Fatalf("unexpected finding: %v", f)
		}
	}
}
