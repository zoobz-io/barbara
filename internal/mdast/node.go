// Package mdast parses Markdown into an [mdast] tree: the abstract syntax tree
// that the remark ecosystem uses. It parses with goldmark and walks the
// goldmark AST into typed mdast nodes whose JSON matches what remark's
// mdast-util-from-markdown produces, verified against the golden fixtures in
// testdata/. That match is what lets any remark renderer render a Barbara page.
//
// The package is a leaf: it imports goldmark and the standard library, nothing
// from the rest of Barbara.
//
// [mdast]: https://github.com/syntax-tree/mdast
package mdast

// Node is a node in an mdast tree. The set of implementations is closed — the
// interface's method is unexported, so only this package defines node types.
type Node interface {
	mdastNode()
}

// Root is the document root.
type Root struct {
	Type     string `json:"type"`
	Children []Node `json:"children"`
}

// Paragraph is a run of inline content.
type Paragraph struct {
	Type     string `json:"type"`
	Children []Node `json:"children"`
}

// Heading is an ATX or setext heading. Depth is 1 through 6.
type Heading struct {
	Type     string `json:"type"`
	Children []Node `json:"children"`
	Depth    int    `json:"depth"`
}

// ThematicBreak is a horizontal rule.
type ThematicBreak struct {
	Type string `json:"type"`
}

// Blockquote is a block quote.
type Blockquote struct {
	Type     string `json:"type"`
	Children []Node `json:"children"`
}

// List is an ordered or unordered list. Start is the ordered start number, or
// nil for an unordered list. Spread is true when the list is loose.
type List struct {
	Start    *int   `json:"start"`
	Type     string `json:"type"`
	Children []Node `json:"children"`
	Ordered  bool   `json:"ordered"`
	Spread   bool   `json:"spread"`
}

// ListItem is one item of a list. Checked is non-nil for a GFM task item.
// Spread is true when the item's own blocks are loose.
type ListItem struct {
	Checked  *bool  `json:"checked"`
	Type     string `json:"type"`
	Children []Node `json:"children"`
	Spread   bool   `json:"spread"`
}

// Code is a fenced or indented code block. Lang and Meta come from a fence's
// info string and are nil when absent.
type Code struct {
	Type  string  `json:"type"`
	Lang  *string `json:"lang"`
	Meta  *string `json:"meta"`
	Value string  `json:"value"`
}

// HTML is a raw HTML block or a run of inline raw HTML.
type HTML struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

// Text is a run of plain text. Soft line breaks are kept as "\n" in Value.
type Text struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

// Emphasis is emphasized (italic) inline content.
type Emphasis struct {
	Type     string `json:"type"`
	Children []Node `json:"children"`
}

// Strong is strongly emphasized (bold) inline content.
type Strong struct {
	Type     string `json:"type"`
	Children []Node `json:"children"`
}

// Delete is struck-through (GFM) inline content.
type Delete struct {
	Type     string `json:"type"`
	Children []Node `json:"children"`
}

// InlineCode is an inline code span.
type InlineCode struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

// Break is a hard line break.
type Break struct {
	Type string `json:"type"`
}

// Link is an inline link or an autolink.
type Link struct {
	Type     string  `json:"type"`
	URL      string  `json:"url"`
	Title    *string `json:"title"`
	Children []Node  `json:"children"`
}

// Image is an inline image. Alt is the image's plain-text description.
type Image struct {
	Type  string  `json:"type"`
	URL   string  `json:"url"`
	Title *string `json:"title"`
	Alt   string  `json:"alt"`
}

// LinkReference is a link that points at a Definition by identifier.
type LinkReference struct {
	Type          string `json:"type"`
	Identifier    string `json:"identifier"`
	Label         string `json:"label"`
	ReferenceType string `json:"referenceType"`
	Children      []Node `json:"children"`
}

// ImageReference is an image that points at a Definition by identifier.
type ImageReference struct {
	Type          string `json:"type"`
	Identifier    string `json:"identifier"`
	Label         string `json:"label"`
	ReferenceType string `json:"referenceType"`
	Alt           string `json:"alt"`
}

// Definition is a link reference definition.
type Definition struct {
	Type       string  `json:"type"`
	Identifier string  `json:"identifier"`
	Label      string  `json:"label"`
	Title      *string `json:"title"`
	URL        string  `json:"url"`
}

// FootnoteReference is an inline reference to a FootnoteDefinition.
type FootnoteReference struct {
	Type       string `json:"type"`
	Identifier string `json:"identifier"`
	Label      string `json:"label"`
}

// FootnoteDefinition is the content of a footnote, referenced by identifier.
type FootnoteDefinition struct {
	Type       string `json:"type"`
	Identifier string `json:"identifier"`
	Label      string `json:"label"`
	Children   []Node `json:"children"`
}

// Table is a GFM table. Align holds one entry per column: "left", "right",
// "center", or nil for the default.
type Table struct {
	Type     string    `json:"type"`
	Align    []*string `json:"align"`
	Children []Node    `json:"children"`
}

// TableRow is one row of a Table, header or body.
type TableRow struct {
	Type     string `json:"type"`
	Children []Node `json:"children"`
}

// TableCell is one cell of a TableRow.
type TableCell struct {
	Type     string `json:"type"`
	Children []Node `json:"children"`
}

func (*Root) mdastNode()               {}
func (*Paragraph) mdastNode()          {}
func (*Heading) mdastNode()            {}
func (*ThematicBreak) mdastNode()      {}
func (*Blockquote) mdastNode()         {}
func (*List) mdastNode()               {}
func (*ListItem) mdastNode()           {}
func (*Code) mdastNode()               {}
func (*HTML) mdastNode()               {}
func (*Text) mdastNode()               {}
func (*Emphasis) mdastNode()           {}
func (*Strong) mdastNode()             {}
func (*Delete) mdastNode()             {}
func (*InlineCode) mdastNode()         {}
func (*Break) mdastNode()              {}
func (*Link) mdastNode()               {}
func (*Image) mdastNode()              {}
func (*LinkReference) mdastNode()      {}
func (*ImageReference) mdastNode()     {}
func (*Definition) mdastNode()         {}
func (*FootnoteReference) mdastNode()  {}
func (*FootnoteDefinition) mdastNode() {}
func (*Table) mdastNode()              {}
func (*TableRow) mdastNode()           {}
func (*TableCell) mdastNode()          {}
