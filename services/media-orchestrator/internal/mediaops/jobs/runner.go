package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/BlazzzPlay/plezy-media-hub/services/media-orchestrator/internal/mediaops/db"
)

type Handler func(context.Context, db.Job) (json.RawMessage, error)
type Runner struct {
	Store        *db.Store
	WorkerPrefix string
	PollInterval time.Duration
	Handlers     map[string]Handler
	Concurrency  map[string]int
}

var DefaultConcurrency = map[string]int{"download": 1, "analyze": 2, "remux": 1, "nas_copy": 1, "metadata": 1, "filebot_rename": 1}

// Run starts durable workers. Each job type has its own concurrency ceiling.
func (r Runner) Run(ctx context.Context) error {
	if r.Store == nil {
		return fmt.Errorf("job store is required")
	}
	if r.PollInterval <= 0 {
		r.PollInterval = 2 * time.Second
	}
	if r.WorkerPrefix == "" {
		r.WorkerPrefix = "mvp"
	}
	if err := r.Store.RecoverStaleJobs(ctx, time.Minute); err != nil {
		return fmt.Errorf("recover stale jobs: %w", err)
	}
	for jobType, handler := range r.Handlers {
		n := r.Concurrency[jobType]
		if n < 1 {
			n = 1
		}
		for i := 0; i < n; i++ {
			go r.worker(ctx, fmt.Sprintf("%s-%s-%d", r.WorkerPrefix, jobType, i+1), jobType, handler)
		}
	}
	<-ctx.Done()
	return ctx.Err()
}

func (r Runner) worker(ctx context.Context, workerID, jobType string, handler Handler) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		job, err := r.Store.ClaimJob(ctx, workerID, jobType)
		if err != nil {
			slog.Error("claim job failed", "worker", workerID, "error", err)
			r.wait(ctx)
			continue
		}
		if job == nil {
			r.wait(ctx)
			continue
		}
		checkpoint := job.Checkpoint
		done := make(chan struct{})
		go r.heartbeat(ctx, workerID, *job, checkpoint, done)
		checkpoint, err = handler(ctx, *job)
		close(done)
		if err != nil {
			if e := r.Store.FailJob(context.Background(), job.ID, workerID, err); e != nil {
				slog.Error("fail job update failed", "job", job.ID, "error", e)
			}
		} else if e := r.Store.CompleteJob(context.Background(), job.ID, workerID, checkpoint); e != nil {
			slog.Error("complete job update failed", "job", job.ID, "error", e)
		}
	}
}
func (r Runner) heartbeat(ctx context.Context, workerID string, job db.Job, checkpoint json.RawMessage, done <-chan struct{}) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-done:
			return
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := r.Store.HeartbeatJob(context.Background(), job.ID, workerID, checkpoint); err != nil {
				slog.Warn("job heartbeat failed", "job", job.ID, "error", err)
			}
		}
	}
}
func (r Runner) wait(ctx context.Context) {
	timer := time.NewTimer(r.PollInterval)
	defer timer.Stop()
	select {
	case <-ctx.Done():
	case <-timer.C:
	}
}

