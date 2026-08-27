package models

import (
	"testing"
	"time"
)

func TestJobPayload_Value(t *testing.T) {
	// Empty payload sends NULL, not an empty string, so the jsonb column is null.
	v, err := JobPayload(nil).Value()
	if err != nil {
		t.Fatalf("Value(nil): %v", err)
	}
	if v != nil {
		t.Errorf("empty payload Value = %v, want nil", v)
	}

	// Non-empty payload sends jsonb text.
	v, err = JobPayload(`{"k":1}`).Value()
	if err != nil {
		t.Fatalf("Value: %v", err)
	}
	if s, ok := v.(string); !ok || s != `{"k":1}` {
		t.Errorf("payload Value = %v, want jsonb text", v)
	}
}

func TestJobPayload_Scan(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		var p JobPayload
		if err := p.Scan(nil); err != nil {
			t.Fatalf("Scan(nil): %v", err)
		}
		if p != nil {
			t.Errorf("Scan(nil) = %v, want nil", p)
		}
	})
	t.Run("bytes", func(t *testing.T) {
		var p JobPayload
		if err := p.Scan([]byte(`{"k":1}`)); err != nil {
			t.Fatalf("Scan([]byte): %v", err)
		}
		if string(p) != `{"k":1}` {
			t.Errorf("Scan([]byte) = %q", string(p))
		}
	})
	t.Run("string", func(t *testing.T) {
		var p JobPayload
		if err := p.Scan(`{"k":2}`); err != nil {
			t.Fatalf("Scan(string): %v", err)
		}
		if string(p) != `{"k":2}` {
			t.Errorf("Scan(string) = %q", string(p))
		}
	})
	t.Run("unsupported type errors", func(t *testing.T) {
		var p JobPayload
		if err := p.Scan(42); err == nil {
			t.Error("Scan(int) should error")
		}
	})
}

func TestJob_GetID(t *testing.T) {
	j := Job{ID: "job-1"}
	if j.GetID() != "job-1" {
		t.Errorf("GetID = %q, want job-1", j.GetID())
	}
}

func TestJob_Clone(t *testing.T) {
	msg := "boom"
	orig := Job{
		ID:        "job-1",
		Status:    JobFailed,
		Attempts:  3,
		LastError: &msg,
		Payload:   JobPayload(`{"k":1}`),
		CreatedAt: time.Now(),
	}
	c := orig.Clone()

	if c.ID != orig.ID || c.Attempts != orig.Attempts || c.Status != orig.Status {
		t.Error("Clone did not copy scalar fields")
	}
	// LastError is deep-copied: mutating the clone's pointee must not touch orig.
	*c.LastError = "changed"
	if *orig.LastError != "boom" {
		t.Error("Clone shares LastError pointer")
	}
	// Payload is deep-copied.
	c.Payload[0] = 'X'
	if orig.Payload[0] == 'X' {
		t.Error("Clone shares Payload backing array")
	}
}

func TestJob_Clone_NilFields(t *testing.T) {
	// A job with no LastError and no Payload clones without panicking.
	c := Job{ID: "job-2"}.Clone()
	if c.LastError != nil || c.Payload != nil {
		t.Error("Clone should leave nil fields nil")
	}
}
