package jobs

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/zoobz-io/barbara/database/models"
)

// mockWriter is a function-field IndexWriter for tests.
type mockWriter struct {
	index  func(ctx context.Context, documentID string, payload []byte) error
	delete func(ctx context.Context, documentID string) error
}

func (m *mockWriter) Index(ctx context.Context, documentID string, payload []byte) error {
	if m.index != nil {
		return m.index(ctx, documentID, payload)
	}
	return nil
}

func (m *mockWriter) Delete(ctx context.Context, documentID string) error {
	if m.delete != nil {
		return m.delete(ctx, documentID)
	}
	return nil
}

func TestPipeline_IndexSuccess(t *testing.T) {
	var gotID string
	var gotPayload []byte
	w := &mockWriter{index: func(_ context.Context, id string, payload []byte) error {
		gotID = id
		gotPayload = payload
		return nil
	}}
	p := NewPipeline(w, 3, time.Millisecond)

	j := &models.Job{ID: "j1", DocumentID: "doc1", Operation: models.JobIndex, Payload: models.JobPayload(`{"key":"a"}`)}
	if err := p.Process(context.Background(), j); err != nil {
		t.Fatalf("Process: %v", err)
	}
	if gotID != "doc1" || string(gotPayload) != `{"key":"a"}` {
		t.Errorf("writer got (%q,%q)", gotID, gotPayload)
	}
}

func TestPipeline_DeleteSuccess(t *testing.T) {
	var deleted string
	w := &mockWriter{delete: func(_ context.Context, id string) error { deleted = id; return nil }}
	p := NewPipeline(w, 3, time.Millisecond)

	j := &models.Job{ID: "j1", DocumentID: "doc1", Operation: models.JobDelete}
	if err := p.Process(context.Background(), j); err != nil {
		t.Fatalf("Process: %v", err)
	}
	if deleted != "doc1" {
		t.Errorf("Delete got %q, want doc1", deleted)
	}
}

func TestPipeline_TransientFailureRetries(t *testing.T) {
	var calls int32
	w := &mockWriter{index: func(_ context.Context, _ string, _ []byte) error {
		if atomic.AddInt32(&calls, 1) < 3 {
			return errors.New("transient")
		}
		return nil // succeeds on the 3rd attempt
	}}
	p := NewPipeline(w, 5, time.Millisecond)

	j := &models.Job{ID: "j1", DocumentID: "doc1", Operation: models.JobIndex}
	if err := p.Process(context.Background(), j); err != nil {
		t.Fatalf("expected eventual success, got: %v", err)
	}
	if got := atomic.LoadInt32(&calls); got != 3 {
		t.Errorf("expected 3 attempts, got %d", got)
	}
}

func TestPipeline_TerminalFailure(t *testing.T) {
	var calls int32
	w := &mockWriter{index: func(_ context.Context, _ string, _ []byte) error {
		atomic.AddInt32(&calls, 1)
		return errors.New("permanent")
	}}
	p := NewPipeline(w, 3, time.Millisecond)

	j := &models.Job{ID: "j1", DocumentID: "doc1", Operation: models.JobIndex}
	if err := p.Process(context.Background(), j); err == nil {
		t.Fatal("expected error after exhausting retries")
	}
	if got := atomic.LoadInt32(&calls); got != 3 {
		t.Errorf("expected 3 attempts, got %d", got)
	}
}

func TestPipeline_UnknownOperation(t *testing.T) {
	p := NewPipeline(&mockWriter{}, 1, time.Millisecond)
	j := &models.Job{ID: "j1", DocumentID: "doc1", Operation: "bogus"}
	if err := p.Process(context.Background(), j); err == nil {
		t.Fatal("expected error for unknown operation")
	}
}
