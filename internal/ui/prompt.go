package ui

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"golang.org/x/term"
)

type Completion struct {
	Prefix      string
	Suggestions []Suggestion
	Hint        string
}

type Suggestion struct {
	Value       string
	Description string
	Category    string
}

type Completer func(line string) Completion

type Prompt struct {
	reader    *bufio.Reader
	out       io.Writer
	inFile    *os.File
	completer Completer
	history   *HistoryNavigator
	isTerm    func() bool
	rawActive bool
	label     string
	current   string
}

type rawModeWriter struct {
	writer io.Writer
}

func NewPrompt(in io.Reader, out io.Writer) *Prompt {
	var inFile *os.File
	if file, ok := in.(*os.File); ok {
		inFile = file
	}

	return &Prompt{
		reader: bufio.NewReader(in),
		out:    out,
		inFile: inFile,
		isTerm: func() bool {
			return inFile != nil && term.IsTerminal(int(inFile.Fd()))
		},
	}
}

func (p *Prompt) SetCompleter(completer Completer) {
	p.completer = completer
}

func (p *Prompt) SetHistory(entries []string) {
	p.history = NewHistoryNavigator(entries)
}

func (p *Prompt) AppendHistory(entry string) bool {
	if p.history == nil {
		p.history = NewHistoryNavigator(nil)
	}
	return p.history.Add(entry)
}

func (p *Prompt) Println(args ...any) {
	p.ClearLine()
	fmt.Fprintln(p.systemWriter(), args...)
}

func (p *Prompt) Printf(format string, args ...any) {
	p.ClearLine()
	fmt.Fprintf(p.systemWriter(), format, args...)
}

func (p *Prompt) ReadPrompt(label string) (string, error) {
	if p.inFile != nil && p.isTerm != nil && p.isTerm() && p.completer != nil {
		return p.readPromptInteractive(label)
	}

	fmt.Fprint(p.out, label)
	return p.readLine()
}

func (p *Prompt) Ask(label, defaultValue string) (string, error) {
	if defaultValue != "" {
		fmt.Fprintf(p.out, "%s [%s]: ", label, defaultValue)
	} else {
		fmt.Fprintf(p.out, "%s: ", label)
	}

	value, err := p.readLine()
	if err != nil {
		return "", err
	}
	if value == "" {
		return defaultValue, nil
	}
	return value, nil
}

func (p *Prompt) AskPassword(label string) (string, error) {
	fmt.Fprintf(p.out, "%s: ", label)
	if p.inFile != nil && p.isTerm != nil && p.isTerm() {
		value, err := term.ReadPassword(int(p.inFile.Fd()))
		fmt.Fprintln(p.out)
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(string(value)), nil
	}

	return p.readLine()
}

func (p *Prompt) Choose(label string, options []string, defaultValue string) (string, error) {
	for idx, option := range options {
		fmt.Fprintf(p.out, "  %d. %s\n", idx+1, option)
	}

	for {
		value, err := p.Ask(label, defaultValue)
		if err != nil {
			return "", err
		}
		if value == "" {
			if defaultValue != "" {
				return defaultValue, nil
			}
			if len(options) == 1 {
				return options[0], nil
			}
			fmt.Fprintln(p.out, "Please choose one of the listed options.")
			continue
		}

		if index, err := strconv.Atoi(value); err == nil {
			if index >= 1 && index <= len(options) {
				return options[index-1], nil
			}
		}

		for _, option := range options {
			if value == option {
				return option, nil
			}
		}

		fmt.Fprintln(p.out, "Please choose one of the listed options.")
	}
}

func (p *Prompt) Confirm(label string, defaultYes bool) (bool, error) {
	defaultValue := "y"
	if !defaultYes {
		defaultValue = "n"
	}

	for {
		value, err := p.Ask(label+" [y/n]", defaultValue)
		if err != nil {
			return false, err
		}

		switch strings.ToLower(strings.TrimSpace(value)) {
		case "y", "yes":
			return true, nil
		case "n", "no":
			return false, nil
		}

		fmt.Fprintln(p.out, "Please answer y or n.")
	}
}

func (p *Prompt) readPromptInteractive(label string) (string, error) {
	state, err := term.MakeRaw(int(p.inFile.Fd()))
	if err != nil {
		return "", err
	}
	defer term.Restore(int(p.inFile.Fd()), state)
	p.rawActive = true
	p.label = label
	p.current = ""
	defer func() {
		p.rawActive = false
		p.label = ""
		p.current = ""
	}()

	fmt.Fprint(p.out, label)

	current := ""
	for {
		b, err := p.reader.ReadByte()
		if err != nil {
			return "", err
		}

		switch b {
		case '\n', '\r':
			p.redrawLine(label, current, "")
			p.writeNewline()
			if p.history != nil {
				p.history.Reset()
			}
			return strings.TrimSpace(current), nil
		case 3:
			if p.history != nil {
				p.history.Reset()
			}
			return "", context.Canceled
		case '\t':
			completion := p.completer(current)
			current = p.applyCompletion(current, completion)
			p.redrawCurrentLine(label, current)
		case 127, 8:
			if current == "" {
				continue
			}
			runes := []rune(current)
			current = string(runes[:len(runes)-1])
			p.redrawCurrentLine(label, current)
		case 27:
			updated, handled, escErr := p.handleEscapeSequence(current)
			if escErr != nil {
				return "", escErr
			}
			if handled {
				current = updated
				p.redrawCurrentLine(label, current)
			}
		default:
			current += string(b)
			p.redrawCurrentLine(label, current)
		}
	}
}

func (p *Prompt) applyCompletion(current string, completion Completion) string {
	candidates := completionValues(completion)
	if len(candidates) == 0 {
		return current
	}

	common := longestCommonPrefix(candidates)
	if len(candidates) == 1 || (completion.Prefix != "" && len(common) > len(completion.Prefix)) {
		prefixLen := len(completion.Prefix)
		if prefixLen > len(current) {
			prefixLen = len(current)
		}

		base := current[:len(current)-prefixLen]
		replacement := candidates[0]
		if len(candidates) > 1 && len(common) > len(completion.Prefix) {
			replacement = common
		}

		updated := base + replacement
		if len(candidates) == 1 {
			updated += " "
		}
		return updated
	}

	p.PrintSystemOutput(func(w io.Writer) {
		p.printSuggestionsTo(w, completion.Suggestions)
		fmt.Fprintln(w)
	})
	return current
}

func (p *Prompt) handleEscapeSequence(current string) (string, bool, error) {
	first, err := p.reader.ReadByte()
	if err != nil {
		return current, false, err
	}
	if first != '[' {
		return current, false, nil
	}

	second, err := p.reader.ReadByte()
	if err != nil {
		return current, false, err
	}

	switch second {
	case 'A':
		if p.history == nil {
			return current, false, nil
		}
		return p.history.Up(current), true, nil
	case 'B':
		if p.history == nil {
			return current, false, nil
		}
		return p.history.Down(current), true, nil
	case 'C', 'D':
		return current, false, nil
	default:
		return current, false, nil
	}
}

func (p *Prompt) redrawLine(label string, current string, hint string) {
	_ = hint
	fmt.Fprintf(p.out, "\r\033[2K%s%s", label, current)
}

func (p *Prompt) ClearLine() {
	if p.isTerm == nil || !p.isTerm() {
		return
	}
	fmt.Fprint(p.out, "\r\033[2K")
}

func (p *Prompt) Redraw() {
	if !p.rawActive {
		return
	}
	p.redrawLine(p.label, p.current, "")
}

func (p *Prompt) PrintSystemOutput(fn func(io.Writer)) {
	if p.rawActive {
		p.ClearLine()
		p.writeNewline()
		fn(p.systemWriter())
		p.Redraw()
		return
	}
	fn(p.systemWriter())
}

func (p *Prompt) systemWriter() io.Writer {
	if p.rawActive {
		return rawModeWriter{writer: p.out}
	}
	return p.out
}

func (p *Prompt) writeNewline() {
	if p.rawActive {
		fmt.Fprint(p.out, "\r\n")
		return
	}
	fmt.Fprint(p.out, "\n")
}

func longestCommonPrefix(values []string) string {
	if len(values) == 0 {
		return ""
	}

	prefix := values[0]
	for _, value := range values[1:] {
		for !strings.HasPrefix(value, prefix) {
			if prefix == "" {
				return ""
			}
			prefix = prefix[:len(prefix)-1]
		}
	}
	return prefix
}

func (p *Prompt) redrawCurrentLine(label string, current string) {
	p.label = label
	p.current = current
	p.redrawLine(label, current, "")
}

func completionValues(completion Completion) []string {
	values := make([]string, 0, len(completion.Suggestions))
	for _, suggestion := range completion.Suggestions {
		values = append(values, suggestion.Value)
	}
	return values
}

func (p *Prompt) printSuggestions(suggestions []Suggestion) {
	p.printSuggestionsTo(p.out, suggestions)
}

func (p *Prompt) printSuggestionsTo(w io.Writer, suggestions []Suggestion) {
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

func (p *Prompt) readLine() (string, error) {
	line, err := p.reader.ReadString('\n')
	if err != nil {
		if err == io.EOF && line != "" {
			return strings.TrimSpace(line), nil
		}
		return "", err
	}
	return strings.TrimSpace(line), nil
}
