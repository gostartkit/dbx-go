package commandlang

import "strings"

type TokenType int

const (
	TokenWord TokenType = iota
	TokenFlag
	TokenString
	TokenEquals
	TokenPipe
	TokenBackslash
	TokenNewline
	TokenEOF
	TokenError
)

type Token struct {
	Type      TokenType
	Literal   string
	StartRune int
	EndRune   int
}

type Lexer struct {
	input []rune
	pos   int
}

func Lex(input string) []Token {
	lexer := &Lexer{input: []rune(input)}
	return lexer.lexAll()
}

func (l *Lexer) lexAll() []Token {
	tokens := make([]Token, 0, len(l.input)+1)
	for {
		token := l.nextToken()
		tokens = append(tokens, token)
		if token.Type == TokenEOF {
			return tokens
		}
	}
}

func (l *Lexer) nextToken() Token {
	l.skipHorizontalWhitespace()
	if l.pos >= len(l.input) {
		return Token{Type: TokenEOF, StartRune: l.pos, EndRune: l.pos}
	}

	start := l.pos
	ch := l.input[l.pos]
	switch ch {
	case '\n':
		l.pos++
		return Token{Type: TokenNewline, Literal: "\n", StartRune: start, EndRune: l.pos}
	case '|':
		l.pos++
		return Token{Type: TokenPipe, Literal: "|", StartRune: start, EndRune: l.pos}
	case '=':
		l.pos++
		return Token{Type: TokenEquals, Literal: "=", StartRune: start, EndRune: l.pos}
	case '\\':
		l.pos++
		return Token{Type: TokenBackslash, Literal: "\\", StartRune: start, EndRune: l.pos}
	case '"', '\'':
		return l.lexQuoted(ch)
	default:
		return l.lexWordOrFlag()
	}
}

func (l *Lexer) skipHorizontalWhitespace() {
	for l.pos < len(l.input) {
		switch l.input[l.pos] {
		case ' ', '\t', '\r':
			l.pos++
		default:
			return
		}
	}
}

func (l *Lexer) lexQuoted(quote rune) Token {
	start := l.pos
	l.pos++
	var builder strings.Builder
	for l.pos < len(l.input) {
		ch := l.input[l.pos]
		switch ch {
		case quote:
			l.pos++
			return Token{
				Type:      TokenString,
				Literal:   builder.String(),
				StartRune: start,
				EndRune:   l.pos,
			}
		case '\\':
			if l.pos+1 >= len(l.input) {
				l.pos++
				return Token{
					Type:      TokenError,
					Literal:   builder.String(),
					StartRune: start,
					EndRune:   l.pos,
				}
			}
			l.pos++
			builder.WriteRune(l.input[l.pos])
			l.pos++
		case '\n':
			return Token{
				Type:      TokenError,
				Literal:   builder.String(),
				StartRune: start,
				EndRune:   l.pos,
			}
		default:
			builder.WriteRune(ch)
			l.pos++
		}
	}
	return Token{Type: TokenError, Literal: builder.String(), StartRune: start, EndRune: l.pos}
}

func (l *Lexer) lexWordOrFlag() Token {
	start := l.pos
	var builder strings.Builder
	tokenType := TokenWord
	if l.hasPrefix("--") {
		tokenType = TokenFlag
	}

	for l.pos < len(l.input) {
		ch := l.input[l.pos]
		switch ch {
		case ' ', '\t', '\r', '\n', '|':
			return Token{Type: tokenType, Literal: builder.String(), StartRune: start, EndRune: l.pos}
		case '=':
			if tokenType == TokenFlag && builder.Len() > 0 {
				return Token{Type: tokenType, Literal: builder.String(), StartRune: start, EndRune: l.pos}
			}
			builder.WriteRune(ch)
			l.pos++
		case '\\':
			if l.pos+1 < len(l.input) {
				next := l.input[l.pos+1]
				if next == ' ' || next == '\t' || next == '\\' || next == '"' || next == '\'' {
					l.pos++
					builder.WriteRune(l.input[l.pos])
					l.pos++
					continue
				}
			}
			return Token{Type: tokenType, Literal: builder.String(), StartRune: start, EndRune: l.pos}
		default:
			builder.WriteRune(ch)
			l.pos++
		}
	}
	return Token{Type: tokenType, Literal: builder.String(), StartRune: start, EndRune: l.pos}
}

func (l *Lexer) hasPrefix(prefix string) bool {
	runes := []rune(prefix)
	if l.pos+len(runes) > len(l.input) {
		return false
	}
	for idx, value := range runes {
		if l.input[l.pos+idx] != value {
			return false
		}
	}
	return true
}
