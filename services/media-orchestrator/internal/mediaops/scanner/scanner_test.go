package scanner

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/BlazzzPlay/plezy-media-hub/services/media-orchestrator/internal/mediaops/db"
)

func TestScanCopiedMoviesWhenConfigured(t *testing.T) {
	root := os.Getenv("PRA_TEST_MEDIA_ROOT")
	if root == "" {
		t.Skip("PRA_TEST_MEDIA_ROOT no está configurado")
	}
	items, err := Scan(context.Background(), root, FFprobe, 7)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 7 {
		t.Fatalf("found %d movies, want 7", len(items))
	}
	for _, item := range items {
		if item.SHA256 == "" || item.Container == "" || item.Width == 0 || item.Height == 0 {
			t.Fatalf("incomplete technical inventory: %#v", item)
		}
	}
	if databaseURL := os.Getenv("PRA_DATABASE_URL"); databaseURL != "" {
		store, err := db.Open(context.Background(), databaseURL)
		if err != nil {
			t.Fatal(err)
		}
		defer store.Close()
		for _, item := range items {
			err = store.SaveInventory(context.Background(), db.InventoryFile{Path: item.Path, RelativePath: item.RelativePath, Size: item.Size, ModifiedAt: item.ModifiedAt, SHA256: item.SHA256, Container: item.Container, VideoCodec: item.VideoCodec, Width: item.Width, Height: item.Height, Duration: item.Duration, HDR: item.HDR, Bitrate: item.Bitrate, Parsed: item.Parsed})
			if err != nil {
				t.Fatal(err)
			}
			if item.Parsed.Title != "" {
				if err := store.SaveParsedCandidate(context.Background(), item.Path, item.Parsed.Title, item.Parsed.Year); err != nil {
					t.Fatal(err)
				}
			}
		}
	}
}

func TestProbeCopiedMovieWhenConfigured(t *testing.T) {
	root := os.Getenv("PRA_TEST_MEDIA_ROOT")
	if root == "" {
		t.Skip("PRA_TEST_MEDIA_ROOT no está configurado")
	}
	var movie string
	_ = filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err == nil && !entry.IsDir() && filepath.Ext(path) == ".mkv" && movie == "" {
			movie = path
		}
		return nil
	})
	if movie == "" {
		t.Fatal("no MKV found")
	}
	metadata, err := FFprobe(context.Background(), movie)
	if err != nil {
		t.Fatal(err)
	}
	if metadata.Container == "" || metadata.Width == 0 || metadata.Height == 0 || metadata.VideoCodec == "" {
		t.Fatalf("incomplete metadata: %#v", metadata)
	}
}

func TestScanHashesOnlyMediaAndSupportsLimit(t *testing.T) {
	root := t.TempDir()
	media := filepath.Join(root, "clip.mkv")
	if err := os.WriteFile(media, []byte("sample-media"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "notes.txt"), []byte("ignore"), 0600); err != nil {
		t.Fatal(err)
	}
	probe := func(context.Context, string) (TechnicalMetadata, error) {
		return TechnicalMetadata{Container: "matroska", VideoCodec: "hevc", Width: 1920, Height: 1080}, nil
	}
	items, err := Scan(context.Background(), root, probe, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].RelativePath != "clip.mkv" {
		t.Fatalf("unexpected inventory: %#v", items)
	}
	if items[0].SHA256 == "" || items[0].Size != int64(len("sample-media")) {
		t.Fatalf("hash/size missing: %#v", items[0])
	}
}

