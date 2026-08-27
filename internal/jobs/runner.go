package jobs

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/zoobz-io/barbara/database/models"
)

// Store is the runner's view of the jobs outbox.
type Store interface {
	ClaimPending(ctx context.Context, limit int) ([]*models.Job, error)
	MarkDone(ctx context.Context, id string) error
	MarkFailed(ctx context.Context, id, errMsg string) error
}

// Default runner tuning.
const (
	DefaultInterval = time.Second
	DefaultBatch    = 50
)

// Runner polls the outbox on an interval and drives each claimed job through the
// pipeline, recording the outcome. A terminal pipeline failure marks the job
// failed with its error; success marks it done.
type Runner struct {
	store    Store
	pipeline *Pipeline
	interval time.Duration
	batch    int
	wg       sync.WaitGroup
}

// NewRunner creates a runner. A non-positive interval or batch falls back to the
// defaults.
func NewRunner(store Store, pipeline *Pipeline, interval time.Duration, batch int) *Runner {
	if interval <= 0 {
		interval = DefaultInterval
	}
	if batch <= 0 {
		batch = DefaultBatch
	}
	return &Runner{store: store, pipeline: pipeline, interval: interval, batch: batch}
}

// Start launches the poll loop in the background. It stops when ctx is
// cancelled; call Stop to wait for the loop to drain and exit.
func (r *Runner) Start(ctx context.Context) {
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		t := time.NewTicker(r.interval)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				r.drain(ctx)
			}
		}
	}()
}

// Stop waits for the poll loop to exit. The caller cancels the context passed to
// Start to trigger shutdown.
func (r *Runner) Stop() {
	r.wg.Wait()
}

// drain claims a batch and processes each job, recording its outcome.
func (r *Runner) drain(ctx context.Context) {
	claimed, err := r.store.ClaimPending(ctx, r.batch)
	if err != nil {
		log.Printf("jobs: claim pending: %v", err)
		return
	}
	for _, j := range claimed {
		if procErr := r.pipeline.Process(ctx, j); procErr != nil {
			if markErr := r.store.MarkFailed(ctx, j.ID, procErr.Error()); markErr != nil {
				log.Printf("jobs: mark failed (%s): %v", j.ID, markErr)
			}
			continue
		}
		if markErr := r.store.MarkDone(ctx, j.ID); markErr != nil {
			log.Printf("jobs: mark done (%s): %v", j.ID, markErr)
		}
	}
}
