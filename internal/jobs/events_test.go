package jobs

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/zoobz-io/barbara/database/models"
	"github.com/zoobz-io/barbara/events"
)

// A landed OpenSearch write emits Index.WriteSucceeded with the document and
// operation.
func TestPipeline_EmitsWriteSucceeded(t *testing.T) {
	p := NewPipeline(&mockWriter{
		index: func(context.Context, string, []byte) error { return nil },
	}, 3, time.Millisecond)

	var got events.IndexWriteSucceededEvent
	fired := false
	l := events.Index.WriteSucceeded.Listen(func(_ context.Context, e events.IndexWriteSucceededEvent) {
		got, fired = e, true
	})
	defer l.Close()

	if err := p.Process(context.Background(), &models.Job{DocumentID: "d-1", Operation: models.JobIndex}); err != nil {
		t.Fatalf("process: %v", err)
	}
	if !fired || got.DocumentID != "d-1" || got.Operation != "index" {
		t.Errorf("WriteSucceeded = %+v (fired=%v), want d-1/index", got, fired)
	}
}

// A terminally-failed write (retries exhausted) emits Index.WriteFailed carrying
// the error, and no success event.
func TestPipeline_EmitsWriteFailed(t *testing.T) {
	p := NewPipeline(&mockWriter{
		index: func(context.Context, string, []byte) error { return errors.New("os down") },
	}, 2, time.Millisecond)

	failed := false
	var got events.IndexWriteFailedEvent
	lf := events.Index.WriteFailed.Listen(func(_ context.Context, e events.IndexWriteFailedEvent) {
		got, failed = e, true
	})
	defer lf.Close()
	succeeded := false
	ls := events.Index.WriteSucceeded.Listen(func(context.Context, events.IndexWriteSucceededEvent) {
		succeeded = true
	})
	defer ls.Close()

	if err := p.Process(context.Background(), &models.Job{DocumentID: "d-2", Operation: models.JobIndex}); err == nil {
		t.Fatal("expected a terminal failure")
	}
	if !failed || got.DocumentID != "d-2" || got.Error == "" {
		t.Errorf("WriteFailed = %+v (fired=%v), want d-2 with an error", got, failed)
	}
	if succeeded {
		t.Error("WriteSucceeded emitted for a failed write")
	}
}
