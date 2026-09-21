package mdast

import (
	"bytes"
	"regexp"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	east "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/text"
)

// parser is a goldmark parser with the GFM extension (tables, strikethrough,
// task lists, and autolinks). It is built once and reused; goldmark parsers are
// safe for concurrent use.
var parser = goldmark.New(goldmark.WithExtensions(extension.GFM)).Parser()

// Parse parses Markdown into an mdast tree.
func Parse(src []byte) (*Root, error) {
	doc := parser.Parse(text.NewReader(src))
	return &Root{Type: "root", Children: convertBlocks(doc, src)}, nil
}

// convertBlocks converts the block-level children of a container.
func convertBlocks(parent ast.Node, src []byte) []Node {
	var out []Node
	for c := parent.FirstChild(); c != nil; c = c.NextSibling() {
		if n := convertBlock(c, src); n != nil {
			out = append(out, n)
		}
	}
	return out
}

// convertBlock converts one block-level node.
func convertBlock(n ast.Node, src []byte) Node {
	switch b := n.(type) {
	case *ast.Paragraph:
		return &Paragraph{Type: "paragraph", Children: convertInline(b, src)}
	case *ast.TextBlock:
		// A tight list item's content. mdast still wraps it in a paragraph.
		return &Paragraph{Type: "paragraph", Children: convertInline(b, src)}
	case *ast.Heading:
		return &Heading{Type: "heading", Depth: b.Level, Children: convertInline(b, src)}
	case *ast.ThematicBreak:
		return &ThematicBreak{Type: "thematicBreak"}
	case *ast.Blockquote:
		return &Blockquote{Type: "blockquote", Children: convertBlocks(b, src)}
	case *ast.List:
		return convertList(b, src)
	case *ast.ListItem:
		return convertListItem(b, src)
	case *ast.FencedCodeBlock:
		return convertFencedCode(b, src)
	case *ast.CodeBlock:
		value := linesValue(b.Lines(), src)
		return &Code{Type: "code", Lang: nil, Meta: nil, Value: value}
	case *ast.HTMLBlock:
		value := linesValue(b.Lines(), src)
		if b.HasClosure() {
			value += string(b.ClosureLine.Value(src))
		}
		return &HTML{Type: "html", Value: strings.TrimSuffix(value, "\n")}
	case *ast.LinkReferenceDefinition:
		return convertDefinition(b)
	case *east.Table:
		return convertTable(b, src)
	}
	return nil
}

// convertList converts a list and its items.
func convertList(l *ast.List, src []byte) *List {
	var start *int
	if l.IsOrdered() {
		s := l.Start
		start = &s
	}
	return &List{
		Type:     "list",
		Ordered:  l.IsOrdered(),
		Start:    start,
		Spread:   !l.IsTight,
		Children: convertBlocks(l, src),
	}
}

// convertListItem converts a list item, lifting a leading GFM task checkbox out
// of its content and onto the item's Checked field.
func convertListItem(item *ast.ListItem, src []byte) *ListItem {
	return &ListItem{
		Type:     "listItem",
		Spread:   itemSpread(item, src),
		Checked:  taskChecked(item),
		Children: convertBlocks(item, src),
	}
}

// itemSpread reports whether a list item is spread: whether a blank line
// separates two of its block children. This is the remark rule — a list item's
// own looseness — and is independent of the list's looseness.
func itemSpread(item *ast.ListItem, src []byte) bool {
	prevStop := -1
	for c := item.FirstChild(); c != nil; c = c.NextSibling() {
		if start := nodeStart(c); prevStop >= 0 && start >= prevStop &&
			bytes.Count(src[prevStop:start], []byte{'\n'}) >= 2 {
			return true
		}
		prevStop = nodeStop(c)
	}
	return false
}

// nodeStart is the smallest source offset covered by a node, found by walking
// its descendants' text segments and block lines. It returns -1 for a node with
// no source span.
func nodeStart(n ast.Node) int {
	start := -1
	consider := func(s int) {
		if s >= 0 && (start < 0 || s < start) {
			start = s
		}
	}
	if t, ok := n.(*ast.Text); ok {
		consider(t.Segment.Start)
	}
	if n.Type() == ast.TypeBlock {
		if ls := n.Lines(); ls != nil && ls.Len() > 0 {
			seg := ls.At(0)
			consider(seg.Start)
		}
	}
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		consider(nodeStart(c))
	}
	return start
}

// nodeStop is the largest source offset covered by a node.
func nodeStop(n ast.Node) int {
	stop := -1
	consider := func(s int) {
		if s > stop {
			stop = s
		}
	}
	if t, ok := n.(*ast.Text); ok {
		consider(t.Segment.Stop)
	}
	if n.Type() == ast.TypeBlock {
		if ls := n.Lines(); ls != nil && ls.Len() > 0 {
			seg := ls.At(ls.Len() - 1)
			consider(seg.Stop)
		}
	}
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		consider(nodeStop(c))
	}
	return stop
}

// taskChecked returns the checkbox state of a GFM task item, or nil when the
// item is not a task item. The checkbox is the first inline child of the item's
// first block.
func taskChecked(item *ast.ListItem) *bool {
	first := item.FirstChild()
	if first == nil {
		return nil
	}
	if box, ok := first.FirstChild().(*east.TaskCheckBox); ok {
		checked := box.IsChecked
		return &checked
	}
	return nil
}

// convertFencedCode converts a fenced code block, splitting the info string
// into a language and the trailing meta.
func convertFencedCode(b *ast.FencedCodeBlock, src []byte) *Code {
	value := linesValue(b.Lines(), src)
	var lang, meta *string
	if b.Info != nil {
		info := string(b.Info.Segment.Value(src))
		if head, tail, found := strings.Cut(info, " "); found {
			lang, meta = &head, &tail
		} else if info != "" {
			lang = &info
		}
	}
	return &Code{Type: "code", Lang: lang, Meta: meta, Value: value}
}

// convertTable converts a GFM table. goldmark models the header as a distinct
// node; mdast represents it as an ordinary row, so both map to a table row.
func convertTable(t *east.Table, src []byte) *Table {
	align := make([]*string, len(t.Alignments))
	for i, a := range t.Alignments {
		align[i] = alignString(a)
	}
	var rows []Node
	for c := t.FirstChild(); c != nil; c = c.NextSibling() {
		rows = append(rows, &TableRow{Type: "tableRow", Children: convertCells(c, src)})
	}
	return &Table{Type: "table", Align: align, Children: rows}
}

// convertCells converts the cells of a table row.
func convertCells(row ast.Node, src []byte) []Node {
	var cells []Node
	for c := row.FirstChild(); c != nil; c = c.NextSibling() {
		cells = append(cells, &TableCell{Type: "tableCell", Children: convertInline(c, src)})
	}
	return cells
}

// convertDefinition converts a link reference definition.
func convertDefinition(d *ast.LinkReferenceDefinition) *Definition {
	label := string(d.Label)
	return &Definition{
		Type:       "definition",
		Identifier: normalizeIdentifier(label),
		Label:      label,
		Title:      optString(d.Title),
		URL:        string(d.Destination),
	}
}

// convertInline converts the inline children of a container into mdast nodes,
// merging adjacent text runs. goldmark splits text at every inline boundary and
// line; mdast keeps one text node per run. A soft line break becomes "\n"
// inside the text value; a hard line break becomes a break node.
func convertInline(parent ast.Node, src []byte) []Node {
	var out []Node
	var buf strings.Builder
	flush := func() {
		if buf.Len() > 0 {
			out = append(out, &Text{Type: "text", Value: buf.String()})
			buf.Reset()
		}
	}
	for c := parent.FirstChild(); c != nil; c = c.NextSibling() {
		switch t := c.(type) {
		case *ast.Text:
			buf.Write([]byte(decodeText(t.Segment.Value(src))))
			if t.SoftLineBreak() {
				buf.WriteByte('\n')
			}
			if t.HardLineBreak() {
				flush()
				out = append(out, &Break{Type: "break"})
			}
		case *ast.String:
			buf.WriteString(decodeText(t.Value))
		case *east.TaskCheckBox:
			// Lifted onto the list item; it contributes no inline content.
		default:
			flush()
			if n := convertInlineNode(c, src); n != nil {
				out = append(out, n)
			}
		}
	}
	flush()
	return out
}

// convertInlineNode converts one non-text inline node.
func convertInlineNode(n ast.Node, src []byte) Node {
	switch i := n.(type) {
	case *ast.Emphasis:
		if i.Level == 2 {
			return &Strong{Type: "strong", Children: convertInline(i, src)}
		}
		return &Emphasis{Type: "emphasis", Children: convertInline(i, src)}
	case *east.Strikethrough:
		return &Delete{Type: "delete", Children: convertInline(i, src)}
	case *ast.CodeSpan:
		return &InlineCode{Type: "inlineCode", Value: codeSpanValue(i, src)}
	case *ast.Link:
		return convertLink(i, src)
	case *ast.AutoLink:
		return convertAutoLink(i, src)
	case *ast.Image:
		return convertImage(i, src)
	case *ast.RawHTML:
		return &HTML{Type: "html", Value: rawHTMLValue(i, src)}
	}
	return nil
}

// convertLink converts an inline link, or a reference link when it resolves to
// a definition.
func convertLink(l *ast.Link, src []byte) Node {
	if l.Reference != nil {
		label := string(l.Reference.Value)
		return &LinkReference{
			Type:          "linkReference",
			Identifier:    normalizeIdentifier(label),
			Label:         label,
			ReferenceType: refType(l.Reference.Type),
			Children:      convertInline(l, src),
		}
	}
	return &Link{
		Type:     "link",
		URL:      string(l.Destination),
		Title:    optString(l.Title),
		Children: convertInline(l, src),
	}
}

// convertAutoLink converts an autolink (an angle-bracket link or a GFM bare
// URL) into a link whose single text child is the shown URL.
func convertAutoLink(a *ast.AutoLink, src []byte) Node {
	label := string(a.Label(src))
	return &Link{
		Type:     "link",
		URL:      string(a.URL(src)),
		Title:    nil,
		Children: []Node{&Text{Type: "text", Value: label}},
	}
}

// convertImage converts an inline image, or an image reference when it resolves
// to a definition.
func convertImage(img *ast.Image, src []byte) Node {
	alt := textContent(img, src)
	if img.Reference != nil {
		label := string(img.Reference.Value)
		return &ImageReference{
			Type:          "imageReference",
			Identifier:    normalizeIdentifier(label),
			Label:         label,
			ReferenceType: refType(img.Reference.Type),
			Alt:           alt,
		}
	}
	return &Image{
		Type:  "image",
		URL:   string(img.Destination),
		Title: optString(img.Title),
		Alt:   alt,
	}
}

// codeSpanValue is the literal content of an inline code span. goldmark stores
// it as raw text children, which are not decoded.
func codeSpanValue(n *ast.CodeSpan, src []byte) string {
	var b strings.Builder
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		if t, ok := c.(*ast.Text); ok {
			b.Write(t.Segment.Value(src))
		}
	}
	return b.String()
}

// rawHTMLValue is the literal content of a run of inline raw HTML.
func rawHTMLValue(n *ast.RawHTML, src []byte) string {
	var b strings.Builder
	for i := 0; i < n.Segments.Len(); i++ {
		seg := n.Segments.At(i)
		b.Write(seg.Value(src))
	}
	return b.String()
}

// textContent is the concatenated decoded text of a node's descendants, used
// for an image's plain-text alt.
func textContent(n ast.Node, src []byte) string {
	var b strings.Builder
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		if t, ok := c.(*ast.Text); ok {
			b.WriteString(decodeText(t.Segment.Value(src)))
		} else {
			b.WriteString(textContent(c, src))
		}
	}
	return b.String()
}

// linesValue joins a block's source lines and drops the single trailing
// newline, matching how mdast stores a code or HTML block's value.
func linesValue(lines *text.Segments, src []byte) string {
	var b strings.Builder
	for i := 0; i < lines.Len(); i++ {
		seg := lines.At(i)
		b.Write(seg.Value(src))
	}
	return strings.TrimSuffix(b.String(), "\n")
}

// optString returns a decoded pointer for a non-empty title, or nil. A title,
// like text, has backslash escapes and character references resolved.
func optString(v []byte) *string {
	if len(v) == 0 {
		return nil
	}
	s := decodeText(v)
	return &s
}

func refType(t ast.ReferenceLinkType) string {
	switch t {
	case ast.ReferenceLinkFull:
		return "full"
	case ast.ReferenceLinkCollapsed:
		return "collapsed"
	default:
		return "shortcut"
	}
}

func alignString(a east.Alignment) *string {
	var s string
	switch a {
	case east.AlignLeft:
		s = "left"
	case east.AlignRight:
		s = "right"
	case east.AlignCenter:
		s = "center"
	default:
		return nil
	}
	return &s
}

var whitespaceRun = regexp.MustCompile(`[ \t\r\n]+`)

// normalizeIdentifier folds a reference label to its identifier the way
// micromark does: collapse whitespace runs to a single space, trim, and case
// fold.
func normalizeIdentifier(label string) string {
	collapsed := strings.TrimSpace(whitespaceRun.ReplaceAllString(label, " "))
	return strings.ToLower(strings.ToUpper(collapsed))
}
