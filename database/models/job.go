package models

import (
	"database/sql/driver"
	"fmt"
	"time"
)

// JobStatus is the state of an outbox job.
type JobStatus = string

// Job status values.
const (
	JobPending    JobStatus = "pending"
	JobProcessing JobStatus = "processing"
	JobDone       JobStatus = "done"
	JobFailed     JobStatus = "failed"
)

// JobOperation is the OpenSearch operation a job performs.
type JobOperation = string

// Job operation values.
const (
	JobIndex  JobOperation = "index"
	JobDelete JobOperation = "delete"
)

// JobPayload is the JSONB projection carried by an index job. It is stored as
// text so lib/pq sends it to a jsonb column rather than as bytea.
type JobPayload []byte

// Value implements driver.Valuer, sending the payload as jsonb text (or NULL).
func (p JobPayload) Value() (driver.Value, error) {
	if len(p) == 0 {
		return nil, nil
	}
	return string(p), nil
}

// Scan implements sql.Scanner for jsonb columns.
func (p *JobPayload) Scan(src any) error {
	switch v := src.(type) {
	case nil:
		*p = nil
	case []byte:
		*p = append((*p)[:0], v...)
	case string:
		*p = []byte(v)
	default:
		return fmt.Errorf("cannot scan %T into JobPayload", src)
	}
	return nil
}

// Job is an outbox entry for an async OpenSearch write. Publishing enqueues a
// job inside the same transaction that moves the published pointer (so the two
// commit atomically); the pipeline claims pending jobs and performs the write,
// retrying until it lands.
type Job struct {
	CreatedAt  time.Time    `json:"created_at" db:"created_at" default:"now()"`
	UpdatedAt  time.Time    `json:"updated_at" db:"updated_at" default:"now()"`
	LastError  *string      `json:"last_error,omitempty" db:"last_error"`
	ID         string       `json:"id" db:"id" constraints:"primarykey"`
	TenantID   string       `json:"tenant_id" db:"tenant_id" constraints:"notnull"`
	DocumentID string       `json:"document_id" db:"document_id" constraints:"notnull"`
	Operation  JobOperation `json:"operation" db:"operation" constraints:"notnull"`
	Status     JobStatus    `json:"status" db:"status" constraints:"notnull" default:"'pending'"`
	Payload    JobPayload   `json:"payload,omitempty" db:"payload"`
	Attempts   int          `json:"attempts" db:"attempts" default:"0"`
}

// GetID returns the job's primary key.
func (j Job) GetID() string { return j.ID }

// Clone returns a deep copy of the job.
func (j Job) Clone() Job {
	c := j
	if j.LastError != nil {
		e := *j.LastError
		c.LastError = &e
	}
	if j.Payload != nil {
		c.Payload = append(JobPayload(nil), j.Payload...)
	}
	return c
}
