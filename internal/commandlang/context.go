package commandlang

import "strings"

type CommandContext struct {
	Tokens                []Token
	CursorRune            int
	CursorToken           *Token
	CommandPath           []string
	CurrentArgIndex       int
	CurrentFlag           string
	ExpectingValueForFlag string
}

func BuildCommandContext(tokens []Token, cursorRune int) CommandContext {
	ctx := CommandContext{
		Tokens:      append([]Token(nil), tokens...),
		CursorRune:  cursorRune,
		CommandPath: make([]string, 0, 4),
	}
	ctx.CursorToken = cursorToken(tokens, cursorRune)

	wordsBeforeCursor := make([]Token, 0, len(tokens))
	flagValuesConsumed := 0
	currentFlag := ""
	expectingFlagValue := ""
	for idx := 0; idx < len(tokens); idx++ {
		token := tokens[idx]
		if token.Type == TokenEOF || token.Type == TokenNewline || token.Type == TokenPipe {
			break
		}
		if token.EndRune > cursorRune && token.StartRune > cursorRune {
			break
		}

		switch token.Type {
		case TokenFlag:
			currentFlag = token.Literal
			if token.EndRune >= cursorRune {
				ctx.CurrentFlag = token.Literal
			}
			if idx+1 < len(tokens) && tokens[idx+1].Type == TokenEquals {
				expectingFlagValue = token.Literal
				continue
			}
			if idx+1 >= len(tokens) || !isValueToken(tokens[idx+1]) {
				expectingFlagValue = token.Literal
			}
		case TokenEquals:
			continue
		case TokenWord, TokenString:
			if currentFlag != "" {
				flagValuesConsumed++
				if token.EndRune >= cursorRune || token.StartRune <= cursorRune && token.EndRune >= cursorRune {
					ctx.ExpectingValueForFlag = currentFlag
				}
				currentFlag = ""
				expectingFlagValue = ""
				continue
			}
			wordsBeforeCursor = append(wordsBeforeCursor, token)
		}
	}

	if ctx.CursorToken != nil {
		switch ctx.CursorToken.Type {
		case TokenFlag:
			ctx.CurrentFlag = ctx.CursorToken.Literal
		case TokenWord, TokenString:
			prev := previousNonSpaceToken(tokens, *ctx.CursorToken)
			if prev != nil && prev.Type == TokenEquals {
				prevPrev := previousNonSpaceToken(tokens, *prev)
				if prevPrev != nil && prevPrev.Type == TokenFlag {
					ctx.ExpectingValueForFlag = prevPrev.Literal
				}
			}
		}
	}
	if expectingFlagValue != "" && (ctx.CursorToken == nil || ctx.CursorToken.Type != TokenFlag) {
		ctx.ExpectingValueForFlag = expectingFlagValue
	}

	for _, token := range wordsBeforeCursor {
		if strings.TrimSpace(token.Literal) == "" {
			continue
		}
		ctx.CommandPath = append(ctx.CommandPath, token.Literal)
	}
	ctx.CurrentArgIndex = len(wordsBeforeCursor)
	if ctx.CursorToken != nil && isValueToken(*ctx.CursorToken) {
		ctx.CurrentArgIndex--
		if ctx.CurrentArgIndex < 0 {
			ctx.CurrentArgIndex = 0
		}
	}
	_ = flagValuesConsumed
	return ctx
}

func cursorToken(tokens []Token, cursorRune int) *Token {
	for idx := range tokens {
		token := tokens[idx]
		if token.Type == TokenEOF {
			break
		}
		if cursorRune >= token.StartRune && cursorRune <= token.EndRune {
			return &tokens[idx]
		}
	}
	return nil
}

func previousNonSpaceToken(tokens []Token, target Token) *Token {
	for idx := len(tokens) - 1; idx >= 0; idx-- {
		token := tokens[idx]
		if token.EndRune > target.StartRune {
			continue
		}
		if token.Type == TokenEOF {
			continue
		}
		return &tokens[idx]
	}
	return nil
}

func isValueToken(token Token) bool {
	return token.Type == TokenWord || token.Type == TokenString || token.Type == TokenError
}

func IsContinuation(tokens []Token) bool {
	last := lastNonEOF(tokens)
	return last != nil && last.Type == TokenBackslash
}

func JoinLogicalLines(input string) string {
	lines := strings.Split(input, "\n")
	parts := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimRight(line, " \t\r")
		if strings.HasSuffix(trimmed, "\\") {
			trimmed = strings.TrimSpace(strings.TrimRight(strings.TrimSuffix(trimmed, "\\"), " \t\r"))
			if trimmed != "" {
				parts = append(parts, trimmed)
			}
			continue
		}
		trimmed = strings.TrimSpace(trimmed)
		if trimmed != "" {
			parts = append(parts, trimmed)
		}
	}
	return strings.Join(parts, " ")
}

func lastNonEOF(tokens []Token) *Token {
	for idx := len(tokens) - 1; idx >= 0; idx-- {
		token := tokens[idx]
		if token.Type == TokenEOF {
			continue
		}
		if token.Type == TokenNewline {
			continue
		}
		return &tokens[idx]
	}
	return nil
}
