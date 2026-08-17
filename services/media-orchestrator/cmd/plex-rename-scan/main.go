package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/BlazzzPlay/plezy-media-hub/services/media-orchestrator/internal/mediaops/config"
	"github.com/BlazzzPlay/plezy-media-hub/services/media-orchestrator/internal/mediaops/db"
	"github.com/BlazzzPlay/plezy-media-hub/services/media-orchestrator/internal/mediaops/reorganize"
	"github.com/BlazzzPlay/plezy-media-hub/services/media-orchestrator/internal/mediaops/scanner"
)

func main() {
	root := flag.String("root", "", "directorio de staging a inventariar (obligatorio)")
	maxFiles := flag.Int("max-files", 0, "máximo de archivos; 0 significa sin límite")
	dryRun := flag.Bool("dry-run", true, "mostrar inventario sin persistirlo")
	flag.Parse()
	if *root == "" {
		fmt.Fprintln(os.Stderr, "--root es obligatorio")
		os.Exit(2)
	}
	c, err := config.Load()
	if err != nil {
		slog.Error("configuration failed", "error", err)
		os.Exit(1)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	items, err := scanner.Scan(ctx, *root, nil, *maxFiles)
	if err != nil {
		slog.Error("scan failed", "error", err)
		os.Exit(1)
	}
	if *dryRun {
		for _, item := range items {
			plan := reorganize.BuildPlan(c.StagingRoot, item.Path, "", item.Parsed)
			fmt.Printf("%s\t%s\t%d bytes\t%s\t%s\t%s\t%s\n", item.SHA256, item.RelativePath, item.Size, item.Container, item.VideoCodec, plan.Target, plan.Action)
		}
		return
	}
	openCtx, openCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer openCancel()
	store, err := db.Open(openCtx, c.DatabaseURL)
	if err != nil {
		slog.Error("database failed", "error", err)
		os.Exit(1)
	}
	defer store.Close()
	for _, item := range items {
		err := store.SaveInventory(context.Background(), db.InventoryFile{Path: item.Path, RelativePath: item.RelativePath, Size: item.Size, ModifiedAt: item.ModifiedAt, SHA256: item.SHA256, Container: item.Container, VideoCodec: item.VideoCodec, Width: item.Width, Height: item.Height, Duration: item.Duration, HDR: item.HDR, Bitrate: item.Bitrate, Parsed: item.Parsed})
		if err != nil {
			slog.Error("persist inventory failed", "path", item.Path, "error", err)
			os.Exit(1)
		}
		if item.Parsed.Title != "" {
			if err := store.SaveParsedCandidate(context.Background(), item.Path, item.Parsed.Title, item.Parsed.Year); err != nil {
				slog.Error("persist identity candidate failed", "path", item.Path, "error", err)
				os.Exit(1)
			}
		}
		externalID := ""
		if approved, err := store.ApprovedExternalID(context.Background(), item.Path); err == nil {
			externalID = approved
		}
		plan := reorganize.BuildPlan(c.StagingRoot, item.Path, externalID, item.Parsed)
		if err := store.SaveReorganizationPlan(context.Background(), db.ReorganizationPlan{SourcePath: plan.Source, TargetPath: plan.Target, Action: plan.Action, Reason: plan.Reason}); err != nil {
			slog.Error("persist reorganization plan failed", "path", item.Path, "error", err)
			os.Exit(1)
		}
	}
	fmt.Printf("Inventariados %d archivos.\n", len(items))
}

