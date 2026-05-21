package editor

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"

	"pkg.gostartkit.com/dbx/internal/commandlang"
)

type Terminal struct {
	reader             *bufio.Reader
	keyReader          *KeyReader
	renderer           *Renderer
	out                io.Writer
	inFile             *os.File
	completer          Completer
	history            *HistoryNavigator
	isTerm             func() bool
	rawActive          bool
	prompt             string
	continuationPrompt string
	editor             *Editor
	session            *CompletionSession
}

type rawModeWriter struct {
	writer io.Writer
}

func NewTerminal(reader *bufio.Reader, out io.Writer, inFile *os.File) *Terminal {
	terminal := &Terminal{
		reader:             reader,
		keyReader:          NewKeyReader(reader),
		renderer:           NewRenderer(out),
		out:                out,
		inFile:             inFile,
		editor:             New(),
		continuationPrompt: "... ",
	}
	terminal.isTerm = func() bool {
		return inFile != nil && term.IsTerminal(int(inFile.Fd()))
	}
	return terminal
}

func (t *Terminal) SetCompleter(completer Completer) {
	t.completer = completer
}

func (t *Terminal) SetHistory(entries []string) {
	t.history = NewHistoryNavigator(entries)
}

func (t *Terminal) AppendHistory(entry string) bool {
	if t.history == nil {
		t.history = NewHistoryNavigator(nil)
	}
	return t.history.Add(entry)
}

func (t *Terminal) ReadLine(prompt string) (string, error) {
	if t.inFile == nil || t.isTerm == nil || !t.isTerm() || t.completer == nil {
		fmt.Fprint(t.out, prompt)
		return t.readLine()
	}
	return t.readLineInteractive(prompt)
}

func (t *Terminal) Println(args ...any) {
	t.ClearLine()
	fmt.Fprintln(t.systemWriter(), args...)
}

func (t *Terminal) Printf(format string, args ...any) {
	t.ClearLine()
	fmt.Fprintf(t.systemWriter(), format, args...)
}

func (t *Terminal) ClearLine() {
	if t.isTerm == nil || !t.isTerm() {
		return
	}
	t.renderer.ClearLine()
}

func (t *Terminal) Redraw() {
	if !t.rawActive {
		return
	}
	t.renderer.Redraw(t.prompt, t.continuationPrompt, t.editor)
}

func (t *Terminal) PrintSystemOutput(fn func(io.Writer)) {
	if t.rawActive {
		t.ClearLine()
		t.writeNewline()
		fn(t.systemWriter())
		t.Redraw()
		return
	}
	fn(t.systemWriter())
}

func (t *Terminal) readLineInteractive(prompt string) (string, error) {
	state, err := term.MakeRaw(int(t.inFile.Fd()))
	if err != nil {
		return "", err
	}
	defer term.Restore(int(t.inFile.Fd()), state)

	t.rawActive = true
	t.prompt = prompt
	t.editor.SetText("")
	t.resetCompletionSession()
	defer func() {
		t.rawActive = false
		t.prompt = ""
		t.editor.SetText("")
		t.resetCompletionSession()
	}()

	fmt.Fprint(t.out, prompt)

	for {
		event, err := t.keyReader.ReadEvent()
		if err != nil {
			return "", err
		}
		done, line, err := t.handleEvent(event)
		if err != nil || done {
			return line, err
		}
	}
}

func (t *Terminal) handleEvent(event KeyEvent) (bool, string, error) {
	if event.Type == KeyTab {
		if t.applyCompletion() {
			t.Redraw()
		}
		return false, "", nil
	}

	t.resetCompletionSession()

	switch event.Type {
	case KeyEnter:
		if commandlang.IsContinuation(commandlang.Lex(t.editor.CurrentLine())) {
			t.editor.setCurrentLine(trimContinuationMarker([]rune(t.editor.CurrentLine())))
			t.editor.AppendLine()
			t.Redraw()
			return false, "", nil
		}
		command := commandlang.JoinLogicalLines(t.editor.Text())
		t.Redraw()
		t.writeNewline()
		if t.history != nil {
			t.history.Reset()
		}
		return true, strings.TrimSpace(command), nil
	case KeyCtrlC:
		t.Redraw()
		t.writeNewline()
		if t.history != nil {
			t.history.Reset()
		}
		return true, "", ErrInputCanceled
	case KeyCtrlD:
		if t.editor.Text() == "" {
			t.writeNewline()
			if t.history != nil {
				t.history.Reset()
			}
			return true, "", io.EOF
		}
		if t.editor.LineCount() > 1 && t.editor.CurrentLine() == "" {
			t.writeNewline()
			if t.history != nil {
				t.history.Reset()
			}
			return true, "", ErrInputCanceled
		}
		if t.editor.DeleteForward() {
			t.Redraw()
		}
	case KeyRune:
		t.editor.InsertRune(event.Rune)
		t.Redraw()
	case KeyBackspace:
		if t.editor.DeleteBackward() {
			t.Redraw()
		}
	case KeyDelete:
		if t.editor.DeleteForward() {
			t.Redraw()
		}
	case KeyLeft:
		if t.editor.MoveLeft() {
			t.Redraw()
		}
	case KeyRight:
		if t.editor.MoveRight() {
			t.Redraw()
		}
	case KeyHome, KeyCtrlA:
		if t.editor.MoveHome() {
			t.Redraw()
		}
	case KeyEnd, KeyCtrlE:
		if t.editor.MoveEnd() {
			t.Redraw()
		}
	case KeyUp:
		if t.history != nil {
			t.editor.SetText(t.history.Up(t.editor.CurrentLine()))
			t.Redraw()
		}
	case KeyDown:
		if t.history != nil {
			t.editor.SetText(t.history.Down(t.editor.CurrentLine()))
			t.Redraw()
		}
	}
	return false, "", nil
}

func (t *Terminal) applyCompletion() bool {
	if t.completer == nil {
		return false
	}

	currentBuffer := t.editor.Buffer()
	currentCursor := t.editor.Position()
	currentLine := currentBuffer.LineString(currentCursor.Line)
	if t.session != nil && (!buffersEqual(currentBuffer, t.session.OriginalBuffer) || currentCursor != t.session.OriginalCursor) && !t.session.Contains(currentBuffer, currentCursor) {
		t.resetCompletionSession()
	}

	if t.session != nil && len(t.session.Suggestions) > 0 {
		if !t.session.ListShown {
			t.PrintSystemOutput(func(w io.Writer) {
				t.printSuggestionsTo(w, t.session.Suggestions)
				fmt.Fprintln(w)
			})
			t.session.ListShown = true
			return false
		}
		selected := t.session.Current()
		t.session.Advance()
		t.applyCompletionResultToSession(selected.Result)
		return true
	}

	request := CompletionRequest{
		Buffer: t.editor.Buffer(),
		Cursor: t.editor.Position(),
	}
	request.Tokens = commandlang.Lex(request.Buffer.String())
	request.CommandContext = commandlang.BuildCommandContext(request.Tokens, bufferRuneOffset(request.Buffer, request.Cursor))
	request.Program = commandlang.ParseTokens(request.Tokens)
	request.SyntaxContext = commandlang.BuildSyntaxContextWithRegistry(request.Program, bufferRuneOffset(request.Buffer, request.Cursor), commandlang.DefaultRegistry())
	completion := t.completer(request)
	if len(completion.Suggestions) == 0 {
		t.resetCompletionSession()
		return false
	}

	if len(completion.Suggestions) == 1 {
		t.resetCompletionSession()
		t.editor.ApplyCompletion(completion.Suggestions[0].Result)
		return true
	}

	t.session = NewCompletionSession(currentBuffer, currentCursor, completion.Suggestions)
	if common, ok := CommonSuggestionResult(currentLine, t.session.Suggestions); ok {
		nextLine, nextCursor := ApplyCompletion(currentLine, common)
		if nextLine != currentLine || nextCursor != currentCursor.Column {
			t.session.CommonResult = common
			t.session.HasCommon = true
			if matched := MatchingSuggestionIndex(currentLine, t.session.Suggestions, nextLine, nextCursor); matched >= 0 {
				t.session.SelectedIndex = (matched + 1) % len(t.session.Suggestions)
			}
			t.applyCompletionResultToSession(common)
			return true
		}
	}

	selected := t.session.Current()
	t.session.Advance()
	t.applyCompletionResultToSession(selected.Result)
	return true
}

func (t *Terminal) applyCompletionResultToSession(result CompletionResult) {
	if t.session == nil {
		t.editor.ApplyCompletion(result)
		return
	}
	buffer, cursor := ApplyCompletionToBuffer(t.session.OriginalBuffer, t.session.OriginalCursor, result)
	t.editor.SetBuffer(buffer)
	t.editor.cursor = cursor
}

func trimContinuationMarker(line []rune) []rune {
	trimmed := append([]rune(nil), line...)
	for len(trimmed) > 0 && (trimmed[len(trimmed)-1] == ' ' || trimmed[len(trimmed)-1] == '\t' || trimmed[len(trimmed)-1] == '\r') {
		trimmed = trimmed[:len(trimmed)-1]
	}
	if len(trimmed) > 0 && trimmed[len(trimmed)-1] == '\\' {
		trimmed = trimmed[:len(trimmed)-1]
	}
	return trimmed
}

func bufferRuneOffset(buffer Buffer, cursor Position) int {
	offset := 0
	lineIndex := clamp(cursor.Line, 0, len(buffer.Lines)-1)
	for idx := 0; idx < lineIndex; idx++ {
		offset += len(buffer.Lines[idx]) + 1
	}
	offset += clamp(cursor.Column, 0, len(buffer.Lines[lineIndex]))
	return offset
}

func (t *Terminal) printSuggestionsTo(w io.Writer, suggestions []Suggestion) {
	maxWidth := 0
	for _, suggestion := range suggestions {
		if len(suggestion.Value) > maxWidth {
			maxWidth = len(suggestion.Value)
		}
	}
	for _, suggestion := range suggestions {
		if suggestion.Description == "" {
			fmt.Fprintln(w, suggestion.Value)
			continue
		}
		fmt.Fprintf(w, "%-*s  %s\n", maxWidth, suggestion.Value, suggestion.Description)
	}
}

func (t *Terminal) systemWriter() io.Writer {
	if t.rawActive {
		return rawModeWriter{writer: t.out}
	}
	return t.out
}

func (t *Terminal) writeNewline() {
	if t.rawActive {
		fmt.Fprint(t.out, "\r\n")
		return
	}
	fmt.Fprint(t.out, "\n")
}

func (t *Terminal) resetCompletionSession() {
	t.session = nil
}

func buffersEqual(left Buffer, right Buffer) bool {
	if len(left.Lines) != len(right.Lines) {
		return false
	}
	for idx := range left.Lines {
		if string(left.Lines[idx]) != string(right.Lines[idx]) {
			return false
		}
	}
	return true
}

func (w rawModeWriter) Write(p []byte) (int, error) {
	text := string(p)
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\n", "\r\n")
	_, err := io.WriteString(w.writer, text)
	if err != nil {
		return 0, err
	}
	return len(p), nil
}

func (t *Terminal) readLine() (string, error) {
	line, err := t.reader.ReadString('\n')
	if err != nil {
		if err == io.EOF && line != "" {
			return strings.TrimSpace(line), nil
		}
		return "", err
	}
	return strings.TrimSpace(line), nil
}
