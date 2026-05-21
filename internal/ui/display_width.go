package ui

import "unicode"

type runeInterval struct {
	first rune
	last  rune
}

var wideRuneIntervals = []runeInterval{
	{first: 0x1100, last: 0x115f},
	{first: 0x2329, last: 0x232a},
	{first: 0x2e80, last: 0xa4cf},
	{first: 0xac00, last: 0xd7a3},
	{first: 0xf900, last: 0xfaff},
	{first: 0xfe10, last: 0xfe19},
	{first: 0xfe30, last: 0xfe6f},
	{first: 0xff00, last: 0xff60},
	{first: 0xffe0, last: 0xffe6},
	{first: 0x1f300, last: 0x1faff},
	{first: 0x20000, last: 0x2fffd},
	{first: 0x30000, last: 0x3fffd},
}

func displayWidthString(value string) int {
	return displayWidthRunes([]rune(value))
}

func displayWidthRunes(runes []rune) int {
	width := 0
	for _, r := range runes {
		width += runeDisplayWidth(r)
	}
	return width
}

func runeDisplayWidth(r rune) int {
	switch {
	case r == 0:
		return 0
	case r < 32 || (r >= 0x7f && r < 0xa0):
		return 0
	case unicode.Is(unicode.Mn, r), unicode.Is(unicode.Me, r), unicode.Is(unicode.Cf, r):
		return 0
	case isWideRune(r):
		return 2
	default:
		return 1
	}
}

func isWideRune(r rune) bool {
	for _, interval := range wideRuneIntervals {
		if r >= interval.first && r <= interval.last {
			return true
		}
	}
	return false
}
