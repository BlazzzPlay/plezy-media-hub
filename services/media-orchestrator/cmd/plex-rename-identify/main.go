package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/BlazzzPlay/plezy-media-hub/services/media-orchestrator/internal/mediaops/config"
	"github.com/BlazzzPlay/plezy-media-hub/services/media-orchestrator/internal/mediaops/db"
	"github.com/BlazzzPlay/plezy-media-hub/services/media-orchestrator/internal/mediaops/filebot"
)

func main() {
	path := flag.String("path", "", "archivo multimedia a identificar")
	query := flag.String("query", "", "consulta FileBot; por defecto usa el nombre del archivo")
	database := flag.String("db", "TheMovieDB", "base FileBot: TheMovieDB, TheTVDB o AniDB")
	persist := flag.Bool("persist", false, "guardar candidatos en PostgreSQL")
	flag.Parse()
	if *path == "" {
		fmt.Fprintln(os.Stderr, "--path es obligatorio")
		os.Exit(2)
	}
	if *query == "" {
		*query = filebot.NormalizeQuery(strings.TrimSuffix(filepath.Base(*path), filepath.Ext(*path)))
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	items, err := filebot.QueryMetadata(ctx, filebot.OSExecutor{}, *database, *query)
	if err != nil {
		slog.Error("FileBot query failed", "error", err)
		os.Exit(1)
	}
	data, _ := json.MarshalIndent(items, "", "  ")
	fmt.Println(string(data))
	if !*persist {
		return
	}
	c, err := config.Load()
	if err != nil {
		slog.Error("configuration failed", "error", err)
		os.Exit(1)
	}
	store, err := db.Open(ctx, c.DatabaseURL)
	if err != nil {
		slog.Error("database failed", "error", err)
		os.Exit(1)
	}
	defer store.Close()
	for _, item := range items {
		if err := store.SaveEnrichedCandidate(ctx, *path, db.EnrichedCandidate{Provider: *database, ExternalID: item.ID, Title: item.Title, Year: item.Year, Genres: item.Genres, Director: item.Director, Actors: item.Actors, Rating: item.Rating, RuntimeMinutes: item.Runtime, Country: item.Country, Language: item.Language, Certification: item.Certification, Tags: item.Tags, Edition: item.Edition}); err != nil {
			slog.Error("candidate persist failed", "error", err)
			os.Exit(1)
		}
	}
}

