package ui

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"golang.org/x/term"
)

type Completion struct {
	Prefix     string
	Candidates []string
}

type Completer func(line string) Completion

type Prompt struct {
	reader    *bufio.Reader
	out       io.Writer
	inFile    *os.File
	completer Completer
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
	}
}

func (p *Prompt) SetCompleter(completer Completer) {
	p.completer = completer
}

func (p *Prompt) Println(args ...any) {
	fmt.Fprintln(p.out, args...)
}

func (p *Prompt) Printf(format string, args ...any) {
	fmt.Fprintf(p.out, format, args...)
}

func (p *Prompt) ReadPrompt(label string) (string, error) {
	if p.inFile != nil && term.IsTerminal(int(p.inFile.Fd())) && p.completer != nil {
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

func (p *Prompt) readPromptInteractive(label string) (string, error) {
	fmt.Fprint(p.out, label)

	var builder strings.Builder
	for {
		b, err := p.reader.ReadByte()
		if err != nil {
			return "", err
		}

		switch b {
		case '\n', '\r':
			fmt.Fprintln(p.out)
			return strings.TrimSpace(builder.String()), nil
		case '\t':
			completion := p.completer(builder.String())
			p.applyCompletion(label, &builder, completion)
		case 127, 8:
			current := builder.String()
			if current == "" {
				continue
			}
			runes := []rune(current)
			builder.Reset()
			builder.WriteString(string(runes[:len(runes)-1]))
			fmt.Fprint(p.out, "\b \b")
		default:
			builder.WriteByte(b)
			fmt.Fprintf(p.out, "%c", b)
		}
	}
}

func (p *Prompt) applyCompletion(label string, builder *strings.Builder, completion Completion) {
	if len(completion.Candidates) == 0 {
		return
	}

	common := longestCommonPrefix(completion.Candidates)
	if len(completion.Candidates) == 1 || (completion.Prefix != "" && len(common) > len(completion.Prefix)) {
		current := builder.String()
		prefixLen := len(completion.Prefix)
		if prefixLen > len(current) {
			prefixLen = len(current)
		}

		base := current[:len(current)-prefixLen]
		replacement := completion.Candidates[0]
		if len(completion.Candidates) > 1 && len(common) > len(completion.Prefix) {
			replacement = common
		}

		updated := base + replacement
		if len(completion.Candidates) == 1 {
			updated += " "
		}

		builder.Reset()
		builder.WriteString(updated)
		fmt.Fprintf(p.out, "\r\033[K%s%s", label, updated)
		return
	}

	fmt.Fprintln(p.out)
	for _, candidate := range completion.Candidates {
		fmt.Fprintln(p.out, candidate)
	}
	fmt.Fprintf(p.out, "%s%s", label, builder.String())
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
