package hast

import (
	"strings"
	"unicode/utf8"
)

// trimLines removes spaces and tabs around the line breaks inside value, but
// not at value's own start or end. It is a port of the `trim-lines` package
// (mdast-util-to-hast's text handler runs a text node's value through it).
func trimLines(value string) string {
	var b strings.Builder
	last := 0
	for i := 0; i < len(value); {
		// Match a line ending: \r\n, \r, or \n.
		size := lineEndingAt(value, i)
		if size == 0 {
			i++
			continue
		}
		b.WriteString(trimLine(value[last:i], last > 0, true))
		b.WriteString(value[i : i+size])
		i += size
		last = i
	}
	b.WriteString(trimLine(value[last:], last > 0, false))
	return b.String()
}

// lineEndingAt returns the byte length of the line ending at index i (\r\n → 2,
// \r or \n → 1), or 0 if there is none.
func lineEndingAt(value string, i int) int {
	switch value[i] {
	case '\r':
		if i+1 < len(value) && value[i+1] == '\n' {
			return 2
		}
		return 1
	case '\n':
		return 1
	}
	return 0
}

// trimLine trims spaces and tabs from a line's start (when start is true) and
// end (when end is true).
func trimLine(value string, start, end bool) string {
	lo, hi := 0, len(value)
	if start {
		for lo < hi && (value[lo] == ' ' || value[lo] == '\t') {
			lo++
		}
	}
	if end {
		for hi > lo && (value[hi-1] == ' ' || value[hi-1] == '\t') {
			hi--
		}
	}
	return value[lo:hi]
}

// trimMarkdownSpaceStart trims leading spaces and tabs, matching
// mdast-util-to-hast's helper of the same name (used after a hard break).
func trimMarkdownSpaceStart(value string) string {
	i := 0
	for i < len(value) && (value[i] == ' ' || value[i] == '\t') {
		i++
	}
	return value[i:]
}

// replaceLineEndings replaces every line ending (\r\n, \r, or \n) with a single
// space, matching mdast-util-to-hast's inline-code handler.
func replaceLineEndings(value string) string {
	var b strings.Builder
	b.Grow(len(value))
	for i := 0; i < len(value); {
		if size := lineEndingAt(value, i); size != 0 {
			b.WriteByte(' ')
			i += size
			continue
		}
		b.WriteByte(value[i])
		i++
	}
	return b.String()
}

// normalizeURI encodes unsafe characters in a URL with percent-encoding,
// skipping already-encoded sequences. It is a port of
// micromark-util-sanitize-uri's normalizeUri (the function link and image
// handlers call — no protocol filtering). Unlike the JavaScript, which iterates
// UTF-16 code units, this iterates bytes for ASCII and whole runes above it;
// the percent-encoded output is identical for valid Unicode, and an invalid
// UTF-8 byte becomes the replacement character's encoding, as a lone surrogate
// does in the original.
func normalizeURI(value string) string {
	var b strings.Builder
	i := 0
	for i < len(value) {
		ch := value[i]
		// A correct percent-encoded value: keep it verbatim.
		if ch == '%' && i+2 < len(value) && isASCIIAlnum(value[i+1]) && isASCIIAlnum(value[i+2]) {
			b.WriteByte('%')
			i++
			continue
		}
		if ch < 0x80 {
			if isAllowedURIByte(ch) {
				b.WriteByte(ch)
			} else {
				percentEncode(&b, []byte{ch})
			}
			i++
			continue
		}
		r, size := utf8.DecodeRuneInString(value[i:])
		if r == utf8.RuneError && size == 1 {
			percentEncode(&b, []byte("�"))
		} else {
			percentEncode(&b, []byte(string(r)))
		}
		i += size
	}
	return b.String()
}

// isAllowedURIByte reports whether an ASCII byte is left unescaped by
// normalizeUri — the character class /[!#$&-;=?-Z_a-z~]/.
func isAllowedURIByte(c byte) bool {
	switch {
	case c == '!', c == '#', c == '$', c == '=', c == '_', c == '~':
		return true
	case c >= '&' && c <= ';': // 0x26..0x3B
		return true
	case c >= '?' && c <= 'Z': // 0x3F..0x5A
		return true
	case c >= 'a' && c <= 'z':
		return true
	}
	return false
}

// isASCIIAlnum reports whether c is an ASCII letter or digit.
func isASCIIAlnum(c byte) bool {
	return (c >= '0' && c <= '9') || (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z')
}

// percentEncode writes each byte of p as an uppercase %XX escape, matching
// JavaScript's encodeURIComponent for the characters normalizeUri passes to it.
func percentEncode(b *strings.Builder, p []byte) {
	const hex = "0123456789ABCDEF"
	for _, c := range p {
		b.WriteByte('%')
		b.WriteByte(hex[c>>4])
		b.WriteByte(hex[c&0x0f])
	}
}
