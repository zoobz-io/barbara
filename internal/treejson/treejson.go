// Package treejson encodes a syntax tree the way the golden fixtures under
// internal/mdast/testdata are written, so a converter's output can be compared
// with them byte for byte. Both internal/mdast and internal/hast test against
// those fixtures, so the encoder lives here rather than in either package's
// tests.
//
// The fixtures are produced by web/scripts/mdast-fixtures.mjs with sorted object
// keys, a two-space indent, no HTML escaping of <, >, and &, and a trailing
// newline. Canonical reproduces that exactly.
package treejson

import (
	"bytes"
	"encoding/json"
	"strconv"
)

// Canonical marshals v the way the fixtures are written: sorted keys, a
// two-space indent, unescaped HTML characters, and a trailing newline. The
// value is round-tripped through a generic value first so map-key sorting
// applies at every level (Go sorts map keys but not struct fields).
func Canonical(v any) ([]byte, error) {
	raw, err := encode(v, "")
	if err != nil {
		return nil, err
	}
	var generic any
	if err := json.Unmarshal(raw, &generic); err != nil {
		return nil, err
	}
	return encode(generic, "  ")
}

// Indent marshals an already-generic value with the fixture's indent and
// escaping. It suits data that is not a tree of structs — a decoded metadata
// map, say — where map-key sorting alone gives a stable result.
func Indent(v any) ([]byte, error) {
	return encode(v, "  ")
}

// encode marshals v with HTML escaping disabled and the given indent (empty for
// none), always ending in a newline (json.Encoder appends one).
func encode(v any, indent string) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if indent != "" {
		enc.SetIndent("", indent)
	}
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// FirstDiff returns a short excerpt of both sides around the first byte that
// differs, so a failing comparison points at where the trees diverge.
func FirstDiff(want, got []byte) string {
	i := 0
	for i < len(want) && i < len(got) && want[i] == got[i] {
		i++
	}
	start := i - 60
	if start < 0 {
		start = 0
	}
	win := func(b []byte) string {
		end := i + 60
		if end > len(b) {
			end = len(b)
		}
		return string(b[start:end])
	}
	return "at byte " + strconv.Itoa(i) + "\n--- want ---\n" + win(want) + "\n--- got ---\n" + win(got)
}
