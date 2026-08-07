package http

import (
	"context"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"

	"github.com/BlazzzPlay/plezy-media-hub/services/media-orchestrator/internal/mediaops/db"
)

func TestCandidatesEndpointAgainstLocalPostgresWhenConfigured(t *testing.T) {
	url := os.Getenv("PRA_DATABASE_URL")
	key := os.Getenv("PRA_API_KEY")
	if url == "" || key == "" { t.Skip("prueba local requiere PRA_DATABASE_URL y PRA_API_KEY") }
	store, err := db.Open(context.Background(), url); if err != nil { t.Fatal(err) }; defer store.Close()
	req := httptest.NewRequest("GET", "/api/v1/candidates?status=pending&limit=10", nil); req.Header.Set("Authorization", "Bearer "+key)
	recorder := httptest.NewRecorder(); New(store, key).Handler().ServeHTTP(recorder, req)
	if recorder.Code != 200 { t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String()) }
}

func TestApproveCandidateThroughAPIWhenExplicitlyConfigured(t *testing.T) {
	url := os.Getenv("PRA_DATABASE_URL"); key := os.Getenv("PRA_API_KEY"); rawID := os.Getenv("PRA_APPROVE_CANDIDATE_ID")
	if url==""||key==""||rawID==""{t.Skip("aprobación real requiere PRA_DATABASE_URL, PRA_API_KEY y PRA_APPROVE_CANDIDATE_ID")}
	id,err:=strconv.ParseInt(rawID,10,64);if err!=nil{t.Fatal(err)};store,err:=db.Open(context.Background(),url);if err!=nil{t.Fatal(err)};defer store.Close()
	req:=httptest.NewRequest("POST","/api/v1/candidates/"+strconv.FormatInt(id,10)+"/approve",nil);req.Header.Set("Authorization","Bearer "+key);rec:=httptest.NewRecorder();New(store,key).Handler().ServeHTTP(rec,req);if rec.Code!=200{t.Fatalf("status=%d body=%s",rec.Code,rec.Body.String())}
}

