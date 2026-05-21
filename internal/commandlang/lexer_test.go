package commandlang

import "testing"

func TestLexQuotedEscapedAndFlags(t *testing.T) {
	t.Parallel()

	tokens := Lex(`exec create-user --role "read only" --name=alice`)
	assertToken(t, tokens[0], TokenWord, "exec", 0, 4)
	assertToken(t, tokens[1], TokenWord, "create-user", 5, 16)
	assertToken(t, tokens[2], TokenFlag, "--role", 17, 23)
	assertToken(t, tokens[3], TokenString, "read only", 24, 35)
	assertToken(t, tokens[4], TokenFlag, "--name", 36, 42)
	assertToken(t, tokens[5], TokenEquals, "=", 42, 43)
	assertToken(t, tokens[6], TokenWord, "alice", 43, 48)
}

func TestLexEscapedSpaceAndChinese(t *testing.T) {
	t.Parallel()

	tokens := Lex(`template render create-user name\ with\ space 中文`)
	assertToken(t, tokens[0], TokenWord, "template", 0, 8)
	assertToken(t, tokens[1], TokenWord, "render", 9, 15)
	assertToken(t, tokens[2], TokenWord, "create-user", 16, 27)
	assertToken(t, tokens[3], TokenWord, "name with space", 28, 45)
	assertToken(t, tokens[4], TokenWord, "中文", 46, 48)
}

func TestLexUnclosedQuoteReturnsErrorToken(t *testing.T) {
	t.Parallel()

	tokens := Lex(`exec "broken`)
	assertToken(t, tokens[1], TokenError, "broken", 5, 12)
}

func TestBuildCommandContextUnderstandsCursorTokenAndFlagValue(t *testing.T) {
	t.Parallel()

	line := `exec create-user --role "read only"`
	tokens := Lex(line)
	cursor := len([]rune(`exec create-user --role "rea`))
	ctx := BuildCommandContext(tokens, cursor)
	if ctx.CursorToken == nil || ctx.CursorToken.Type != TokenString {
		t.Fatalf("cursor token = %#v", ctx.CursorToken)
	}
	if ctx.ExpectingValueForFlag != "--role" {
		t.Fatalf("ExpectingValueForFlag = %q", ctx.ExpectingValueForFlag)
	}
}

func TestContinuationAndJoinLogicalLines(t *testing.T) {
	t.Parallel()

	input := "exec grant-readonly \\\n  --database app \\\n  --user alice"
	tokens := Lex(input)
	if !IsContinuation(Lex("exec grant-readonly \\")) {
		t.Fatalf("expected continuation")
	}
	if got := JoinLogicalLines(input); got != "exec grant-readonly --database app --user alice" {
		t.Fatalf("joined = %q", got)
	}
	if len(tokens) == 0 {
		t.Fatalf("expected tokens")
	}
}

func assertToken(t *testing.T, token Token, tokenType TokenType, literal string, start int, end int) {
	t.Helper()
	if token.Type != tokenType || token.Literal != literal || token.StartRune != start || token.EndRune != end {
		t.Fatalf("token = %#v, want type=%v literal=%q range=%d:%d", token, tokenType, literal, start, end)
	}
}
