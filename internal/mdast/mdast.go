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
// task lists, autolinks) and footnotes. It is built once and reused; goldmark
// parsers are safe for concurrent use.
var parser = goldmark.New(goldmark.WithExtensions(extension.GFM, extension.Footnote)).Parser()

// Parse parses Markdown into an mdast tree.
func Parse(src []byte) (*Root, error) {
	doc := parser.Parse(text.NewReader(src))
	c := &conv{src: src, footnotes: footnoteLabels(doc)}
	return &Root{Type: "root", Children: c.blocks(doc)}, nil
}

// conv carries per-parse state through the walk.
type conv struct {
	// footnotes maps a footnote's index to its label. goldmark identifies an
	// inline footnote reference by index; mdast identifies it by label.
	footnotes map[int]string
	src       []byte
}

// footnoteLabels reads the index-to-label map from the footnote list goldmark
// appends at the end of a document.
func footnoteLabels(doc ast.Node) map[int]string {
	labels := map[int]string{}
	for c := doc.FirstChild(); c != nil; c = c.NextSibling() {
		list, ok := c.(*east.FootnoteList)
		if !ok {
			continue
		}
		for f := list.FirstChild(); f != nil; f = f.NextSibling() {
			if fn, ok := f.(*east.Footnote); ok {
				labels[fn.Index] = string(fn.Ref)
			}
		}
	}
	return labels
}

// blocks converts the block-level children of a container. A children slice is
// always non-nil so it marshals as [] and never null.
func (c *conv) blocks(parent ast.Node) []Node {
	out := []Node{}
	for ch := parent.FirstChild(); ch != nil; ch = ch.NextSibling() {
		if list, ok := ch.(*east.FootnoteList); ok {
			// goldmark collects footnote definitions into a list at the end of
			// the document; mdast keeps each as a root-level footnoteDefinition.
			for f := list.FirstChild(); f != nil; f = f.NextSibling() {
				if fn, ok := f.(*east.Footnote); ok {
					out = append(out, c.footnoteDefinition(fn))
				}
			}
			continue
		}
		if n := c.block(ch); n != nil {
			out = append(out, n)
		}
	}
	return out
}

// block converts one block-level node.
func (c *conv) block(n ast.Node) Node {
	switch b := n.(type) {
	case *ast.Paragraph:
		return &Paragraph{Type: "paragraph", Children: c.inline(b)}
	case *ast.TextBlock:
		// A tight list item's content. mdast still wraps it in a paragraph.
		return &Paragraph{Type: "paragraph", Children: c.inline(b)}
	case *ast.Heading:
		return &Heading{Type: "heading", Depth: b.Level, Children: c.inline(b)}
	case *ast.ThematicBreak:
		return &ThematicBreak{Type: "thematicBreak"}
	case *ast.Blockquote:
		return &Blockquote{Type: "blockquote", Children: c.blocks(b)}
	case *ast.List:
		return c.list(b)
	case *ast.ListItem:
		return c.listItem(b)
	case *ast.FencedCodeBlock:
		return c.fencedCode(b)
	case *ast.CodeBlock:
		return &Code{Type: "code", Lang: nil, Meta: nil, Value: linesValue(b.Lines(), c.src)}
	case *ast.HTMLBlock:
		value := linesValue(b.Lines(), c.src)
		if b.HasClosure() {
			value += string(b.ClosureLine.Value(c.src))
		}
		return &HTML{Type: "html", Value: strings.TrimSuffix(value, "\n")}
	case *ast.LinkReferenceDefinition:
		return convertDefinition(b)
	case *east.Table:
		return c.table(b)
	}
	return nil
}

// footnoteDefinition converts a goldmark footnote to an mdast footnoteDefinition.
func (c *conv) footnoteDefinition(fn *east.Footnote) *FootnoteDefinition {
	label := string(fn.Ref)
	return &FootnoteDefinition{
		Type:       "footnoteDefinition",
		Identifier: normalizeIdentifier(label),
		Label:      label,
		Children:   c.blocks(fn),
	}
}

// list converts a list and its items.
func (c *conv) list(l *ast.List) *List {
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
		Children: c.blocks(l),
	}
}

// listItem converts a list item, lifting a leading GFM task checkbox out of its
// content and onto the item's Checked field.
func (c *conv) listItem(item *ast.ListItem) *ListItem {
	return &ListItem{
		Type:     "listItem",
		Spread:   c.itemSpread(item),
		Checked:  taskChecked(item),
		Children: c.blocks(item),
	}
}

// itemSpread reports whether a list item is spread: whether a blank line
// separates two of its block children. This is the remark rule — a list item's
// own looseness — and is independent of the list's looseness.
func (c *conv) itemSpread(item *ast.ListItem) bool {
	prevStop := -1
	for ch := item.FirstChild(); ch != nil; ch = ch.NextSibling() {
		if start := nodeStart(ch); prevStop >= 0 && start >= prevStop &&
			bytes.Count(c.src[prevStop:start], []byte{'\n'}) >= 2 {
			return true
		}
		prevStop = nodeStop(ch)
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

// fencedCode converts a fenced code block, splitting the info string into a
// language and the trailing meta. The info string is trimmed, the language is
// the text up to the first whitespace, and the meta is the remainder with its
// leading whitespace dropped — matching remark.
func (c *conv) fencedCode(b *ast.FencedCodeBlock) *Code {
	value := linesValue(b.Lines(), c.src)
	var lang, meta *string
	if b.Info != nil {
		info := strings.TrimSpace(string(b.Info.Segment.Value(c.src)))
		if info != "" {
			head := info
			if i := strings.IndexAny(info, " \t"); i >= 0 {
				head = info[:i]
				if tail := strings.TrimLeft(info[i:], " \t"); tail != "" {
					meta = &tail
				}
			}
			lang = &head
		}
	}
	return &Code{Type: "code", Lang: lang, Meta: meta, Value: value}
}

// table converts a GFM table. goldmark models the header as a distinct node;
// mdast represents it as an ordinary row, so both map to a table row.
func (c *conv) table(t *east.Table) *Table {
	align := make([]*string, len(t.Alignments))
	for i, a := range t.Alignments {
		align[i] = alignString(a)
	}
	rows := []Node{}
	for ch := t.FirstChild(); ch != nil; ch = ch.NextSibling() {
		rows = append(rows, &TableRow{Type: "tableRow", Children: c.cells(ch)})
	}
	return &Table{Type: "table", Align: align, Children: rows}
}

// cells converts the cells of a table row.
func (c *conv) cells(row ast.Node) []Node {
	cells := []Node{}
	for ch := row.FirstChild(); ch != nil; ch = ch.NextSibling() {
		cells = append(cells, &TableCell{Type: "tableCell", Children: c.inline(ch)})
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

// inline converts the inline children of a container into mdast nodes, merging
// adjacent text runs. goldmark splits text at every inline boundary and line;
// mdast keeps one text node per run. A soft line break becomes "\n" inside the
// text value; a hard line break becomes a break node. The slice is always
// non-nil so it marshals as [] and never null.
func (c *conv) inline(parent ast.Node) []Node {
	out := []Node{}
	var buf strings.Builder
	flush := func() {
		if buf.Len() > 0 {
			out = append(out, &Text{Type: "text", Value: buf.String()})
			buf.Reset()
		}
	}
	for ch := parent.FirstChild(); ch != nil; ch = ch.NextSibling() {
		switch t := ch.(type) {
		case *ast.Text:
			buf.WriteString(decodeText(t.Segment.Value(c.src)))
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
			if n := c.inlineNode(ch); n != nil {
				out = append(out, n)
			}
		}
	}
	flush()
	return out
}

// inlineNode converts one non-text inline node. A goldmark footnote backlink —
// the return arrow goldmark adds to a definition — has no case and is dropped,
// which is right: mdast carries no backlink.
func (c *conv) inlineNode(n ast.Node) Node {
	switch i := n.(type) {
	case *ast.Emphasis:
		if i.Level == 2 {
			return &Strong{Type: "strong", Children: c.inline(i)}
		}
		return &Emphasis{Type: "emphasis", Children: c.inline(i)}
	case *east.Strikethrough:
		return &Delete{Type: "delete", Children: c.inline(i)}
	case *ast.CodeSpan:
		return &InlineCode{Type: "inlineCode", Value: codeSpanValue(i, c.src)}
	case *ast.Link:
		return c.link(i)
	case *ast.AutoLink:
		return c.autoLink(i)
	case *ast.Image:
		return c.image(i)
	case *ast.RawHTML:
		return &HTML{Type: "html", Value: rawHTMLValue(i, c.src)}
	case *east.FootnoteLink:
		label := c.footnotes[i.Index]
		return &FootnoteReference{
			Type:       "footnoteReference",
			Identifier: normalizeIdentifier(label),
			Label:      label,
		}
	}
	return nil
}

// link converts an inline link, or a reference link when it resolves to a
// definition.
func (c *conv) link(l *ast.Link) Node {
	if l.Reference != nil {
		label := string(l.Reference.Value)
		return &LinkReference{
			Type:          "linkReference",
			Identifier:    normalizeIdentifier(label),
			Label:         label,
			ReferenceType: refType(l.Reference.Type),
			Children:      c.inline(l),
		}
	}
	return &Link{
		Type:     "link",
		URL:      string(l.Destination),
		Title:    optString(l.Title),
		Children: c.inline(l),
	}
}

// autoLink converts an autolink (an angle-bracket link or a GFM bare URL) into a
// link whose single text child is the shown URL. An email autolink gets the
// mailto: scheme, which goldmark leaves off the URL.
func (c *conv) autoLink(a *ast.AutoLink) Node {
	url := string(a.URL(c.src))
	if a.AutoLinkType == ast.AutoLinkEmail {
		url = "mailto:" + url
	}
	return &Link{
		Type:     "link",
		URL:      url,
		Title:    nil,
		Children: []Node{&Text{Type: "text", Value: string(a.Label(c.src))}},
	}
}

// image converts an inline image, or an image reference when it resolves to a
// definition.
func (c *conv) image(img *ast.Image) Node {
	alt := textContent(img, c.src)
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
