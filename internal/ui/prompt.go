package ui

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"golang.org/x/term"

	"pkg.gostartkit.com/dbx/internal/ui/editor"
)

var ErrPromptCanceled = editor.ErrInputCanceled

type Prompt struct {
	reader   *bufio.Reader
	out      io.Writer
	inFile   *os.File
	terminal *editor.Terminal
}

func NewPrompt(in io.Reader, out io.Writer) *Prompt {
	var inFile *os.File
	if file, ok := in.(*os.File); ok {
		inFile = file
	}

	reader := bufio.NewReader(in)
	return &Prompt{
		reader:   reader,
		out:      out,
		inFile:   inFile,
		terminal: editor.NewTerminal(reader, out, inFile),
	}
}

func (p *Prompt) SetCompleter(completer Completer) {
	p.terminal.SetCompleter(completer)
}

func (p *Prompt) SetHistory(entries []string) {
	p.terminal.SetHistory(entries)
}

func (p *Prompt) Writer() io.Writer {
	return p.out
}

func (p *Prompt) AppendHistory(entry string) bool {
	return p.terminal.AppendHistory(entry)
}

func (p *Prompt) Println(args ...any) {
	p.terminal.Println(args...)
}

func (p *Prompt) Printf(format string, args ...any) {
	p.terminal.Printf(format, args...)
}

func (p *Prompt) ReadPrompt(label string) (string, error) {
	return p.terminal.ReadLine(label)
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
	if p.inFile != nil && term.IsTerminal(int(p.inFile.Fd())) {
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

func (p *Prompt) ClearLine() {
	p.terminal.ClearLine()
}

func (p *Prompt) Redraw() {
	p.terminal.Redraw()
}

func (p *Prompt) PrintSystemOutput(fn func(io.Writer)) {
	p.terminal.PrintSystemOutput(fn)
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
