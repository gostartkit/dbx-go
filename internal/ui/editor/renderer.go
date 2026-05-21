package editor

import (
	"fmt"
	"io"
	"strings"
	"unicode"
)

type Renderer struct {
	out  io.Writer
	rows int
}

type Cell struct {
	Rune  rune
	Width int
}

type ScreenLine struct {
	Cells []Cell
}

type RenderFrame struct {
	Lines     []ScreenLine
	CursorRow int
	CursorCol int
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

func (r *Renderer) Redraw(prompt string, continuationPrompt string, editor *Editor) {
	frame := BuildRenderFrame(prompt, continuationPrompt, editor.Buffer(), editor.Position())
	r.WriteFrame(frame)
}

func (r *Renderer) WriteFrame(frame RenderFrame) {
	if r.rows > 1 {
		fmt.Fprintf(r.out, "\r\033[%dA", r.rows-1)
	}
	clearRows := r.rows
	if clearRows == 0 {
		clearRows = 1
	}
	for row := 0; row < clearRows; row++ {
		fmt.Fprint(r.out, "\r\033[2K")
		if row < clearRows-1 {
			fmt.Fprint(r.out, "\033[1B")
		}
	}
	if clearRows > 1 {
		fmt.Fprintf(r.out, "\r\033[%dA", clearRows-1)
	}

	for idx, line := range frame.Lines {
		for _, cell := range line.Cells {
			fmt.Fprint(r.out, string(cell.Rune))
		}
		if idx < len(frame.Lines)-1 {
			fmt.Fprint(r.out, "\r\n")
		}
	}

	rows := len(frame.Lines)
	if rows == 0 {
		rows = 1
	}
	if rows == 1 {
		lineWidth := 0
		if len(frame.Lines) > 0 {
			for _, cell := range frame.Lines[0].Cells {
				lineWidth += cell.Width
			}
		}
		back := lineWidth - frame.CursorCol
		if back > 0 {
			fmt.Fprintf(r.out, "\033[%dD", back)
		}
		r.rows = rows
		return
	}
	up := rows - 1 - frame.CursorRow
	if up > 0 {
		fmt.Fprintf(r.out, "\033[%dA", up)
	}
	fmt.Fprint(r.out, "\r")
	if frame.CursorCol > 0 {
		fmt.Fprintf(r.out, "\033[%dC", frame.CursorCol)
	}
	r.rows = rows
}

func BuildRenderFrame(prompt string, continuationPrompt string, buffer Buffer, cursor Position) RenderFrame {
	lines := make([]ScreenLine, 0, len(buffer.Lines))
	cursorRow := clamp(cursor.Line, 0, len(buffer.Lines)-1)
	cursorCol := 0
	for idx, line := range buffer.Lines {
		prefix := prompt
		if idx > 0 {
			prefix = continuationPrompt
		}
		screenLine := ScreenLine{Cells: buildCells([]rune(prefix))}
		screenLine.Cells = append(screenLine.Cells, buildCells(line)...)
		lines = append(lines, screenLine)
		if idx == cursorRow {
			column := clamp(cursor.Column, 0, len(line))
			cursorCol = displayWidthRunes([]rune(prefix)) + displayWidthRunes(line[:column])
		}
	}
	if len(lines) == 0 {
		lines = append(lines, ScreenLine{Cells: buildCells([]rune(prompt))})
	}
	return RenderFrame{
		Lines:     lines,
		CursorRow: cursorRow,
		CursorCol: cursorCol,
	}
}

func RenderSingleLine(prompt string, buffer Buffer, cursor Position) string {
	frame := BuildRenderFrame(prompt, "... ", buffer, cursor)
	var builder strings.Builder
	renderer := NewRenderer(&builder)
	renderer.WriteFrame(frame)
	return builder.String()
}

func buildCells(runes []rune) []Cell {
	cells := make([]Cell, 0, len(runes))
	for _, value := range runes {
		cells = append(cells, Cell{
			Rune:  value,
			Width: runeDisplayWidth(value),
		})
	}
	return cells
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
