package editor

import (
	"bufio"
	"bytes"
	"strings"
	"testing"
)

func TestKeyReaderParsesANSISequences(t *testing.T) {
	t.Parallel()

	reader := NewKeyReader(bufio.NewReader(strings.NewReader("\x1b[D\x1b[C\x1b[A\x1b[B\x1b[H\x1b[F\x1b[1~\x1b[4~\x1b[7~\x1b[8~\x1b[3~\x1bOH\x1bOF")))
	want := []KeyType{
		KeyLeft,
		KeyRight,
		KeyUp,
		KeyDown,
		KeyHome,
		KeyEnd,
		KeyHome,
		KeyEnd,
		KeyHome,
		KeyEnd,
		KeyDelete,
		KeyHome,
		KeyEnd,
	}

	for idx, expected := range want {
		event, err := reader.ReadEvent()
		if err != nil {
			t.Fatalf("ReadEvent[%d] error: %v", idx, err)
		}
		if event.Type != expected {
			t.Fatalf("ReadEvent[%d] = %v, want %v", idx, event.Type, expected)
		}
	}
}

func TestKeyReaderReadsUTF8AndAltRune(t *testing.T) {
	t.Parallel()

	reader := NewKeyReader(bufio.NewReader(bytes.NewBufferString("你\x1bb")))
	event, err := reader.ReadEvent()
	if err != nil {
		t.Fatalf("ReadEvent rune error: %v", err)
	}
	if event.Type != KeyRune || event.Rune != '你' {
		t.Fatalf("first event = %#v", event)
	}

	event, err = reader.ReadEvent()
	if err != nil {
		t.Fatalf("ReadEvent alt error: %v", err)
	}
	if event.Type != KeyAlt || event.Rune != 'b' {
		t.Fatalf("alt event = %#v", event)
	}
}
