package github

import (
	"testing"

	gh "github.com/google/go-github/v66/github"
)

func TestCategorizeFiles(t *testing.T) {
	files := []*gh.CommitFile{
		{Filename: gh.String("pkg/added.go"), Status: gh.String("added")},
		{Filename: gh.String("pkg/modified.go"), Status: gh.String("modified")},
		{Filename: gh.String("pkg/removed.go"), Status: gh.String("removed")},
		{Filename: gh.String("pkg/new_name.go"), Status: gh.String("renamed"), PreviousFilename: gh.String("pkg/old_name.go")},
		{Filename: gh.String("README.md"), Status: gh.String("modified")},
	}

	toAnalyze, deleted := CategorizeFiles(files)

	if len(toAnalyze) != 3 {
		t.Fatalf("expected 3 analyzable files, got %d", len(toAnalyze))
	}
	if len(deleted) != 1 {
		t.Fatalf("expected 1 deleted file, got %d", len(deleted))
	}

	wantAnalyze := map[string]ChangedFile{
		"pkg/added.go":    {Path: "pkg/added.go", Status: "added"},
		"pkg/modified.go": {Path: "pkg/modified.go", Status: "modified"},
		"pkg/new_name.go": {Path: "pkg/new_name.go", Status: "renamed", PreviousPath: "pkg/old_name.go"},
	}
	for _, file := range toAnalyze {
		want, ok := wantAnalyze[file.Path]
		if !ok {
			t.Fatalf("unexpected analyzable file: %+v", file)
		}
		if file.Status != want.Status {
			t.Fatalf("file %s: expected status %q, got %q", file.Path, want.Status, file.Status)
		}
		if file.PreviousPath != want.PreviousPath {
			t.Fatalf("file %s: expected previous path %q, got %q", file.Path, want.PreviousPath, file.PreviousPath)
		}
		delete(wantAnalyze, file.Path)
	}
	if len(wantAnalyze) != 0 {
		t.Fatalf("missing analyzable files: %v", wantAnalyze)
	}

	if deleted[0].Path != "pkg/removed.go" {
		t.Fatalf("expected deleted file pkg/removed.go, got %s", deleted[0].Path)
	}
	if deleted[0].Status != "removed" {
		t.Fatalf("expected deleted status removed, got %q", deleted[0].Status)
	}
	if deleted[0].PreviousPath != "" {
		t.Fatalf("expected deleted file previous path empty, got %q", deleted[0].PreviousPath)
	}
}

func TestCategorizeFiles_PatchPopulated(t *testing.T) {
	files := []*gh.CommitFile{{Filename: gh.String("pkg/patch.go"), Status: gh.String("modified"), Patch: gh.String("@@ -1 +1 @@\n-old\n+new")}}
	toAnalyze, deleted := CategorizeFiles(files)
	if len(deleted) != 0 {
		t.Fatalf("expected no deleted files, got %d", len(deleted))
	}
	if len(toAnalyze) != 1 {
		t.Fatalf("expected 1 analyzable file, got %d", len(toAnalyze))
	}
	if toAnalyze[0].Patch != "@@ -1 +1 @@\n-old\n+new" {
		t.Fatalf("expected patch to be preserved, got %q", toAnalyze[0].Patch)
	}
}
