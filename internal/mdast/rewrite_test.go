package mdast

import "testing"

// as is a checked type assertion that fails the test on a mismatch.
func as[T any](t *testing.T, v any) T {
	t.Helper()
	x, ok := v.(T)
	if !ok {
		t.Fatalf("got %T, want %T", v, *new(T))
	}
	return x
}

// TestRewrite checks that Rewrite passes every link, image, and definition URL
// through resolve and leaves other fields untouched.
func TestRewrite(t *testing.T) {
	title := "t"
	root := &Root{Type: "root", Children: []Node{
		&Paragraph{Type: "paragraph", Children: []Node{
			&Text{Type: "text", Value: "before"},
			&Link{Type: "link", URL: "a.md", Title: &title, Children: []Node{
				&Image{Type: "image", URL: "inner.png", Alt: "inner"},
			}},
			&Image{Type: "image", URL: "b.png", Alt: "b"},
		}},
		&Definition{Type: "definition", Identifier: "d", Label: "d", URL: "c.txt"},
	}}

	var seen []string
	Rewrite(root, func(url string) string {
		seen = append(seen, url)
		return "RW:" + url
	})

	// Every URL was offered to resolve, children included.
	wantSeen := map[string]bool{"a.md": true, "inner.png": true, "b.png": true, "c.txt": true}
	if len(seen) != len(wantSeen) {
		t.Fatalf("resolve saw %v, want the %d URLs", seen, len(wantSeen))
	}
	for _, u := range seen {
		if !wantSeen[u] {
			t.Errorf("resolve saw unexpected URL %q", u)
		}
	}

	para := as[*Paragraph](t, root.Children[0])
	link := as[*Link](t, para.Children[1])
	if link.URL != "RW:a.md" {
		t.Errorf("link URL = %q, want rewritten", link.URL)
	}
	if link.Title == nil || *link.Title != "t" {
		t.Errorf("link title changed: %v", link.Title)
	}
	if as[*Text](t, para.Children[0]).Value != "before" {
		t.Error("text value changed")
	}
	if got := as[*Image](t, link.Children[0]).URL; got != "RW:inner.png" {
		t.Errorf("nested image URL = %q, want rewritten", got)
	}
	if got := as[*Image](t, para.Children[2]).URL; got != "RW:b.png" {
		t.Errorf("image URL = %q, want rewritten", got)
	}
	if got := as[*Definition](t, root.Children[1]).URL; got != "RW:c.txt" {
		t.Errorf("definition URL = %q, want rewritten", got)
	}
}
