package mdast

// Rewrite walks the tree and replaces the URL of every link, image, and
// definition node with resolve(url). It changes only those URL fields; every
// other node and field is left as is.
//
// Rewrite does not decide which URLs to change — resolve does. That keeps this
// package free of any knowledge of Barbara's routes: a caller passes a resolve
// that turns the URLs it cares about (say, relative asset paths) into the URLs
// it wants, and returns the rest unchanged.
func Rewrite(root *Root, resolve func(url string) string) {
	for _, c := range root.Children {
		rewriteNode(c, resolve)
	}
}

func rewriteNode(n Node, resolve func(url string) string) {
	switch t := n.(type) {
	case *Link:
		t.URL = resolve(t.URL)
	case *Image:
		t.URL = resolve(t.URL)
	case *Definition:
		t.URL = resolve(t.URL)
	}
	for _, c := range childrenOf(n) {
		rewriteNode(c, resolve)
	}
}

// childrenOf returns a node's child nodes, or nil for a leaf.
func childrenOf(n Node) []Node {
	switch t := n.(type) {
	case *Root:
		return t.Children
	case *Paragraph:
		return t.Children
	case *Heading:
		return t.Children
	case *Blockquote:
		return t.Children
	case *List:
		return t.Children
	case *ListItem:
		return t.Children
	case *Emphasis:
		return t.Children
	case *Strong:
		return t.Children
	case *Delete:
		return t.Children
	case *Link:
		return t.Children
	case *LinkReference:
		return t.Children
	case *FootnoteDefinition:
		return t.Children
	case *Table:
		return t.Children
	case *TableRow:
		return t.Children
	case *TableCell:
		return t.Children
	}
	return nil
}
