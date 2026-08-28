package transformers

import "testing"

func TestPagination(t *testing.T) {
	cases := []struct {
		name                  string
		query                 map[string]string
		wantLimit, wantOffset int
	}{
		{"defaults", map[string]string{}, 50, 0},
		{"explicit", map[string]string{"limit": "10", "offset": "20"}, 10, 20},
		{"non-positive limit ignored", map[string]string{"limit": "0"}, 50, 0},
		{"negative offset ignored", map[string]string{"offset": "-1"}, 50, 0},
		{"unparseable ignored", map[string]string{"limit": "abc", "offset": "x"}, 50, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			limit, offset := Pagination(tc.query)
			if limit != tc.wantLimit || offset != tc.wantOffset {
				t.Errorf("got (%d,%d), want (%d,%d)", limit, offset, tc.wantLimit, tc.wantOffset)
			}
		})
	}
}
