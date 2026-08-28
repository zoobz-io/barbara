package transformers

import "strconv"

// Pagination converts the query parameters at the request boundary into a
// limit/offset pair, applying defaults (limit 50, offset 0) and ignoring
// non-positive or unparseable values. This is the wire -> domain half of the
// list transform.
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
