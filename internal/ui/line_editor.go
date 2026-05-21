package ui

type keyKind int

const (
	keyIgnored keyKind = iota
	keyRune
	keyEnter
	keyTab
	keyBackspace
	keyDelete
	keyLeft
	keyRight
	keyUp
	keyDown
	keyHome
	keyEnd
	keyCtrlA
	keyCtrlC
	keyCtrlD
	keyCtrlE
)

type keyEvent struct {
	kind keyKind
	r    rune
}

type lineEditorResult struct {
	changed bool
	submit  bool
	cancel  bool
	eof     bool
}

type lineEditor struct {
	buffer  []rune
	cursor  int
	history *HistoryNavigator
}

func newLineEditor(history *HistoryNavigator) *lineEditor {
	return &lineEditor{history: history}
}

func (e *lineEditor) String() string {
	return string(e.buffer)
}

func (e *lineEditor) Cursor() int {
	return e.cursor
}

func (e *lineEditor) SetLine(line string) {
	e.buffer = []rune(line)
	e.cursor = len(e.buffer)
}

func (e *lineEditor) HandleKey(event keyEvent) lineEditorResult {
	switch event.kind {
	case keyRune:
		e.insertRune(event.r)
		return lineEditorResult{changed: true}
	case keyEnter:
		return lineEditorResult{submit: true}
	case keyBackspace:
		return lineEditorResult{changed: e.backspace()}
	case keyDelete:
		return lineEditorResult{changed: e.delete()}
	case keyLeft:
		return lineEditorResult{changed: e.moveLeft()}
	case keyRight:
		return lineEditorResult{changed: e.moveRight()}
	case keyHome, keyCtrlA:
		return lineEditorResult{changed: e.moveHome()}
	case keyEnd, keyCtrlE:
		return lineEditorResult{changed: e.moveEnd()}
	case keyUp:
		return lineEditorResult{changed: e.historyUp()}
	case keyDown:
		return lineEditorResult{changed: e.historyDown()}
	case keyCtrlC:
		return lineEditorResult{cancel: true}
	case keyCtrlD:
		if len(e.buffer) == 0 {
			return lineEditorResult{eof: true}
		}
		return lineEditorResult{changed: e.delete()}
	default:
		return lineEditorResult{}
	}
}

func (e *lineEditor) insertRune(r rune) {
	if e.cursor == len(e.buffer) {
		e.buffer = append(e.buffer, r)
		e.cursor++
		return
	}

	e.buffer = append(e.buffer, 0)
	copy(e.buffer[e.cursor+1:], e.buffer[e.cursor:])
	e.buffer[e.cursor] = r
	e.cursor++
}

func (e *lineEditor) backspace() bool {
	if e.cursor == 0 {
		return false
	}
	copy(e.buffer[e.cursor-1:], e.buffer[e.cursor:])
	e.buffer = e.buffer[:len(e.buffer)-1]
	e.cursor--
	return true
}

func (e *lineEditor) delete() bool {
	if e.cursor >= len(e.buffer) {
		return false
	}
	copy(e.buffer[e.cursor:], e.buffer[e.cursor+1:])
	e.buffer = e.buffer[:len(e.buffer)-1]
	return true
}

func (e *lineEditor) moveLeft() bool {
	if e.cursor == 0 {
		return false
	}
	e.cursor--
	return true
}

func (e *lineEditor) moveRight() bool {
	if e.cursor >= len(e.buffer) {
		return false
	}
	e.cursor++
	return true
}

func (e *lineEditor) moveHome() bool {
	if e.cursor == 0 {
		return false
	}
	e.cursor = 0
	return true
}

func (e *lineEditor) moveEnd() bool {
	if e.cursor == len(e.buffer) {
		return false
	}
	e.cursor = len(e.buffer)
	return true
}

func (e *lineEditor) historyUp() bool {
	if e.history == nil {
		return false
	}
	next := e.history.Up(e.String())
	if next == e.String() && len(e.buffer) > 0 {
		e.cursor = len(e.buffer)
		return false
	}
	e.SetLine(next)
	return true
}

func (e *lineEditor) historyDown() bool {
	if e.history == nil {
		return false
	}
	next := e.history.Down(e.String())
	if next == e.String() && len(e.buffer) > 0 {
		e.cursor = len(e.buffer)
		return false
	}
	e.SetLine(next)
	return true
}
