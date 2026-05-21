package editor

import (
	"bufio"
	"errors"
	"io"
	"unicode/utf8"
)

type KeyReader struct {
	reader *bufio.Reader
}

func NewKeyReader(reader *bufio.Reader) *KeyReader {
	return &KeyReader{reader: reader}
}

func (r *KeyReader) ReadEvent() (KeyEvent, error) {
	b, err := r.reader.ReadByte()
	if err != nil {
		return KeyEvent{}, err
	}

	switch b {
	case '\n', '\r':
		return KeyEvent{Type: KeyEnter}, nil
	case '\t':
		return KeyEvent{Type: KeyTab}, nil
	case 1:
		return KeyEvent{Type: KeyCtrlA}, nil
	case 3:
		return KeyEvent{Type: KeyCtrlC}, nil
	case 4:
		return KeyEvent{Type: KeyCtrlD}, nil
	case 5:
		return KeyEvent{Type: KeyCtrlE}, nil
	case 8, 127:
		return KeyEvent{Type: KeyBackspace}, nil
	case 27:
		return r.readEscape()
	default:
		if b < utf8.RuneSelf {
			return KeyEvent{Type: KeyRune, Rune: rune(b)}, nil
		}
		if err := r.reader.UnreadByte(); err != nil {
			return KeyEvent{}, err
		}
		value, _, err := r.reader.ReadRune()
		if err != nil {
			return KeyEvent{}, err
		}
		return KeyEvent{Type: KeyRune, Rune: value}, nil
	}
}

func (r *KeyReader) readEscape() (KeyEvent, error) {
	first, err := r.reader.ReadByte()
	if err != nil {
		if errors.Is(err, io.EOF) {
			return KeyEvent{Type: KeyIgnored}, nil
		}
		return KeyEvent{}, err
	}

	switch first {
	case '[':
		return r.readCSI()
	case 'O':
		return r.readSS3()
	default:
		if first < utf8.RuneSelf {
			return KeyEvent{Type: KeyAlt, Rune: rune(first)}, nil
		}
		if err := r.reader.UnreadByte(); err != nil {
			return KeyEvent{}, err
		}
		value, _, err := r.reader.ReadRune()
		if err != nil {
			return KeyEvent{}, err
		}
		return KeyEvent{Type: KeyAlt, Rune: value}, nil
	}
}

func (r *KeyReader) readCSI() (KeyEvent, error) {
	sequence := make([]byte, 0, 4)
	for len(sequence) < 8 {
		b, err := r.reader.ReadByte()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return KeyEvent{Type: KeyIgnored}, nil
			}
			return KeyEvent{}, err
		}
		sequence = append(sequence, b)
		if (b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z') || b == '~' {
			break
		}
	}

	switch string(sequence) {
	case "A":
		return KeyEvent{Type: KeyUp}, nil
	case "B":
		return KeyEvent{Type: KeyDown}, nil
	case "C":
		return KeyEvent{Type: KeyRight}, nil
	case "D":
		return KeyEvent{Type: KeyLeft}, nil
	case "H", "1~", "7~":
		return KeyEvent{Type: KeyHome}, nil
	case "F", "4~", "8~":
		return KeyEvent{Type: KeyEnd}, nil
	case "3~":
		return KeyEvent{Type: KeyDelete}, nil
	default:
		return KeyEvent{Type: KeyIgnored}, nil
	}
}

func (r *KeyReader) readSS3() (KeyEvent, error) {
	b, err := r.reader.ReadByte()
	if err != nil {
		if errors.Is(err, io.EOF) {
			return KeyEvent{Type: KeyIgnored}, nil
		}
		return KeyEvent{}, err
	}

	switch b {
	case 'H':
		return KeyEvent{Type: KeyHome}, nil
	case 'F':
		return KeyEvent{Type: KeyEnd}, nil
	default:
		return KeyEvent{Type: KeyIgnored}, nil
	}
}
