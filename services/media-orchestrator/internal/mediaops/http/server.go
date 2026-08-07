package http

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/BlazzzPlay/plezy-media-hub/services/media-orchestrator/internal/mediaops/db"
	"github.com/BlazzzPlay/plezy-media-hub/services/media-orchestrator/internal/mediaops/reorganize"
)

type Store interface {
	Ping(context.Context) error
	ListContents(context.Context, string, int) ([]db.Content, error)
	CreateContent(context.Context, db.Content) (db.Content, error)
	ListCandidates(context.Context, string, int) ([]db.Candidate, error)
	ApproveCandidate(context.Context, int64) error
	RejectCandidate(context.Context, int64) error
	ListReorganizationPlans(context.Context, string, int) ([]db.ReorganizationPlan, error)
	LoadReorganizationPlanByID(context.Context, int64) (db.ReorganizationPlan, error)
	CompleteReorganizationPlan(context.Context, string, string, string, int64) error
	FailReorganizationPlan(context.Context, string, string) error
	ListJobs(context.Context, string, int) ([]db.Job, error)
}

type Server struct {
	store     Store
	apiKey    string
	mediaRoot string
}

func New(store Store, apiKey string) *Server { return NewWithMediaRoot(store, apiKey, "") }

func NewWithMediaRoot(store Store, apiKey, mediaRoot string) *Server {
	return &Server{store: store, apiKey: apiKey, mediaRoot: filepath.Clean(mediaRoot)}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", s.health)
	mux.HandleFunc("GET /api/v1/monitor/disk", s.monitorDisk)
	mux.HandleFunc("GET /api/v1/contents", s.listContents)
	mux.HandleFunc("POST /api/v1/contents", s.createContent)
	mux.HandleFunc("GET /api/v1/candidates", s.listCandidates)
	mux.HandleFunc("POST /api/v1/candidates/{id}/approve", s.approveCandidate)
	mux.HandleFunc("POST /api/v1/candidates/{id}/reject", s.rejectCandidate)
	mux.HandleFunc("GET /api/v1/reorganization-plans", s.listPlans)
	mux.HandleFunc("POST /api/v1/reorganization-plans/{id}/execute", s.executePlan)
	mux.HandleFunc("GET /api/v1/workflow-jobs", s.listJobs)
	return s.auth(mux)
}

func (s *Server) listJobs(w http.ResponseWriter, r *http.Request) {
	limit := 100
	if raw := r.URL.Query().Get("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 || n > 500 {
			writeError(w, 400, "limit must be 1..500")
			return
		}
		limit = n
	}
	jobs, err := s.store.ListJobs(r.Context(), r.URL.Query().Get("status"), limit)
	if err != nil {
		writeError(w, 500, "could not list workflow jobs")
		return
	}
	if jobs == nil {
		jobs = []db.Job{}
	}
	writeJSON(w, 200, map[string]any{"items": jobs, "limit": limit})
}

func (s *Server) listPlans(w http.ResponseWriter, r *http.Request) {
	limit := 100
	if raw := r.URL.Query().Get("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 || n > 500 {
			writeError(w, 400, "limit must be 1..500")
			return
		}
		limit = n
	}
	plans, err := s.store.ListReorganizationPlans(r.Context(), r.URL.Query().Get("action"), limit)
	if err != nil {
		writeError(w, 500, "could not list reorganization plans")
		return
	}
	if plans == nil {
		plans = []db.ReorganizationPlan{}
	}
	writeJSON(w, 200, map[string]any{"items": plans, "limit": limit})
}

func (s *Server) executePlan(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id < 1 {
		writeError(w, 400, "invalid plan id")
		return
	}
	var body struct {
		Confirm bool `json:"confirm"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || !body.Confirm {
		writeError(w, 400, "confirm=true is required")
		return
	}
	stored, err := s.store.LoadReorganizationPlanByID(r.Context(), id)
	if err != nil {
		if err == sql.ErrNoRows {
			writeError(w, 404, "plan not found")
			return
		}
		writeError(w, 500, "could not load plan")
		return
	}
	result, err := reorganize.ExecutePlan(r.Context(), reorganize.Plan{Source: stored.SourcePath, TargetFile: stored.TargetPath, Operation: stored.Operation, Action: stored.Action}, stored.SHA256)
	if err != nil {
		_ = s.store.FailReorganizationPlan(context.Background(), stored.SourcePath, err.Error())
		writeError(w, 409, err.Error())
		return
	}
	if err := s.store.CompleteReorganizationPlan(r.Context(), result.Source, result.Target, result.SHA256, result.Bytes); err != nil {
		writeError(w, 500, "file operation succeeded but plan state could not be saved")
		return
	}
	writeJSON(w, 200, result)
}

func (s *Server) auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" || s.apiKey == "" {
			next.ServeHTTP(w, r)
			return
		}
		if r.Header.Get("Authorization") != "Bearer "+s.apiKey {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		next.ServeHTTP(w, r)
	})
}
func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	if err := s.store.Ping(r.Context()); err != nil {
		writeError(w, 503, "database unavailable")
		return
	}
	writeJSON(w, 200, map[string]string{"status": "ok", "database": "ok"})
}

func (s *Server) monitorDisk(w http.ResponseWriter, r *http.Request) {
	path := s.mediaRoot
	if path == "" || path == "." {
		path = "."
	}
	available, total := diskStats(path)
	used := uint64(0)
	if total >= available {
		used = total - available
	}
	writeJSON(w, 200, map[string]any{
		"path":            path,
		"available_bytes": available,
		"total_bytes":     total,
		"used_bytes":      used,
		"available_gb":    float64(available) / (1024 * 1024 * 1024),
		"total_gb":        float64(total) / (1024 * 1024 * 1024),
		"used_gb":         float64(used) / (1024 * 1024 * 1024),
	})
}
func (s *Server) listContents(w http.ResponseWriter, r *http.Request) {
	limit := 100
	if raw := r.URL.Query().Get("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 || n > 500 {
			writeError(w, 400, "limit must be 1..500")
			return
		}
		limit = n
	}
	items, err := s.store.ListContents(r.Context(), r.URL.Query().Get("category"), limit)
	if err != nil {
		writeError(w, 500, "could not list contents")
		return
	}
	if items == nil {
		items = []db.Content{}
	}
	writeJSON(w, 200, map[string]any{"items": items, "limit": limit})
}
func (s *Server) createContent(w http.ResponseWriter, r *http.Request) {
	var c db.Content
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		writeError(w, 400, "invalid JSON")
		return
	}
	c.Category = strings.TrimSpace(c.Category)
	c.Kind = strings.TrimSpace(c.Kind)
	c.Title = strings.TrimSpace(c.Title)
	if c.Category == "" || c.Kind == "" || c.Title == "" {
		writeError(w, 400, "category, kind and title are required")
		return
	}
	created, err := s.store.CreateContent(r.Context(), c)
	if err != nil {
		writeError(w, 409, "content could not be created")
		return
	}
	writeJSON(w, 201, created)
}
func (s *Server) listCandidates(w http.ResponseWriter, r *http.Request) {
	limit := 100
	if raw := r.URL.Query().Get("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 || n > 500 {
			writeError(w, 400, "limit must be 1..500")
			return
		}
		limit = n
	}
	items, err := s.store.ListCandidates(r.Context(), r.URL.Query().Get("status"), limit)
	if err != nil {
		writeError(w, 500, "could not list candidates")
		return
	}
	if items == nil {
		items = []db.Candidate{}
	}
	writeJSON(w, 200, map[string]any{"items": items, "limit": limit})
}
func (s *Server) candidateID(r *http.Request) (int64, error) {
	return strconv.ParseInt(r.PathValue("id"), 10, 64)
}
func (s *Server) approveCandidate(w http.ResponseWriter, r *http.Request) {
	id, err := s.candidateID(r)
	if err != nil {
		writeError(w, 400, "invalid candidate id")
		return
	}
	if err = s.store.ApproveCandidate(r.Context(), id); err != nil {
		if err == sql.ErrNoRows {
			writeError(w, 404, "candidate not found")
			return
		}
		writeError(w, 409, "candidate could not be approved")
		return
	}
	writeJSON(w, 200, map[string]any{"id": id, "status": "accepted"})
}
func (s *Server) rejectCandidate(w http.ResponseWriter, r *http.Request) {
	id, err := s.candidateID(r)
	if err != nil {
		writeError(w, 400, "invalid candidate id")
		return
	}
	if err = s.store.RejectCandidate(r.Context(), id); err != nil {
		if err == sql.ErrNoRows {
			writeError(w, 404, "candidate not found or already reviewed")
			return
		}
		writeError(w, 409, "candidate could not be rejected")
		return
	}
	writeJSON(w, 200, map[string]any{"id": id, "status": "rejected"})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

