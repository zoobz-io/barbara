// Package jobs runs the async OpenSearch-write outbox: a pipz pipeline that
// performs each job's write with retry, and a runner that claims pending jobs
// from the jobs table and drives them through it. Constructed and started by
// boot; the enqueue side is a transactional method on the stores aggregate.
package jobs

import (
	"context"
	"fmt"
	"time"

	"github.com/zoobz-io/pipz"

	"github.com/zoobz-io/barbara/database/models"
	"github.com/zoobz-io/barbara/events"
)

// IndexWriter is the OpenSearch write side the pipeline drives. It is
// implemented by the search store; the pipeline depends only on this interface
// so it can be tested without a live cluster.
type IndexWriter interface {
	Index(ctx context.Context, documentID string, payload []byte) error
	Delete(ctx context.Context, documentID string) error
}

var (
	osWriteID   = pipz.NewIdentity("os-write", "Write a document projection to OpenSearch")
	pipelineID  = pipz.NewIdentity("jobs-pipeline", "Async OpenSearch write with retry")
)

// Pipeline performs a single job's OpenSearch write, retrying transient
// failures with exponential backoff. When the retries are exhausted the last
// error is returned so the runner can record a terminal failure.
type Pipeline struct {
	chain pipz.Chainable[*models.Job]
}

// NewPipeline builds the pipeline. maxAttempts and baseDelay govern the backoff
// around the OpenSearch write.
func NewPipeline(w IndexWriter, maxAttempts int, baseDelay time.Duration) *Pipeline {
	write := pipz.Effect(osWriteID, func(ctx context.Context, j *models.Job) error {
		switch j.Operation {
		case models.JobIndex:
			return w.Index(ctx, j.DocumentID, []byte(j.Payload))
		case models.JobDelete:
			return w.Delete(ctx, j.DocumentID)
		default:
			return fmt.Errorf("unknown job operation %q", j.Operation)
		}
	})
	return &Pipeline{chain: pipz.NewBackoff(pipelineID, write, maxAttempts, baseDelay)}
}

// Process runs one job through the pipeline, emitting the OpenSearch-write
// outcome — this is where the write actually resolves, after any retries.
func (p *Pipeline) Process(ctx context.Context, j *models.Job) error {
	if _, err := p.chain.Process(ctx, j); err != nil {
		events.Index.WriteFailed.Emit(ctx, events.IndexWriteFailedEvent{
			DocumentID: j.DocumentID, Operation: j.Operation, Error: err.Error(),
		})
		return err
	}
	events.Index.WriteSucceeded.Emit(ctx, events.IndexWriteSucceededEvent{
		DocumentID: j.DocumentID, Operation: j.Operation,
	})
	return nil
}
