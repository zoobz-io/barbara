package transformers

// AddTag returns tags with tag included, treating the tag list as a set: adding
// a tag already present is a no-op. The bool reports whether the result differs
// from the input, so callers can skip the write (and any reprojection) when
// nothing changed. Wire-free, so it lives in the data-layer transformers.
func AddTag(tags []string, tag string) ([]string, bool) {
	for _, existing := range tags {
		if existing == tag {
			return tags, false
		}
	}
	return append(append([]string(nil), tags...), tag), true
}

// RemoveTag returns tags with tag removed; removing an absent tag is a no-op.
// The bool reports whether the result differs from the input.
func RemoveTag(tags []string, tag string) ([]string, bool) {
	out := make([]string, 0, len(tags))
	for _, existing := range tags {
		if existing != tag {
			out = append(out, existing)
		}
	}
	if len(out) == len(tags) {
		return tags, false
	}
	return out, true
}
