package editor

import "strings"

type HistoryNavigator struct {
	entries []string
	index   int
	draft   string
}

func NewHistoryNavigator(entries []string) *HistoryNavigator {
	navigator := &HistoryNavigator{}
	navigator.SetEntries(entries)
	return navigator
}

func (n *HistoryNavigator) SetEntries(entries []string) {
	n.entries = append([]string(nil), entries...)
	n.index = len(n.entries)
	n.draft = ""
}

func (n *HistoryNavigator) Entries() []string {
	return append([]string(nil), n.entries...)
}

func (n *HistoryNavigator) Up(current string) string {
	if len(n.entries) == 0 {
		return current
	}
	if n.index == len(n.entries) {
		n.draft = current
	}
	if n.index > 0 {
		n.index--
	}
	return n.entries[n.index]
}

func (n *HistoryNavigator) Down(current string) string {
	if len(n.entries) == 0 {
		return current
	}
	if n.index >= len(n.entries) {
		return current
	}
	if n.index < len(n.entries)-1 {
		n.index++
		return n.entries[n.index]
	}
	n.index = len(n.entries)
	return n.draft
}

func (n *HistoryNavigator) Reset() {
	n.index = len(n.entries)
	n.draft = ""
}

func (n *HistoryNavigator) Add(entry string) bool {
	entry = strings.TrimSpace(entry)
	if entry == "" {
		n.Reset()
		return false
	}
	if len(n.entries) > 0 && n.entries[len(n.entries)-1] == entry {
		n.Reset()
		return false
	}
	n.entries = append(n.entries, entry)
	n.Reset()
	return true
}
