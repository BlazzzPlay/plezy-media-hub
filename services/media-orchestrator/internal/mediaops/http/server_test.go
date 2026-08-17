package http

import (
	"context"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/BlazzzPlay/plezy-media-hub/services/media-orchestrator/internal/mediaops/db"
)

type fakeStore struct {
	pingErr   error
	items     []db.Content
	plans     []db.ReorganizationPlan
	completed bool
}

func (f fakeStore) Ping(context.Context) error { return f.pingErr }
func (f fakeStore) ListContents(context.Context, string, int) ([]db.Content, error) {
	return f.items, nil
}
func (f fakeStore) CreateContent(_ context.Context, c db.Content) (db.Content, error) {
	c.ID = 1
	return c, nil
}
func (f fakeStore) ListCandidates(context.Context, string, int) ([]db.Candidate, error) {
	return []db.Candidate{}, nil
}
func (f fakeStore) ApproveCandidate(context.Context, int64) error { return nil }
func (f fakeStore) RejectCandidate(context.Context, int64) error  { return nil }
func (f fakeStore) ListReorganizationPlans(context.Context, string, int) ([]db.ReorganizationPlan, error) {
	return f.plans, nil
}
func (f fakeStore) LoadReorganizationPlanByID(_ context.Context, id int64) (db.ReorganizationPlan, error) {
	for _, p := range f.plans {
		if p.ID == id {
			return p, nil
		}
	}
	return db.ReorganizationPlan{}, os.ErrNotExist
}
func (f fakeStore) CompleteReorganizationPlan(context.Context, string, string, string, int64) error {
	f.completed = true
	return nil
}
func (f fakeStore) FailReorganizationPlan(context.Context, string, string) error { return nil }
func (f fakeStore) ListJobs(context.Context, string, int) ([]db.Job, error)      { return []db.Job{}, nil }

func TestHealthAndAuth(t *testing.T) {
	s := New(fakeStore{}, "secret")
	ts := httptest.NewServer(s.Handler())
	defer ts.Close()
	r, err := ts.Client().Get(ts.URL + "/health")
	if err != nil {
		t.Fatal(err)
	}
	if r.StatusCode != 200 {
		t.Fatalf("health status=%d", r.StatusCode)
	}
	r, err = ts.Client().Get(ts.URL + "/api/v1/contents")
	if err != nil {
		t.Fatal(err)
	}
	if r.StatusCode != 401 {
		t.Fatalf("unauth status=%d", r.StatusCode)
	}
}

func TestMonitorDiskRequiresAuthAndReturnsMetrics(t *testing.T) {
	s := NewWithMediaRoot(fakeStore{}, "secret", ".")
	unauth := httptest.NewRequest("GET", "/api/v1/monitor/disk", nil)
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, unauth)
	if w.Code != 401 {
		t.Fatalf("unauth monitor status=%d", w.Code)
	}
	req := httptest.NewRequest("GET", "/api/v1/monitor/disk", nil)
	req.Header.Set("Authorization", "Bearer secret")
	w = httptest.NewRecorder()
	s.Handler().ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("monitor status=%d body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "available_bytes") || !strings.Contains(w.Body.String(), "total_bytes") {
		t.Fatalf("monitor response missing disk metrics: %s", w.Body.String())
	}
}
func TestCreateContentValidatesAndCreates(t *testing.T) {
	s := New(fakeStore{}, "secret")
	req := httptest.NewRequest("POST", "/api/v1/contents", strings.NewReader(`{"category":"01-Movies","kind":"movie","title":"Heat","year":1995}`))
	req.Header.Set("Authorization", "Bearer secret")
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, req)
	if w.Code != 201 {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	bad := httptest.NewRequest("POST", "/api/v1/contents", strings.NewReader(`{"title":"Heat"}`))
	bad.Header.Set("Authorization", "Bearer secret")
	w = httptest.NewRecorder()
	s.Handler().ServeHTTP(w, bad)
	if w.Code != 400 {
		t.Fatalf("invalid status=%d", w.Code)
	}
}

func TestCandidatesReviewEndpoints(t *testing.T) {
	s := New(fakeStore{}, "secret")
	req := httptest.NewRequest("GET", "/api/v1/candidates?status=pending", nil)
	req.Header.Set("Authorization", "Bearer secret")
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("list status=%d", w.Code)
	}
	req = httptest.NewRequest("POST", "/api/v1/candidates/7/approve", nil)
	req.Header.Set("Authorization", "Bearer secret")
	w = httptest.NewRecorder()
	s.Handler().ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("approve status=%d", w.Code)
	}
}

func TestPlansRequireExplicitConfirmation(t *testing.T) {
	source := filepath.Join(t.TempDir(), "source.mkv")
	if err := os.WriteFile(source, []byte("api-test"), 0600); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(t.TempDir(), "target.mkv")
	s := New(fakeStore{plans: []db.ReorganizationPlan{{ID: 8, SourcePath: source, TargetPath: target, Action: "planned"}}}, "secret")
	req := httptest.NewRequest("POST", "/api/v1/reorganization-plans/8/execute", strings.NewReader(`{"confirm":false}`))
	req.Header.Set("Authorization", "Bearer secret")
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, req)
	if w.Code != 400 {
		t.Fatalf("without confirmation status=%d", w.Code)
	}
}

