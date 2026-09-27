package hast

import (
	"strconv"
	"strings"

	"github.com/zoobz-io/barbara/internal/mdast"
)

// clobberPrefix prefixes every footnote id and fragment. It matches
// mdast-util-to-hast's default and guards against DOM clobbering, where an
// element id shadows a global on `window`.
const clobberPrefix = "user-content-"

// FromMdast converts an mdast tree into a hast tree, reproducing
// mdast-util-to-hast (at the version pinned in web/) byte for byte. The two
// option deviations — raw HTML passed through as `raw` nodes, and a fenced
// code block's meta kept as `dataMeta` — are baked in; see the package doc.
func FromMdast(root *mdast.MdastRoot) *HastRoot {
	c := &conv{
		definitionByID: map[string]*mdast.MdastDefinition{},
		footnoteByID:   map[string]*mdast.MdastFootnoteDefinition{},
		footnoteCounts: map[string]int{},
	}
	c.collect(root.Children)

	// Build the tree first: converting footnote references populates the order
	// and counts the footer reads, so the footer must run afterwards.
	result := c.root(root)
	if foot := c.footer(); foot != nil {
		result.Children = append(result.Children, &HastText{Type: "text", Value: "\n"}, foot)
	}
	return result
}

// conv carries the per-conversion state: the link and footnote definitions
// (collected up front, keyed by uppercased identifier, first occurrence wins),
// and the footnote reference order and per-identifier reference counts (filled
// as references are converted).
type conv struct {
	definitionByID map[string]*mdast.MdastDefinition
	footnoteByID   map[string]*mdast.MdastFootnoteDefinition
	footnoteCounts map[string]int
	footnoteOrder  []string
}

// collect walks the tree and records every definition and footnote definition
// by uppercased identifier, keeping the first occurrence of each — the behavior
// mdast-util-definitions documents and CommonMark follows.
func (c *conv) collect(nodes []mdast.Node) {
	for _, n := range nodes {
		switch v := n.(type) {
		case *mdast.MdastDefinition:
			id := strings.ToUpper(v.Identifier)
			if _, ok := c.definitionByID[id]; !ok {
				c.definitionByID[id] = v
			}
		case *mdast.MdastFootnoteDefinition:
			id := strings.ToUpper(v.Identifier)
			if _, ok := c.footnoteByID[id]; !ok {
				c.footnoteByID[id] = v
			}
			c.collect(v.Children)
		default:
			c.collect(childrenOf(n))
		}
	}
}

// childrenOf returns a node's children, or nil for a leaf.
func childrenOf(n mdast.Node) []mdast.Node {
	switch v := n.(type) {
	case *mdast.MdastParagraph:
		return v.Children
	case *mdast.MdastHeading:
		return v.Children
	case *mdast.MdastBlockquote:
		return v.Children
	case *mdast.MdastList:
		return v.Children
	case *mdast.MdastListItem:
		return v.Children
	case *mdast.MdastEmphasis:
		return v.Children
	case *mdast.MdastStrong:
		return v.Children
	case *mdast.MdastDelete:
		return v.Children
	case *mdast.MdastLink:
		return v.Children
	case *mdast.MdastLinkReference:
		return v.Children
	case *mdast.MdastTable:
		return v.Children
	case *mdast.MdastTableRow:
		return v.Children
	case *mdast.MdastTableCell:
		return v.Children
	}
	return nil
}

// all converts a run of sibling mdast nodes, passing parent (their mdast parent)
// to each — a table row and a list item read it. It mirrors mdast-util-to-hast's
// `all`: a node that follows a hard break has the leading markdown whitespace
// trimmed from its first text (a break already ends the line), and a handler
// that returns several nodes is spread in place. The result is never nil, so an
// element with no children still marshals as `[]`.
func (c *conv) all(nodes []mdast.Node, parent mdast.Node) []HastNode {
	values := []HastNode{}
	for i, node := range nodes {
		results, multiple := c.one(node, parent)
		if len(results) == 0 {
			continue
		}
		if i > 0 && !multiple {
			if _, isBreak := nodes[i-1].(*mdast.MdastBreak); isBreak {
				trimHeadSpace(results[0])
			}
		}
		values = append(values, results...)
	}
	return values
}

// trimHeadSpace trims leading markdown whitespace from a text node, or from the
// first child of an element node.
func trimHeadSpace(n HastNode) {
	switch v := n.(type) {
	case *HastText:
		v.Value = trimMarkdownSpaceStart(v.Value)
	case *HastElement:
		if len(v.Children) > 0 {
			if head, ok := v.Children[0].(*HastText); ok {
				head.Value = trimMarkdownSpaceStart(head.Value)
			}
		}
	}
}

// one converts a single mdast node. It returns the resulting hast nodes and
// whether the handler produced several of them (a hard break, or a reverted
// reference) — `all` uses that to match mdast-util-to-hast's break trimming.
// An ignored node (a definition, a footnote definition) returns no nodes.
func (c *conv) one(node mdast.Node, parent mdast.Node) (out []HastNode, multiple bool) {
	switch n := node.(type) {
	case *mdast.MdastParagraph:
		return single(c.element("p", nil, c.all(n.Children, n)))
	case *mdast.MdastHeading:
		return single(c.element("h"+strconv.Itoa(n.Depth), nil, c.all(n.Children, n)))
	case *mdast.MdastThematicBreak:
		return single(c.element("hr", nil, nil))
	case *mdast.MdastBlockquote:
		return single(c.element("blockquote", nil, wrap(c.all(n.Children, n), true)))
	case *mdast.MdastList:
		return single(c.list(n))
	case *mdast.MdastListItem:
		list, _ := parent.(*mdast.MdastList)
		return single(c.listItem(n, list))
	case *mdast.MdastCode:
		return single(c.code(n))
	case *mdast.MdastHTML:
		return single(&HastRaw{Type: "raw", Value: n.Value})
	case *mdast.MdastText:
		return single(&HastText{Type: "text", Value: trimLines(n.Value)})
	case *mdast.MdastEmphasis:
		return single(c.element("em", nil, c.all(n.Children, n)))
	case *mdast.MdastStrong:
		return single(c.element("strong", nil, c.all(n.Children, n)))
	case *mdast.MdastDelete:
		return single(c.element("del", nil, c.all(n.Children, n)))
	case *mdast.MdastInlineCode:
		text := &HastText{Type: "text", Value: replaceLineEndings(n.Value)}
		return single(c.element("code", nil, []HastNode{text}))
	case *mdast.MdastBreak:
		return []HastNode{c.element("br", nil, nil), &HastText{Type: "text", Value: "\n"}}, true
	case *mdast.MdastLink:
		return single(c.link(n))
	case *mdast.MdastImage:
		return single(c.image(n))
	case *mdast.MdastLinkReference:
		return c.linkReference(n)
	case *mdast.MdastImageReference:
		return c.imageReference(n)
	case *mdast.MdastFootnoteReference:
		return single(c.footnoteReference(n))
	case *mdast.MdastDefinition, *mdast.MdastFootnoteDefinition:
		return nil, false // collected, never rendered inline
	case *mdast.MdastTable:
		return single(c.table(n))
	case *mdast.MdastTableRow:
		table, _ := parent.(*mdast.MdastTable)
		return single(c.tableRow(n, table))
	case *mdast.MdastTableCell:
		return single(c.element("td", nil, c.all(n.Children, n)))
	}
	return nil, false
}

// single wraps one node as the (nodes, multiple) pair one returns.
func single(n HastNode) ([]HastNode, bool) {
	return []HastNode{n}, false
}

// element builds an element node. A nil properties map is stored as an empty
// map so it marshals as `{}`, and nil children as an empty slice so they
// marshal as `[]` — both matching mdast-util-to-hast's output.
func (c *conv) element(tagName string, properties map[string]any, children []HastNode) *HastElement {
	if properties == nil {
		properties = map[string]any{}
	}
	if children == nil {
		children = []HastNode{}
	}
	return &HastElement{Type: "element", TagName: tagName, Properties: properties, Children: children}
}

// root converts the document root: block children wrapped with line endings
// between them (but not around them).
func (c *conv) root(node *mdast.MdastRoot) *HastRoot {
	return &HastRoot{Type: "root", Children: wrap(c.all(node.Children, node), false)}
}

// code converts a fenced or indented code block into `<pre><code>`. It keeps
// the language as a `language-<lang>` class and, unlike the default mapping,
// keeps the fence meta as the `dataMeta` property (rendered `data-meta`).
func (c *conv) code(node *mdast.MdastCode) *HastElement {
	value := ""
	if node.Value != "" {
		value = node.Value + "\n"
	}
	properties := map[string]any{}
	if node.Lang != nil && *node.Lang != "" {
		properties["className"] = []string{"language-" + *node.Lang}
	}
	if node.Meta != nil && *node.Meta != "" {
		properties["dataMeta"] = *node.Meta
	}
	code := c.element("code", properties, []HastNode{&HastText{Type: "text", Value: value}})
	return c.element("pre", nil, []HastNode{code})
}

// list converts an ordered or unordered list. It carries an explicit `start`
// when the ordered start is not 1, and adds the `contains-task-list` class when
// any item is a task item — matching GitHub's rendering.
func (c *conv) list(node *mdast.MdastList) *HastElement {
	results := c.all(node.Children, node)
	properties := map[string]any{}
	if node.Ordered && node.Start != nil && *node.Start != 1 {
		properties["start"] = *node.Start
	}
	for _, child := range results {
		if hasClass(child, "task-list-item") {
			properties["className"] = []string{"contains-task-list"}
			break
		}
	}
	tagName := "ul"
	if node.Ordered {
		tagName = "ol"
	}
	return c.element(tagName, properties, wrap(results, true))
}

// hasClass reports whether n is an element whose className list contains cls.
func hasClass(n HastNode, cls string) bool {
	el, ok := n.(*HastElement)
	if !ok {
		return false
	}
	classes, ok := el.Properties["className"].([]string)
	if !ok {
		return false
	}
	for _, c := range classes {
		if c == cls {
			return true
		}
	}
	return false
}

// listItem converts a list item. A task item gets a leading disabled checkbox
// input and the `task-list-item` class. In a tight list the item's first
// paragraph is unwrapped (its inline children hoisted straight into the `<li>`);
// in a loose list every block keeps its wrapper and line endings surround them.
func (c *conv) listItem(node *mdast.MdastListItem, parent *mdast.MdastList) *HastElement {
	results := c.all(node.Children, node)
	loose := listItemLoose(node)
	if parent != nil {
		loose = listLoose(parent)
	}
	properties := map[string]any{}

	if node.Checked != nil {
		var paragraph *HastElement
		if len(results) > 0 {
			if p, ok := results[0].(*HastElement); ok && p.TagName == "p" {
				paragraph = p
			}
		}
		if paragraph == nil {
			paragraph = c.element("p", nil, nil)
			results = append([]HastNode{paragraph}, results...)
		}
		if len(paragraph.Children) > 0 {
			paragraph.Children = append([]HastNode{&HastText{Type: "text", Value: " "}}, paragraph.Children...)
		}
		input := c.element("input", map[string]any{"type": "checkbox", "checked": *node.Checked, "disabled": true}, nil)
		paragraph.Children = append([]HastNode{input}, paragraph.Children...)
		properties["className"] = []string{"task-list-item"}
	}

	children := []HastNode{}
	for i, child := range results {
		el, isEl := child.(*HastElement)
		isTightP := isEl && el.TagName == "p" && !loose
		// A line ending precedes every child except a tight list's first paragraph.
		if loose || i != 0 || !isEl || el.TagName != "p" {
			children = append(children, &HastText{Type: "text", Value: "\n"})
		}
		if isTightP {
			children = append(children, el.Children...)
		} else {
			children = append(children, child)
		}
	}
	if n := len(results); n > 0 {
		tail := results[n-1]
		el, isEl := tail.(*HastElement)
		if !isEl || el.TagName != "p" || loose {
			children = append(children, &HastText{Type: "text", Value: "\n"})
		}
	}
	return c.element("li", properties, children)
}

// listItemLoose reports whether a list item is loose. mdast-util-to-hast falls
// back to "more than one child" only when spread is unset; internal/mdast always
// sets spread (faithful to remark), so the recorded value is authoritative.
func listItemLoose(node *mdast.MdastListItem) bool {
	return node.Spread
}

// listLoose reports whether a list is loose: its own spread, or any loose item.
func listLoose(node *mdast.MdastList) bool {
	if node.Spread {
		return true
	}
	for _, child := range node.Children {
		if li, ok := child.(*mdast.MdastListItem); ok && listItemLoose(li) {
			return true
		}
	}
	return false
}

// link converts an inline link or autolink into an `<a>`.
func (c *conv) link(node *mdast.MdastLink) *HastElement {
	properties := map[string]any{"href": normalizeURI(node.URL)}
	if node.Title != nil {
		properties["title"] = *node.Title
	}
	return c.element("a", properties, c.all(node.Children, node))
}

// image converts an inline image into an `<img>`.
func (c *conv) image(node *mdast.MdastImage) *HastElement {
	properties := map[string]any{"src": normalizeURI(node.URL), "alt": node.Alt}
	if node.Title != nil {
		properties["title"] = *node.Title
	}
	return c.element("img", properties, nil)
}

// linkReference resolves a link reference against its definition, or reverts to
// the source text when the definition is missing.
func (c *conv) linkReference(node *mdast.MdastLinkReference) ([]HastNode, bool) {
	def := c.definitionByID[strings.ToUpper(node.Identifier)]
	if def == nil {
		return c.revertLink(node), true
	}
	properties := map[string]any{"href": normalizeURI(def.URL)}
	if def.Title != nil {
		properties["title"] = *def.Title
	}
	return single(c.element("a", properties, c.all(node.Children, node)))
}

// imageReference resolves an image reference against its definition, or reverts
// to the source text when the definition is missing.
func (c *conv) imageReference(node *mdast.MdastImageReference) ([]HastNode, bool) {
	def := c.definitionByID[strings.ToUpper(node.Identifier)]
	if def == nil {
		return []HastNode{&HastText{Type: "text", Value: "![" + node.Alt + referenceSuffix(node.ReferenceType, node.Label, node.Identifier)}}, true
	}
	properties := map[string]any{"src": normalizeURI(def.URL), "alt": node.Alt}
	if def.Title != nil {
		properties["title"] = *def.Title
	}
	return single(c.element("img", properties, nil))
}

// revertLink renders a link reference without a definition as its literal
// source: the bracketed label plus the reference suffix.
func (c *conv) revertLink(node *mdast.MdastLinkReference) []HastNode {
	suffix := referenceSuffix(node.ReferenceType, node.Label, node.Identifier)
	contents := c.all(node.Children, node)
	if len(contents) > 0 {
		if head, ok := contents[0].(*HastText); ok {
			head.Value = "[" + head.Value
		} else {
			contents = append([]HastNode{&HastText{Type: "text", Value: "["}}, contents...)
		}
	} else {
		contents = append(contents, &HastText{Type: "text", Value: "["})
	}
	if tail, ok := contents[len(contents)-1].(*HastText); ok {
		tail.Value += suffix
	} else {
		contents = append(contents, &HastText{Type: "text", Value: suffix})
	}
	return contents
}

// referenceSuffix reproduces the closing part of a reverted reference: "]" for a
// shortcut reference, "][]" for a collapsed one, and "][label]" for a full one.
func referenceSuffix(referenceType, label, identifier string) string {
	switch referenceType {
	case "collapsed":
		return "][]"
	case "full":
		text := label
		if text == "" {
			text = identifier
		}
		return "][" + text + "]"
	default:
		return "]"
	}
}

// footnoteReference converts an inline footnote reference into a `<sup>` linking
// to the footnote in the footer. It records the reference order (first use) and
// counts repeat uses, both of which the footer reads to build back references.
func (c *conv) footnoteReference(node *mdast.MdastFootnoteReference) *HastElement {
	id := strings.ToUpper(node.Identifier)
	safeID := normalizeURI(strings.ToLower(id))

	count, seen := c.footnoteCounts[id]
	var counter int
	if !seen {
		c.footnoteOrder = append(c.footnoteOrder, id)
		counter = len(c.footnoteOrder)
	} else {
		counter = indexOf(c.footnoteOrder, id) + 1
	}
	count++
	c.footnoteCounts[id] = count

	refID := clobberPrefix + "fnref-" + safeID
	if count > 1 {
		refID += "-" + strconv.Itoa(count)
	}
	link := c.element("a", map[string]any{
		"href":            "#" + clobberPrefix + "fn-" + safeID,
		"id":              refID,
		"dataFootnoteRef": true,
		"ariaDescribedBy": []string{"footnote-label"},
	}, []HastNode{&HastText{Type: "text", Value: strconv.Itoa(counter)}})
	return c.element("sup", nil, []HastNode{link})
}

// footer builds the footnotes section appended to the document — one `<li>` per
// referenced definition, in reference order, each ending with a back reference
// per use. It returns nil when no footnote was referenced.
func (c *conv) footer() *HastElement {
	listItems := []HastNode{}
	for referenceIndex, id := range c.footnoteOrder {
		def := c.footnoteByID[id]
		if def == nil {
			continue
		}
		content := c.all(def.Children, def)
		safeID := normalizeURI(strings.ToLower(id))
		counts := c.footnoteCounts[id]

		backReferences := []HastNode{}
		for rereferenceIndex := 1; rereferenceIndex <= counts; rereferenceIndex++ {
			if len(backReferences) > 0 {
				backReferences = append(backReferences, &HastText{Type: "text", Value: " "})
			}
			href := "#" + clobberPrefix + "fnref-" + safeID
			ariaLabel := "Back to reference " + strconv.Itoa(referenceIndex+1)
			if rereferenceIndex > 1 {
				href += "-" + strconv.Itoa(rereferenceIndex)
				ariaLabel += "-" + strconv.Itoa(rereferenceIndex)
			}
			backReferences = append(backReferences, c.element("a", map[string]any{
				"href":                href,
				"dataFootnoteBackref": "",
				"ariaLabel":           ariaLabel,
				"className":           []string{"data-footnote-backref"},
			}, backContent(rereferenceIndex)))
		}

		content = appendBackReferences(content, backReferences)
		listItems = append(listItems, c.element("li",
			map[string]any{"id": clobberPrefix + "fn-" + safeID},
			wrap(content, true)))
	}

	if len(listItems) == 0 {
		return nil
	}
	label := c.element("h2", map[string]any{"className": []string{"sr-only"}, "id": "footnote-label"},
		[]HastNode{&HastText{Type: "text", Value: "Footnotes"}})
	ol := c.element("ol", nil, wrap(listItems, true))
	return c.element("section",
		map[string]any{"dataFootnotes": true, "className": []string{"footnotes"}},
		[]HastNode{label, &HastText{Type: "text", Value: "\n"}, ol, &HastText{Type: "text", Value: "\n"}})
}

// backContent is GitHub's default back-reference content: a return arrow, plus a
// superscript counter when the same footnote is referenced more than once.
func backContent(rereferenceIndex int) []HastNode {
	result := []HastNode{&HastText{Type: "text", Value: "↩"}}
	if rereferenceIndex > 1 {
		result = append(result, &HastElement{
			Type: "element", TagName: "sup", Properties: map[string]any{},
			Children: []HastNode{&HastText{Type: "text", Value: strconv.Itoa(rereferenceIndex)}},
		})
	}
	return result
}

// appendBackReferences appends the back references to a footnote's content. When
// the content ends in a paragraph they go inside it, after a separating space;
// otherwise they are appended as new blocks.
func appendBackReferences(content, backReferences []HastNode) []HastNode {
	if n := len(content); n > 0 {
		if tail, ok := content[n-1].(*HastElement); ok && tail.TagName == "p" {
			if m := len(tail.Children); m > 0 {
				if tailTail, ok := tail.Children[m-1].(*HastText); ok {
					tailTail.Value += " "
				} else {
					tail.Children = append(tail.Children, &HastText{Type: "text", Value: " "})
				}
			} else {
				tail.Children = append(tail.Children, &HastText{Type: "text", Value: " "})
			}
			tail.Children = append(tail.Children, backReferences...)
			return content
		}
	}
	return append(content, backReferences...)
}

// table converts a GFM table into `<table>` with a `<thead>` (the first row) and,
// when there are more rows, a `<tbody>`.
func (c *conv) table(node *mdast.MdastTable) *HastElement {
	rows := c.all(node.Children, node)
	tableContent := []HastNode{}
	if len(rows) > 0 {
		head := c.element("thead", nil, wrap([]HastNode{rows[0]}, true))
		tableContent = append(tableContent, head)
		rows = rows[1:]
	}
	if len(rows) > 0 {
		body := c.element("tbody", nil, wrap(rows, true))
		tableContent = append(tableContent, body)
	}
	return c.element("table", nil, wrap(tableContent, true))
}

// tableRow converts a table row. Header rows (the first, when a parent table is
// known) use `<th>`; body rows use `<td>`. Cells are padded or truncated to the
// column count and carry the column alignment.
func (c *conv) tableRow(node *mdast.MdastTableRow, table *mdast.MdastTable) *HastElement {
	tagName := "td"
	var align []*string
	if table != nil {
		if indexOfRow(table.Children, node) == 0 {
			tagName = "th"
		}
		align = table.Align
	}
	length := len(node.Children)
	if align != nil {
		length = len(align)
	}
	cells := []HastNode{}
	for cellIndex := 0; cellIndex < length; cellIndex++ {
		properties := map[string]any{}
		if align != nil && cellIndex < len(align) && align[cellIndex] != nil {
			properties["align"] = *align[cellIndex]
		}
		var children []HastNode
		if cellIndex < len(node.Children) {
			if cell, ok := node.Children[cellIndex].(*mdast.MdastTableCell); ok {
				children = c.all(cell.Children, cell)
			}
		}
		cells = append(cells, c.element(tagName, properties, children))
	}
	return c.element("tr", nil, wrap(cells, true))
}

// indexOfRow returns the position of row among a table's children, or -1.
func indexOfRow(rows []mdast.Node, row *mdast.MdastTableRow) int {
	for i, r := range rows {
		if r == row {
			return i
		}
	}
	return -1
}

// indexOf returns the position of s in xs, or -1.
func indexOf(xs []string, s string) int {
	for i, x := range xs {
		if x == s {
			return i
		}
	}
	return -1
}

// wrap joins nodes with newline text nodes between each; when loose, it also
// adds a newline before the first and after the last. It mirrors
// mdast-util-to-hast's `wrap` and never returns nil.
func wrap(nodes []HastNode, loose bool) []HastNode {
	result := []HastNode{}
	if loose {
		result = append(result, &HastText{Type: "text", Value: "\n"})
	}
	for i, n := range nodes {
		if i > 0 {
			result = append(result, &HastText{Type: "text", Value: "\n"})
		}
		result = append(result, n)
	}
	if loose && len(nodes) > 0 {
		result = append(result, &HastText{Type: "text", Value: "\n"})
	}
	return result
}
