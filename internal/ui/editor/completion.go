package editor

import "sort"

type CompletionSession struct {
	BaseLine     string
	BaseCursor   int
	Suggestions  []Suggestion
	Index        int
	CommonResult CompletionResult
	HasCommon    bool
	ListShown    bool
}

func NewCompletionSession(baseLine string, baseCursor int, suggestions []Suggestion) *CompletionSession {
	if len(suggestions) == 0 {
		return nil
	}
	cloned := make([]Suggestion, 0, len(suggestions))
	seen := make(map[string]struct{}, len(suggestions))
	for _, suggestion := range suggestions {
		key := suggestion.Value + "|" + suggestion.Description + "|" + suggestion.Category
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		cloned = append(cloned, suggestion)
	}
	if len(cloned) == 0 {
		return nil
	}
	return &CompletionSession{
		BaseLine:    baseLine,
		BaseCursor:  baseCursor,
		Suggestions: cloned,
	}
}

func (s *CompletionSession) Current() Suggestion {
	if s == nil || len(s.Suggestions) == 0 {
		return Suggestion{}
	}
	if s.Index < 0 || s.Index >= len(s.Suggestions) {
		s.Index = 0
	}
	return s.Suggestions[s.Index]
}

func (s *CompletionSession) Advance() {
	if s == nil || len(s.Suggestions) == 0 {
		return
	}
	s.Index = (s.Index + 1) % len(s.Suggestions)
}

func (s *CompletionSession) Contains(line string, cursor int) bool {
	if s == nil {
		return false
	}
	if s.HasCommon {
		commonLine, commonCursor := ApplyCompletion(s.BaseLine, s.CommonResult)
		if line == commonLine && cursor == commonCursor {
			return true
		}
	}
	for _, suggestion := range s.Suggestions {
		nextLine, nextCursor := ApplyCompletion(s.BaseLine, suggestion.Result)
		if line == nextLine && cursor == nextCursor {
			return true
		}
	}
	return false
}

func ApplyCompletion(line string, result CompletionResult) (string, int) {
	source := []rune(line)
	edits := append([]CompletionEdit(nil), result.Edits...)
	sort.SliceStable(edits, func(i int, j int) bool {
		return edits[i].StartRune < edits[j].StartRune
	})

	out := make([]rune, 0, len(source))
	current := 0
	for _, edit := range edits {
		start := clamp(edit.StartRune, 0, len(source))
		end := clamp(edit.EndRune, start, len(source))
		if current < start {
			out = append(out, source[current:start]...)
		}
		out = append(out, []rune(edit.Text)...)
		current = end
	}
	out = append(out, source[current:]...)
	cursor := clamp(result.Cursor, 0, len(out))
	return string(out), cursor
}

func CommonSuggestionResult(baseLine string, suggestions []Suggestion) (CompletionResult, bool) {
	if len(suggestions) < 2 {
		return CompletionResult{}, false
	}

	baseEdits := suggestions[0].Result.Edits
	if len(baseEdits) != 1 {
		return CompletionResult{}, false
	}
	baseEdit := baseEdits[0]
	prefix := []rune(baseEdit.Text)
	for _, suggestion := range suggestions[1:] {
		if len(suggestion.Result.Edits) != 1 {
			return CompletionResult{}, false
		}
		nextEdit := suggestion.Result.Edits[0]
		if nextEdit.StartRune != baseEdit.StartRune || nextEdit.EndRune != baseEdit.EndRune {
			return CompletionResult{}, false
		}
		prefix = commonPrefixRunes(prefix, []rune(nextEdit.Text))
		if len(prefix) == 0 {
			return CompletionResult{}, false
		}
	}
	line, cursor := ApplyCompletion(baseLine, CompletionResult{
		Edits: []CompletionEdit{{
			StartRune: baseEdit.StartRune,
			EndRune:   baseEdit.EndRune,
			Text:      string(prefix),
		}},
		Cursor: baseEdit.StartRune + len(prefix),
	})
	if line == baseLine {
		return CompletionResult{}, false
	}
	return CompletionResult{
		Edits: []CompletionEdit{{
			StartRune: baseEdit.StartRune,
			EndRune:   baseEdit.EndRune,
			Text:      string(prefix),
		}},
		Cursor: cursor,
	}, true
}

func MatchingSuggestionIndex(baseLine string, suggestions []Suggestion, line string, cursor int) int {
	for idx, suggestion := range suggestions {
		nextLine, nextCursor := ApplyCompletion(baseLine, suggestion.Result)
		if nextLine == line && nextCursor == cursor {
			return idx
		}
	}
	return -1
}

func commonPrefixRunes(left []rune, right []rune) []rune {
	limit := len(left)
	if len(right) < limit {
		limit = len(right)
	}
	index := 0
	for index < limit && left[index] == right[index] {
		index++
	}
	return append([]rune(nil), left[:index]...)
}
