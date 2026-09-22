package mdast

import (
	"strconv"

	"github.com/yuin/goldmark/util"
)

// decodeText resolves a raw text segment the way remark does: it drops
// backslash escapes before ASCII punctuation and decodes HTML character
// references (named, decimal, and hex). It is a port of goldmark's
// html.defaultWriter.Write state machine, but it emits the decoded bytes
// literally instead of HTML-escaping them, because an mdast text value holds
// the decoded characters, not their HTML representation.
//
// It must not be used on raw content — code spans, code blocks, and raw HTML —
// where backslashes and ampersands stay literal.
func decodeText(source []byte) string {
	out := make([]byte, 0, len(source))
	escaped := false
	var ok bool
	limit := len(source)
	n := 0
	for i := 0; i < limit; i++ {
		c := source[i]
		if escaped {
			if util.IsPunct(c) {
				out = append(out, source[n:i-1]...)
				n = i
				escaped = false
				continue
			}
		}
		if c == '&' {
			pos := i
			next := i + 1
			if next < limit && source[next] == '#' {
				nnext := next + 1
				if nnext < limit {
					nc := source[nnext]
					if nc == 'x' || nc == 'X' {
						start := nnext + 1
						i, ok = util.ReadWhile(source, [2]int{start, limit}, util.IsHexDecimal)
						if ok && i < limit && source[i] == ';' && i-start < 7 {
							v, _ := strconv.ParseUint(string(source[start:i]), 16, 32)
							out = append(out, source[n:pos]...)
							n = i + 1
							out = appendRune(out, rune(v))
							continue
						}
					} else if nc >= '0' && nc <= '9' {
						start := nnext
						i, ok = util.ReadWhile(source, [2]int{start, limit}, util.IsNumeric)
						if ok && i < limit && i-start < 8 && source[i] == ';' {
							v, _ := strconv.ParseUint(string(source[start:i]), 10, 32)
							out = append(out, source[n:pos]...)
							n = i + 1
							out = appendRune(out, rune(v))
							continue
						}
					}
				}
			} else {
				start := next
				i, ok = util.ReadWhile(source, [2]int{start, limit}, util.IsAlphaNumeric)
				if ok && i < limit && source[i] == ';' {
					name := string(source[start:i])
					if entity, found := util.LookUpHTML5EntityByName(name); found {
						out = append(out, source[n:pos]...)
						n = i + 1
						out = append(out, entity.Characters...)
						continue
					}
				}
			}
			i = next - 1
		}
		if c == '\\' {
			escaped = true
			continue
		}
		escaped = false
	}
	out = append(out, source[n:]...)
	return string(out)
}

// appendRune appends r, replacing an invalid rune with the Unicode replacement
// character, matching goldmark's handling of out-of-range references.
func appendRune(dst []byte, r rune) []byte {
	return append(dst, []byte(string(util.ToValidRune(r)))...)
}
