package sanitize

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/charmbracelet/x/ansi"
)

const tabWidth = 4

func Text(s string) string {
	s = strings.ToValidUTF8(s, string(utf8.RuneError))
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = ansi.Strip(s)

	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch {
		case r == '\n':
			b.WriteRune(r)
		case r == '\t':
			b.WriteString(strings.Repeat(" ", tabWidth))
		case unicode.In(r, unicode.Zl, unicode.Zp):
			b.WriteRune('\n')
		case unicode.IsControl(r), unicode.Is(unicode.Bidi_Control, r), isInvisible(r):
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

func isInvisible(r rune) bool {
	return (r >= 0xE0000 && r <= 0xE007F) || r == 0x200B || r == 0x2060 || r == 0xFEFF
}

func Line(s string) string {
	return strings.Join(strings.Fields(Text(s)), " ")
}
