package reorganize

import (
	"context"
	"os"
	"testing"

	"github.com/BlazzzPlay/plezy-media-hub/services/media-orchestrator/internal/mediaops/db"
	"github.com/BlazzzPlay/plezy-media-hub/services/media-orchestrator/internal/mediaops/naming"
)

func TestPersistApprovedPlanWhenConfigured(t *testing.T) {
	path := os.Getenv("PRA_PLAN_TEST_PATH")
	url := os.Getenv("PRA_DATABASE_URL")
	root := os.Getenv("PRA_STAGING_ROOT")
	if path == "" || url == "" || root == "" {
		t.Skip("plan integration requires PRA_PLAN_TEST_PATH, PRA_DATABASE_URL and PRA_STAGING_ROOT")
	}
	store, err := db.Open(context.Background(), url)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	input, err := store.LoadPlanInput(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	parsed := naming.ParsedName{Category: input.Category, Kind: input.Kind, Title: input.Title, Year: input.Year}
	plan := BuildPlan(root, input.Path, input.ExternalID, parsed)
	if plan.Action != "planned" {
		t.Fatalf("plan=%#v", plan)
	}
	if err := store.SaveReorganizationPlan(context.Background(), db.ReorganizationPlan{SourcePath: plan.Source, TargetPath: plan.TargetFile, Action: plan.Action, Reason: plan.Reason}); err != nil {
		t.Fatal(err)
	}
}

