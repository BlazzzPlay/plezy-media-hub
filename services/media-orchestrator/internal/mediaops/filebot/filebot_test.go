package filebot

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/BlazzzPlay/plezy-media-hub/services/media-orchestrator/internal/mediaops/db"
)

type fakeExecutor struct{ command Command }

func (f *fakeExecutor) Run(_ context.Context, c Command) ([]byte, error) {
	f.command = c
	return []byte("dry-run"), nil
}
func TestBuildRenameDryRun(t *testing.T) {
	cmd := BuildRenameCommand("filebot.exe", `C:\in\movie.mkv`, "TheMovieDB", "{plex}", true)
	got := strings.Join(cmd.Args, " ")
	if !strings.Contains(got, "--action test") || strings.Contains(got, "--action move") {
		t.Fatalf("args=%v", cmd.Args)
	}
}
func TestPlexMatchUsesExplicitFields(t *testing.T) {
	year := 2020
	data, err := GeneratePlexMatch(PlexMatch{Title: "Radio Garka", Year: &year, Season: &year, Episode: &year, AirDate: "2020-01-03"})
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, want := range []string{"Title: Radio Garka", "Year: 2020", "Season: 2020", "AirDate: 2020-01-03"} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing %q in %s", want, text)
		}
	}
}

func TestSeasonPlexMatchUsesYearAsSeason(t *testing.T) {
	year := 2020
	data, err := GenerateSeasonPlexMatch(SeasonPlexMatch{Title: "Radio Garka", Year: &year, Season: 2020})
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, want := range []string{"Title: Radio Garka", "Year: 2020", "Season: 2020"} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing %q in %s", want, text)
		}
	}
}

func TestFormatEpisodeNamePrioritizesDateAndArtist(t *testing.T) {
	got, err := FormatEpisodeName(EpisodeName{Title: "Festival noche 1", AirDate: "2020-02-21", Artist: "Humorista", Season: 2020, Episode: 1, Extension: "mkv"})
	if err != nil {
		t.Fatal(err)
	}
	if got != "2020-02-21 - Humorista - Festival noche 1.mkv" {
		t.Fatalf("got %q", got)
	}
}
func TestMoveRequiresExplicitFlag(t *testing.T) {
	f := &fakeExecutor{}
	t.Setenv("PRA_ALLOW_FILEBOT_MOVE", "")
	if _, err := Rename(context.Background(), f, "movie", "TheMovieDB", "{plex}", false); err == nil {
		t.Fatal("move should be blocked")
	}
}

func TestDryRunAcceptsFileBotTestPreview(t *testing.T) {
	f := &fakeExecutor{}
	f.command = Command{}
	f2 := previewExecutor{}
	if _, err := Rename(context.Background(), f2, "movie", "TheMovieDB", "{plex}", true); err != nil {
		t.Fatal(err)
	}
}

type metadataExecutor struct{}

func (metadataExecutor) Run(context.Context, Command) ([]byte, error) {
	return []byte("931285|Mortal Kombat II|2026|[Action, Fantasy, Adventure]|Simon McQuoid|[Karl Urban, Adeline Rudolph]|7.9|116|US|eng|R|[]|\n"), nil
}
func TestQueryMetadataParsesFileBotFields(t *testing.T) {
	items, err := QueryMetadata(context.Background(), metadataExecutor{}, "TheMovieDB", "Mortal Kombat II")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].ID != "931285" || items[0].Year == nil || *items[0].Year != 2026 || items[0].Director != "Simon McQuoid" || len(items[0].Actors) != 2 {
		t.Fatalf("items=%#v", items)
	}
}

type invalidRatingMetadataExecutor struct{}

func (invalidRatingMetadataExecutor) Run(context.Context, Command) ([]byte, error) {
	return []byte("433631|El caballero de los siete reinos|2026|[Fantasy]|Owen Harris|[Peter Claffey]|373782|40|usa|eng|TV-MA|[]|\n"), nil
}

func TestQueryMetadataIgnoresFileBotNonRatingValues(t *testing.T) {
	items, err := QueryMetadata(context.Background(), invalidRatingMetadataExecutor{}, "TheTVDB", "El caballero de los siete reinos")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Rating != 0 {
		t.Fatalf("items=%#v", items)
	}
}

func TestRealFileBotCandidatePersistenceWhenConfigured(t *testing.T) {
	path := os.Getenv("PRA_TEST_MEDIA_FILE")
	databaseURL := os.Getenv("PRA_DATABASE_URL")
	if path == "" || databaseURL == "" {
		t.Skip("prueba real requiere PRA_TEST_MEDIA_FILE y PRA_DATABASE_URL")
	}
	query := NormalizeQuery(strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)))
	items, err := QueryMetadata(context.Background(), OSExecutor{}, "TheMovieDB", query)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) == 0 {
		t.Fatal("FileBot no devolvió candidatos")
	}
	store, err := db.Open(context.Background(), databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	item := items[0]
	t.Logf("FileBot candidate: id=%s title=%s rating=%.2f runtime=%d", item.ID, item.Title, item.Rating, item.Runtime)
	if err := store.SaveEnrichedCandidate(context.Background(), path, db.EnrichedCandidate{Provider: item.Provider, ExternalID: item.ID, Title: item.Title, Year: item.Year, Genres: item.Genres, Director: item.Director, Actors: item.Actors, Rating: item.Rating, RuntimeMinutes: item.Runtime, Country: item.Country, Language: item.Language, Certification: item.Certification, Tags: item.Tags, Edition: item.Edition}); err != nil {
		t.Fatal(err)
	}
}

type previewExecutor struct{}

func (previewExecutor) Run(context.Context, Command) ([]byte, error) {
	return []byte("[TEST] from movie to target"), fmt.Errorf("exit status 1")
}

