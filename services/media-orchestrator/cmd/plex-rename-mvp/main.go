package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/BlazzzPlay/plezy-media-hub/services/media-orchestrator/internal/mediaops/config"
	"github.com/BlazzzPlay/plezy-media-hub/services/media-orchestrator/internal/mediaops/db"
	mvphttp "github.com/BlazzzPlay/plezy-media-hub/services/media-orchestrator/internal/mediaops/http"
)

func main() {
	c, err := config.Load()
	if err != nil {
		slog.Error("configuration failed", "error", err)
		os.Exit(1)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	store, err := db.Open(ctx, c.DatabaseURL)
	if err != nil {
		slog.Error("database failed", "error", err)
		os.Exit(1)
	}
	defer store.Close()
	srv := &http.Server{Addr: ":" + itoa(c.Port), Handler: mvphttp.NewWithMediaRoot(store, c.APIKey, c.StagingRoot).Handler(), ReadHeaderTimeout: 5 * time.Second}
	slog.Info("media-orchestrator listening", "addr", srv.Addr, "media_config", c.MediaConfig, "staging_root", c.StagingRoot)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("server failed", "error", err)
		os.Exit(1)
	}
}
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	b := make([]byte, 0, 6)
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}
