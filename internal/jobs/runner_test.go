package jobs

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/zoobz-io/barbara/database/models"
)

// mockStore is an in-memory Store for tests.
type mockStore struct {
	mu      sync.Mutex
	pending []*models.Job
	done    []string
	failed  map[string]string
	claimed bool
}

func newMockStore(jobs ...*models.Job) *mockStore {
	return &mockStore{pending: jobs, failed: map[string]string{}}
}

func (m *mockStore) ClaimPending(_ context.Context, limit int) ([]*models.Job, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.claimed {
		return nil, nil // only hand out the batch once
	}
	m.claimed = true
	if limit < len(m.pending) {
		return m.pending[:limit], nil
	}
	return m.pending, nil
}

func (m *mockStore) MarkDone(_ context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.done = append(m.done, id)
	return nil
}

func (m *mockStore) MarkFailed(_ context.Context, id, errMsg string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.failed[id] = errMsg
	return nil
}

func (m *mockStore) snapshot() ([]string, map[string]string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	d := append([]string(nil), m.done...)
	f := map[string]string{}
	for k, v := range m.failed {
		f[k] = v
	}
	return d, f
}

// runOnce starts a runner, waits until it has processed the batch, then stops.
func runOnce(t *testing.T, store *mockStore, w IndexWriter, want int) {
	t.Helper()
	r := NewRunner(store, NewPipeline(w, 2, time.Millisecond), 5*time.Millisecond, 10)
	ctx, cancel := context.WithCancel(context.Background())
	r.Start(ctx)

	deadline := time.After(2 * time.Second)
	for {
		d, f := store.snapshot()
		if len(d)+len(f) >= want {
			break
		}
		select {
		case <-deadline:
			cancel()
			r.Stop()
			t.Fatalf("timed out: processed %d/%d", len(d)+len(f), want)
		case <-time.After(2 * time.Millisecond):
		}
	}
	cancel()
	r.Stop()
}

func TestRunner_MarksDoneOnSuccess(t *testing.T) {
	store := newMockStore(
		&models.Job{ID: "j1", DocumentID: "d1", Operation: models.JobIndex},
		&models.Job{ID: "j2", DocumentID: "d2", Operation: models.JobDelete},
	)
	runOnce(t, store, &mockWriter{}, 2)

	done, failed := store.snapshot()
	if len(done) != 2 || len(failed) != 0 {
		t.Errorf("done=%v failed=%v, want both done", done, failed)
	}
}

func TestRunner_MarksFailedOnTerminalError(t *testing.T) {
	store := newMockStore(&models.Job{ID: "j1", DocumentID: "d1", Operation: models.JobIndex})
	w := &mockWriter{index: func(_ context.Context, _ string, _ []byte) error {
		return errors.New("permanent")
	}}
	runOnce(t, store, w, 1)

	done, failed := store.snapshot()
	if len(done) != 0 {
		t.Errorf("expected none done, got %v", done)
	}
	if msg, ok := failed["j1"]; !ok || msg == "" {
		t.Errorf("expected j1 marked failed with a message, got %v", failed)
	}
}

func TestRunner_StopIsClean(t *testing.T) {
	// A runner with no work should start and, once its context is cancelled,
	// stop promptly rather than block.
	r := NewRunner(newMockStore(), NewPipeline(&mockWriter{}, 1, time.Millisecond), time.Millisecond, 10)
	ctx, cancel := context.WithCancel(context.Background())
	r.Start(ctx)
	time.Sleep(5 * time.Millisecond)
	cancel()

	stopped := make(chan struct{})
	go func() { r.Stop(); close(stopped) }()
	select {
	case <-stopped:
	case <-time.After(time.Second):
		t.Fatal("Stop did not return after context cancellation")
	}
}
