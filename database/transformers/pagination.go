package transformers

import "strconv"

// Pagination converts the query parameters at the request boundary into a
// limit/offset pair, applying defaults (limit 50, offset 0) and ignoring
// non-positive or unparseable values. It has no dependency on any surface's
// wire types, so it lives in the data-layer transformers and is shared by every
// surface rather than duplicated per domain module.
func Pagination(query map[string]string) (limit, offset int) {
	limit, offset = 50, 0
	if n, err := strconv.Atoi(query["limit"]); err == nil && n > 0 {
		limit = n
	}
	if n, err := strconv.Atoi(query["offset"]); err == nil && n >= 0 {
		offset = n
	}
	return limit, offset
}
