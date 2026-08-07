package db

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestOpenAndMigrateConfiguredPostgres(t *testing.T) {
	url := os.Getenv("PRA_DATABASE_URL")
	if url == "" {
		t.Skip("PRA_DATABASE_URL no está configurada; prueba de integración local")
	}
	store, err := Open(context.Background(), url)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if err := store.Ping(context.Background()); err != nil {
		t.Fatal(err)
	}
	items, err := store.ListContents(context.Background(), "", 1)
	if err != nil {
		t.Fatal(err)
	}
	if items == nil {
		t.Fatal("ListContents returned nil; expected an empty slice")
	}
	file := InventoryFile{Path: "mvp-test/sample.mkv", RelativePath: "sample.mkv", Size: 12, ModifiedAt: time.Now().UTC(), SHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", Container: "matroska", VideoCodec: "hevc", Width: 1920, Height: 1080}
	if err := store.SaveInventory(context.Background(), file); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveInventory(context.Background(), file); err != nil {
		t.Fatal(err)
	}
}

