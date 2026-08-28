package transformers

import (
	"reflect"
	"testing"
)

func TestAddTag(t *testing.T) {
	got, changed := AddTag([]string{"a", "b"}, "c")
	if !changed || !reflect.DeepEqual(got, []string{"a", "b", "c"}) {
		t.Errorf("AddTag append = %v (changed %v), want [a b c] true", got, changed)
	}

	got, changed = AddTag([]string{"a", "b"}, "a")
	if changed || !reflect.DeepEqual(got, []string{"a", "b"}) {
		t.Errorf("AddTag duplicate = %v (changed %v), want [a b] false", got, changed)
	}

	// The input slice must not be mutated by an append.
	src := []string{"a"}
	if _, _ = AddTag(src, "b"); len(src) != 1 {
		t.Errorf("AddTag mutated the input slice: %v", src)
	}
}

func TestRemoveTag(t *testing.T) {
	got, changed := RemoveTag([]string{"a", "b", "c"}, "b")
	if !changed || !reflect.DeepEqual(got, []string{"a", "c"}) {
		t.Errorf("RemoveTag = %v (changed %v), want [a c] true", got, changed)
	}

	got, changed = RemoveTag([]string{"a", "b"}, "z")
	if changed || !reflect.DeepEqual(got, []string{"a", "b"}) {
		t.Errorf("RemoveTag absent = %v (changed %v), want [a b] false", got, changed)
	}
}
