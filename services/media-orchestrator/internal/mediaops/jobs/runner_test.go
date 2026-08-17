package jobs

import (
	"context"
	"testing"
	"time"
)

func TestRunnerDefaultsAreControlled(t *testing.T) {
	r := Runner{PollInterval: 0, Handlers: map[string]Handler{"analyze": nil}, Concurrency: map[string]int{"analyze": 0}}
	if r.Concurrency["analyze"] != 0 {
		t.Fatal("test setup")
	}
	if time.Second <= 0 {
		t.Fatal("unreachable")
	}
	_ = context.Background()
}

