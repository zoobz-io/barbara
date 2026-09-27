// Package mdast parses Markdown into an [mdast] tree: the abstract syntax tree
// that the remark ecosystem uses. It parses with goldmark and walks the
// goldmark AST into typed mdast nodes whose JSON matches what remark's
// mdast-util-from-markdown produces, verified against the golden fixtures in
// testdata/. That match is what lets any remark renderer render a Barbara page.
//
// The package is a leaf: it imports goldmark and the standard library, nothing
// from the rest of Barbara.
//
// # Struct tags
//
// Alongside the JSON tags, node types carry rocco schema tags so the OpenAPI
// spec (and the generated SDK) describes the tree as a typed, discriminated
// union rather than an opaque blob: each Type field is a const of its node
// name, and each children field is a union over every non-root node type,
// discriminated by that type. These tags are inert strings — the package still
// imports nothing outside goldmark and the standard library.
//
// [mdast]: https://github.com/syntax-tree/mdast
package mdast

// nodeUnion lists every node type a children field may hold — every node type
// except MdastRoot, which appears only at the top. It is the discriminate tag value
// on each children field. It is duplicated in those tags because Go struct tags
// must be string literals; keep this doc list and the tags in step.
//
//	MdastParagraph,MdastHeading,MdastThematicBreak,MdastBlockquote,MdastList,MdastListItem,MdastCode,MdastHTML,MdastText,
//	MdastEmphasis,MdastStrong,MdastDelete,MdastInlineCode,MdastBreak,MdastLink,MdastImage,MdastLinkReference,
//	MdastImageReference,MdastDefinition,MdastFootnoteReference,MdastFootnoteDefinition,MdastTable,MdastTableRow,MdastTableCell

// Node is a node in an mdast tree. The set of implementations is closed — the
// interface's method is unexported, so only this package defines node types.
type Node interface {
	mdastNode()
}

// MdastRoot is the document root.
type MdastRoot struct {
	Type     string `json:"type" const:"root"`
	Children []Node `json:"children" discriminate:"MdastParagraph,MdastHeading,MdastThematicBreak,MdastBlockquote,MdastList,MdastListItem,MdastCode,MdastHTML,MdastText,MdastEmphasis,MdastStrong,MdastDelete,MdastInlineCode,MdastBreak,MdastLink,MdastImage,MdastLinkReference,MdastImageReference,MdastDefinition,MdastFootnoteReference,MdastFootnoteDefinition,MdastTable,MdastTableRow,MdastTableCell" discriminated_by:"type"`
}

// MdastParagraph is a run of inline content.
type MdastParagraph struct {
	Type     string `json:"type" const:"paragraph"`
	Children []Node `json:"children" discriminate:"MdastParagraph,MdastHeading,MdastThematicBreak,MdastBlockquote,MdastList,MdastListItem,MdastCode,MdastHTML,MdastText,MdastEmphasis,MdastStrong,MdastDelete,MdastInlineCode,MdastBreak,MdastLink,MdastImage,MdastLinkReference,MdastImageReference,MdastDefinition,MdastFootnoteReference,MdastFootnoteDefinition,MdastTable,MdastTableRow,MdastTableCell" discriminated_by:"type"`
}

// MdastHeading is an ATX or setext heading. Depth is 1 through 6.
type MdastHeading struct {
	Type     string `json:"type" const:"heading"`
	Children []Node `json:"children" discriminate:"MdastParagraph,MdastHeading,MdastThematicBreak,MdastBlockquote,MdastList,MdastListItem,MdastCode,MdastHTML,MdastText,MdastEmphasis,MdastStrong,MdastDelete,MdastInlineCode,MdastBreak,MdastLink,MdastImage,MdastLinkReference,MdastImageReference,MdastDefinition,MdastFootnoteReference,MdastFootnoteDefinition,MdastTable,MdastTableRow,MdastTableCell" discriminated_by:"type"`
	Depth    int    `json:"depth"`
}

// MdastThematicBreak is a horizontal rule.
type MdastThematicBreak struct {
	Type string `json:"type" const:"thematicBreak"`
}

// MdastBlockquote is a block quote.
type MdastBlockquote struct {
	Type     string `json:"type" const:"blockquote"`
	Children []Node `json:"children" discriminate:"MdastParagraph,MdastHeading,MdastThematicBreak,MdastBlockquote,MdastList,MdastListItem,MdastCode,MdastHTML,MdastText,MdastEmphasis,MdastStrong,MdastDelete,MdastInlineCode,MdastBreak,MdastLink,MdastImage,MdastLinkReference,MdastImageReference,MdastDefinition,MdastFootnoteReference,MdastFootnoteDefinition,MdastTable,MdastTableRow,MdastTableCell" discriminated_by:"type"`
}

// MdastList is an ordered or unordered list. Start is the ordered start number, or
// nil for an unordered list. Spread is true when the list is loose.
type MdastList struct {
	Start    *int   `json:"start"`
	Type     string `json:"type" const:"list"`
	Children []Node `json:"children" discriminate:"MdastParagraph,MdastHeading,MdastThematicBreak,MdastBlockquote,MdastList,MdastListItem,MdastCode,MdastHTML,MdastText,MdastEmphasis,MdastStrong,MdastDelete,MdastInlineCode,MdastBreak,MdastLink,MdastImage,MdastLinkReference,MdastImageReference,MdastDefinition,MdastFootnoteReference,MdastFootnoteDefinition,MdastTable,MdastTableRow,MdastTableCell" discriminated_by:"type"`
	Ordered  bool   `json:"ordered"`
	Spread   bool   `json:"spread"`
}

// MdastListItem is one item of a list. Checked is non-nil for a GFM task item.
// Spread is true when the item's own blocks are loose.
type MdastListItem struct {
	Checked  *bool  `json:"checked"`
	Type     string `json:"type" const:"listItem"`
	Children []Node `json:"children" discriminate:"MdastParagraph,MdastHeading,MdastThematicBreak,MdastBlockquote,MdastList,MdastListItem,MdastCode,MdastHTML,MdastText,MdastEmphasis,MdastStrong,MdastDelete,MdastInlineCode,MdastBreak,MdastLink,MdastImage,MdastLinkReference,MdastImageReference,MdastDefinition,MdastFootnoteReference,MdastFootnoteDefinition,MdastTable,MdastTableRow,MdastTableCell" discriminated_by:"type"`
	Spread   bool   `json:"spread"`
}

// MdastCode is a fenced or indented code block. Lang and Meta come from a fence's
// info string and are nil when absent.
type MdastCode struct {
	Type  string  `json:"type" const:"code"`
	Lang  *string `json:"lang"`
	Meta  *string `json:"meta"`
	Value string  `json:"value"`
}

// MdastHTML is a raw HTML block or a run of inline raw HTML.
type MdastHTML struct {
	Type  string `json:"type" const:"html"`
	Value string `json:"value"`
}

// MdastText is a run of plain text. Soft line breaks are kept as "\n" in Value.
type MdastText struct {
	Type  string `json:"type" const:"text"`
	Value string `json:"value"`
}

// MdastEmphasis is emphasized (italic) inline content.
type MdastEmphasis struct {
	Type     string `json:"type" const:"emphasis"`
	Children []Node `json:"children" discriminate:"MdastParagraph,MdastHeading,MdastThematicBreak,MdastBlockquote,MdastList,MdastListItem,MdastCode,MdastHTML,MdastText,MdastEmphasis,MdastStrong,MdastDelete,MdastInlineCode,MdastBreak,MdastLink,MdastImage,MdastLinkReference,MdastImageReference,MdastDefinition,MdastFootnoteReference,MdastFootnoteDefinition,MdastTable,MdastTableRow,MdastTableCell" discriminated_by:"type"`
}

// MdastStrong is strongly emphasized (bold) inline content.
type MdastStrong struct {
	Type     string `json:"type" const:"strong"`
	Children []Node `json:"children" discriminate:"MdastParagraph,MdastHeading,MdastThematicBreak,MdastBlockquote,MdastList,MdastListItem,MdastCode,MdastHTML,MdastText,MdastEmphasis,MdastStrong,MdastDelete,MdastInlineCode,MdastBreak,MdastLink,MdastImage,MdastLinkReference,MdastImageReference,MdastDefinition,MdastFootnoteReference,MdastFootnoteDefinition,MdastTable,MdastTableRow,MdastTableCell" discriminated_by:"type"`
}

// MdastDelete is struck-through (GFM) inline content.
type MdastDelete struct {
	Type     string `json:"type" const:"delete"`
	Children []Node `json:"children" discriminate:"MdastParagraph,MdastHeading,MdastThematicBreak,MdastBlockquote,MdastList,MdastListItem,MdastCode,MdastHTML,MdastText,MdastEmphasis,MdastStrong,MdastDelete,MdastInlineCode,MdastBreak,MdastLink,MdastImage,MdastLinkReference,MdastImageReference,MdastDefinition,MdastFootnoteReference,MdastFootnoteDefinition,MdastTable,MdastTableRow,MdastTableCell" discriminated_by:"type"`
}

// MdastInlineCode is an inline code span.
type MdastInlineCode struct {
	Type  string `json:"type" const:"inlineCode"`
	Value string `json:"value"`
}

// MdastBreak is a hard line break.
type MdastBreak struct {
	Type string `json:"type" const:"break"`
}

// MdastLink is an inline link or an autolink.
type MdastLink struct {
	Type     string  `json:"type" const:"link"`
	URL      string  `json:"url"`
	Title    *string `json:"title"`
	Children []Node  `json:"children" discriminate:"MdastParagraph,MdastHeading,MdastThematicBreak,MdastBlockquote,MdastList,MdastListItem,MdastCode,MdastHTML,MdastText,MdastEmphasis,MdastStrong,MdastDelete,MdastInlineCode,MdastBreak,MdastLink,MdastImage,MdastLinkReference,MdastImageReference,MdastDefinition,MdastFootnoteReference,MdastFootnoteDefinition,MdastTable,MdastTableRow,MdastTableCell" discriminated_by:"type"`
}

// MdastImage is an inline image. Alt is the image's plain-text description.
type MdastImage struct {
	Type  string  `json:"type" const:"image"`
	URL   string  `json:"url"`
	Title *string `json:"title"`
	Alt   string  `json:"alt"`
}

// MdastLinkReference is a link that points at a MdastDefinition by identifier.
type MdastLinkReference struct {
	Type          string `json:"type" const:"linkReference"`
	Identifier    string `json:"identifier"`
	Label         string `json:"label"`
	ReferenceType string `json:"referenceType"`
	Children      []Node `json:"children" discriminate:"MdastParagraph,MdastHeading,MdastThematicBreak,MdastBlockquote,MdastList,MdastListItem,MdastCode,MdastHTML,MdastText,MdastEmphasis,MdastStrong,MdastDelete,MdastInlineCode,MdastBreak,MdastLink,MdastImage,MdastLinkReference,MdastImageReference,MdastDefinition,MdastFootnoteReference,MdastFootnoteDefinition,MdastTable,MdastTableRow,MdastTableCell" discriminated_by:"type"`
}

// MdastImageReference is an image that points at a MdastDefinition by identifier.
type MdastImageReference struct {
	Type          string `json:"type" const:"imageReference"`
	Identifier    string `json:"identifier"`
	Label         string `json:"label"`
	ReferenceType string `json:"referenceType"`
	Alt           string `json:"alt"`
}

// MdastDefinition is a link reference definition.
type MdastDefinition struct {
	Type       string  `json:"type" const:"definition"`
	Identifier string  `json:"identifier"`
	Label      string  `json:"label"`
	Title      *string `json:"title"`
	URL        string  `json:"url"`
}

// MdastFootnoteReference is an inline reference to a MdastFootnoteDefinition.
type MdastFootnoteReference struct {
	Type       string `json:"type" const:"footnoteReference"`
	Identifier string `json:"identifier"`
	Label      string `json:"label"`
}

// MdastFootnoteDefinition is the content of a footnote, referenced by identifier.
type MdastFootnoteDefinition struct {
	Type       string `json:"type" const:"footnoteDefinition"`
	Identifier string `json:"identifier"`
	Label      string `json:"label"`
	Children   []Node `json:"children" discriminate:"MdastParagraph,MdastHeading,MdastThematicBreak,MdastBlockquote,MdastList,MdastListItem,MdastCode,MdastHTML,MdastText,MdastEmphasis,MdastStrong,MdastDelete,MdastInlineCode,MdastBreak,MdastLink,MdastImage,MdastLinkReference,MdastImageReference,MdastDefinition,MdastFootnoteReference,MdastFootnoteDefinition,MdastTable,MdastTableRow,MdastTableCell" discriminated_by:"type"`
}

// MdastTable is a GFM table. Align holds one entry per column: "left", "right",
// "center", or nil for the default.
type MdastTable struct {
	Type     string    `json:"type" const:"table"`
	Align    []*string `json:"align"`
	Children []Node    `json:"children" discriminate:"MdastParagraph,MdastHeading,MdastThematicBreak,MdastBlockquote,MdastList,MdastListItem,MdastCode,MdastHTML,MdastText,MdastEmphasis,MdastStrong,MdastDelete,MdastInlineCode,MdastBreak,MdastLink,MdastImage,MdastLinkReference,MdastImageReference,MdastDefinition,MdastFootnoteReference,MdastFootnoteDefinition,MdastTable,MdastTableRow,MdastTableCell" discriminated_by:"type"`
}

// MdastTableRow is one row of a MdastTable, header or body.
type MdastTableRow struct {
	Type     string `json:"type" const:"tableRow"`
	Children []Node `json:"children" discriminate:"MdastParagraph,MdastHeading,MdastThematicBreak,MdastBlockquote,MdastList,MdastListItem,MdastCode,MdastHTML,MdastText,MdastEmphasis,MdastStrong,MdastDelete,MdastInlineCode,MdastBreak,MdastLink,MdastImage,MdastLinkReference,MdastImageReference,MdastDefinition,MdastFootnoteReference,MdastFootnoteDefinition,MdastTable,MdastTableRow,MdastTableCell" discriminated_by:"type"`
}

// MdastTableCell is one cell of a MdastTableRow.
type MdastTableCell struct {
	Type     string `json:"type" const:"tableCell"`
	Children []Node `json:"children" discriminate:"MdastParagraph,MdastHeading,MdastThematicBreak,MdastBlockquote,MdastList,MdastListItem,MdastCode,MdastHTML,MdastText,MdastEmphasis,MdastStrong,MdastDelete,MdastInlineCode,MdastBreak,MdastLink,MdastImage,MdastLinkReference,MdastImageReference,MdastDefinition,MdastFootnoteReference,MdastFootnoteDefinition,MdastTable,MdastTableRow,MdastTableCell" discriminated_by:"type"`
}

func (*MdastRoot) mdastNode()               {}
func (*MdastParagraph) mdastNode()          {}
func (*MdastHeading) mdastNode()            {}
func (*MdastThematicBreak) mdastNode()      {}
func (*MdastBlockquote) mdastNode()         {}
func (*MdastList) mdastNode()               {}
func (*MdastListItem) mdastNode()           {}
func (*MdastCode) mdastNode()               {}
func (*MdastHTML) mdastNode()               {}
func (*MdastText) mdastNode()               {}
func (*MdastEmphasis) mdastNode()           {}
func (*MdastStrong) mdastNode()             {}
func (*MdastDelete) mdastNode()             {}
func (*MdastInlineCode) mdastNode()         {}
func (*MdastBreak) mdastNode()              {}
func (*MdastLink) mdastNode()               {}
func (*MdastImage) mdastNode()              {}
func (*MdastLinkReference) mdastNode()      {}
func (*MdastImageReference) mdastNode()     {}
func (*MdastDefinition) mdastNode()         {}
func (*MdastFootnoteReference) mdastNode()  {}
func (*MdastFootnoteDefinition) mdastNode() {}
func (*MdastTable) mdastNode()              {}
func (*MdastTableRow) mdastNode()           {}
func (*MdastTableCell) mdastNode()          {}
