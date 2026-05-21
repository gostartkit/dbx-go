package editor

type Editor struct {
	buffer Buffer
	cursor Position
}

func New() *Editor {
	return &Editor{buffer: NewBufferFromString("")}
}

func (e *Editor) SetBuffer(buffer Buffer) {
	e.buffer = buffer.Clone()
	e.buffer.ensureLine()
	e.cursor.Line = clamp(e.cursor.Line, 0, len(e.buffer.Lines)-1)
	e.cursor.Column = clamp(e.cursor.Column, 0, len(e.buffer.Lines[e.cursor.Line]))
}

func (e *Editor) SetText(text string) {
	e.buffer = NewBufferFromString(text)
	e.cursor = Position{Line: 0, Column: len(e.buffer.Lines[0])}
}

func (e *Editor) Buffer() Buffer {
	return e.buffer.Clone()
}

func (e *Editor) Text() string {
	return e.buffer.String()
}

func (e *Editor) CurrentLine() string {
	return string(e.currentLine())
}

func (e *Editor) Cursor() int {
	return e.cursor.Column
}

func (e *Editor) Position() Position {
	return e.cursor
}

func (e *Editor) InsertRune(value rune) {
	line := e.currentLine()
	column := clamp(e.cursor.Column, 0, len(line))
	updated := append([]rune(nil), line[:column]...)
	updated = append(updated, value)
	updated = append(updated, line[column:]...)
	e.setCurrentLine(updated)
	e.cursor.Column++
}

func (e *Editor) DeleteBackward() bool {
	line := e.currentLine()
	if e.cursor.Column == 0 || len(line) == 0 {
		return false
	}
	column := clamp(e.cursor.Column, 0, len(line))
	updated := append([]rune(nil), line[:column-1]...)
	updated = append(updated, line[column:]...)
	e.setCurrentLine(updated)
	e.cursor.Column--
	return true
}

func (e *Editor) DeleteForward() bool {
	line := e.currentLine()
	column := clamp(e.cursor.Column, 0, len(line))
	if column >= len(line) {
		return false
	}
	updated := append([]rune(nil), line[:column]...)
	updated = append(updated, line[column+1:]...)
	e.setCurrentLine(updated)
	return true
}

func (e *Editor) MoveLeft() bool {
	if e.cursor.Column == 0 {
		return false
	}
	e.cursor.Column--
	return true
}

func (e *Editor) MoveRight() bool {
	line := e.currentLine()
	if e.cursor.Column >= len(line) {
		return false
	}
	e.cursor.Column++
	return true
}

func (e *Editor) MoveHome() bool {
	if e.cursor.Column == 0 {
		return false
	}
	e.cursor.Column = 0
	return true
}

func (e *Editor) MoveEnd() bool {
	line := e.currentLine()
	if e.cursor.Column == len(line) {
		return false
	}
	e.cursor.Column = len(line)
	return true
}

func (e *Editor) ApplyCompletion(result CompletionResult) {
	line, cursor := ApplyCompletion(e.CurrentLine(), result)
	e.setCurrentLine([]rune(line))
	e.cursor.Column = cursor
}

func (e *Editor) currentLine() []rune {
	e.buffer.ensureLine()
	return e.buffer.Lines[e.cursor.Line]
}

func (e *Editor) setCurrentLine(line []rune) {
	e.buffer.ensureLine()
	e.buffer.Lines[e.cursor.Line] = append([]rune(nil), line...)
}
