package editor

import (
	"fmt"
	"io"
	"strings"
	"unicode"
)

type Renderer struct {
	out io.Writer
}

type runeInterval struct {
	first rune
	last  rune
}

var wideRuneIntervals = []runeInterval{
	{first: 0x1100, last: 0x115f},
	{first: 0x2329, last: 0x232a},
	{first: 0x2e80, last: 0xa4cf},
	{first: 0xac00, last: 0xd7a3},
	{first: 0xf900, last: 0xfaff},
	{first: 0xfe10, last: 0xfe19},
	{first: 0xfe30, last: 0xfe6f},
	{first: 0xff00, last: 0xff60},
	{first: 0xffe0, last: 0xffe6},
	{first: 0x1f300, last: 0x1faff},
	{first: 0x20000, last: 0x2fffd},
	{first: 0x30000, last: 0x3fffd},
}

func NewRenderer(out io.Writer) *Renderer {
	return &Renderer{out: out}
}

func (r *Renderer) ClearLine() {
	fmt.Fprint(r.out, "\r\033[2K")
}

func (r *Renderer) Redraw(prompt string, editor *Editor) {
	fmt.Fprint(r.out, RenderSingleLine(prompt, editor.Buffer(), editor.Position()))
}

func RenderSingleLine(prompt string, buffer Buffer, cursor Position) string {
	line := buffer.Line(cursor.Line)
	column := clamp(cursor.Column, 0, len(line))
	var builder strings.Builder
	builder.WriteString("\r\033[2K")
	builder.WriteString(prompt)
	builder.WriteString(string(line))
	if moveLeft := displayWidthRunes(line[column:]); moveLeft > 0 {
		fmt.Fprintf(&builder, "\033[%dD", moveLeft)
	}
	return builder.String()
}

func displayWidthRunes(runes []rune) int {
	width := 0
	for _, value := range runes {
		width += runeDisplayWidth(value)
	}
	return width
}

func runeDisplayWidth(value rune) int {
	switch {
	case value == 0:
		return 0
	case value < 32 || (value >= 0x7f && value < 0xa0):
		return 0
	case unicode.Is(unicode.Mn, value), unicode.Is(unicode.Me, value), unicode.Is(unicode.Cf, value):
		return 0
	case isWideRune(value):
		return 2
	default:
		return 1
	}
}

func isWideRune(value rune) bool {
	for _, interval := range wideRuneIntervals {
		if value >= interval.first && value <= interval.last {
			return true
		}
	}
	return false
}
