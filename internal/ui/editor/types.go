package editor

import (
	"errors"
	"strings"

	"pkg.gostartkit.com/dbx/internal/commandlang"
)

var ErrInputCanceled = errors.New("input canceled")

type Position struct {
	Line   int
	Column int
}

type Buffer struct {
	Lines [][]rune
}

func NewBufferFromString(value string) Buffer {
	lines := strings.Split(value, "\n")
	if len(lines) == 0 {
		lines = []string{""}
	}
	buffer := Buffer{Lines: make([][]rune, 0, len(lines))}
	for _, line := range lines {
		buffer.Lines = append(buffer.Lines, []rune(line))
	}
	buffer.ensureLine()
	return buffer
}

func (b Buffer) Clone() Buffer {
	clone := Buffer{Lines: make([][]rune, 0, len(b.Lines))}
	for _, line := range b.Lines {
		clone.Lines = append(clone.Lines, append([]rune(nil), line...))
	}
	clone.ensureLine()
	return clone
}

func (b Buffer) Line(index int) []rune {
	if index < 0 || index >= len(b.Lines) {
		return nil
	}
	return append([]rune(nil), b.Lines[index]...)
}

func (b Buffer) LineString(index int) string {
	return string(b.Line(index))
}

func (b Buffer) String() string {
	lines := make([]string, 0, len(b.Lines))
	for _, line := range b.Lines {
		lines = append(lines, string(line))
	}
	return strings.Join(lines, "\n")
}

func (b *Buffer) ensureLine() {
	if len(b.Lines) == 0 {
		b.Lines = [][]rune{{}}
	}
}

type CompletionEdit struct {
	StartRune int
	EndRune   int
	Text      string
}

type CompletionResult struct {
	Edits  []CompletionEdit
	Cursor int
}

type Suggestion struct {
	Value       string
	Description string
	Category    string
	Result      CompletionResult
}

type Completion struct {
	Prefix      string
	Suggestions []Suggestion
	Hint        string
}

type CompletionRequest struct {
	Buffer         Buffer
	Cursor         Position
	Tokens         []commandlang.Token
	CommandContext commandlang.CommandContext
	Program        *commandlang.Program
	SyntaxContext  commandlang.SyntaxContext
}

func NewSingleLineCompletionRequest(line string, cursor int) CompletionRequest {
	if cursor < 0 {
		cursor = 0
	}
	lineRunes := []rune(line)
	if cursor > len(lineRunes) {
		cursor = len(lineRunes)
	}
	request := CompletionRequest{
		Buffer: NewBufferFromString(line),
		Cursor: Position{Line: 0, Column: cursor},
	}
	request.Tokens = commandlang.Lex(line)
	request.CommandContext = commandlang.BuildCommandContext(request.Tokens, cursor)
	request.Program = commandlang.ParseTokens(request.Tokens)
	request.SyntaxContext = commandlang.BuildSyntaxContext(request.Program, cursor, nil)
	return request
}

func (r CompletionRequest) CurrentLine() string {
	return r.Buffer.LineString(r.Cursor.Line)
}

func (r CompletionRequest) CurrentLinePrefix() string {
	line := []rune(r.CurrentLine())
	cursor := clamp(r.Cursor.Column, 0, len(line))
	return string(line[:cursor])
}

func (r CompletionRequest) CurrentLineSuffix() string {
	line := []rune(r.CurrentLine())
	cursor := clamp(r.Cursor.Column, 0, len(line))
	return string(line[cursor:])
}

type Completer func(request CompletionRequest) Completion

type KeyType int

const (
	KeyIgnored KeyType = iota
	KeyRune
	KeyEnter
	KeyTab
	KeyBackspace
	KeyDelete
	KeyLeft
	KeyRight
	KeyUp
	KeyDown
	KeyHome
	KeyEnd
	KeyCtrlA
	KeyCtrlC
	KeyCtrlD
	KeyCtrlE
	KeyAlt
)

type KeyEvent struct {
	Type KeyType
	Rune rune
}

func clamp(value int, min int, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}
