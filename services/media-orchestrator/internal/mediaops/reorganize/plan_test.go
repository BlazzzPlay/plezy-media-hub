package reorganize

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/BlazzzPlay/plezy-media-hub/services/media-orchestrator/internal/mediaops/naming"
)

func TestBuildMoviePlanNeverMoves(t *testing.T) {
	year := 1995
	parsed := naming.ParsedName{Category: "01-Movies", Kind: "movie", Title: "Heat", Year: &year}
	got := BuildPlan(`C:\Plex\staging`, `C:\Plex\staging\source.mkv`, "tmdb-heat", parsed)
	want := filepath.Join(`C:\Plex\staging`, `01-Movies`, `G-I`, `Heat (1995) {tmdb-heat}`)
	if got.Target != want || got.Action != "planned" {
		t.Fatalf("plan=%#v want target=%s", got, want)
	}
}

func TestExecutePlanMovesAndVerifiesHash(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source.mkv")
	if err := os.WriteFile(source, []byte("verified-media"), 0600); err != nil {
		t.Fatal(err)
	}
	year := 2026
	plan := BuildPlan(root, source, "tmdb-1", naming.ParsedName{Category: "01-Movies", Kind: "movie", Title: "Example", Year: &year})
	plan.Action = "planned"
	result, err := ExecutePlan(context.Background(), plan, "")
	if err != nil {
		t.Fatal(err)
	}
	if result.Target != plan.TargetFile || result.SHA256 == "" || result.Bytes == 0 {
		t.Fatalf("result=%#v", result)
	}
	if _, err := os.Stat(source); !os.IsNotExist(err) {
		t.Fatalf("source still exists: %v", err)
	}
	if _, err := os.Stat(plan.TargetFile); err != nil {
		t.Fatalf("target missing: %v", err)
	}
}

func TestExecutePlanCopiesAndPreservesSource(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source.mkv")
	target := filepath.Join(root, "isolated", "copy.mkv")
	content := []byte("verified-copy-media")
	if err := os.WriteFile(source, content, 0600); err != nil {
		t.Fatal(err)
	}
	result, err := ExecutePlan(context.Background(), Plan{
		Source: source, TargetFile: target, Operation: "copy", Action: "planned",
	}, "")
	if err != nil {
		t.Fatal(err)
	}
	if result.Operation != "copy" || result.Target != target || result.Bytes != int64(len(content)) {
		t.Fatalf("result=%#v", result)
	}
	if _, err := os.Stat(source); err != nil {
		t.Fatalf("source was not preserved: %v", err)
	}
	if got, err := os.ReadFile(target); err != nil || string(got) != string(content) {
		t.Fatalf("target content mismatch: %q, %v", got, err)
	}
}

func TestLetterGroups(t *testing.T) {
	for input, want := range map[string]string{"Avatar": "A-C", "Mortal Kombat": "M-O", "Star Wars": "S-U", "7 Samurai": "0-9", "Élite": "pending-review"} {
		if got := LetterGroup(input); got != want {
			t.Errorf("%q=%q want %q", input, got, want)
		}
	}
}

