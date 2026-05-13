package charutil

import (
	"testing"
)

func TestToCharsAscii(t *testing.T) {
	input := []byte("hello world")
	c := ToChars(input)
	if !c.IsBytes() {
		t.Error("Expected ascii mode for pure ASCII input")
	}
	if c.Length() != 11 {
		t.Errorf("Expected length 11, got %d", c.Length())
	}
	if c.ToString() != "hello world" {
		t.Errorf("Expected 'hello world', got '%s'", c.ToString())
	}
	if c.Get(0) != 'h' {
		t.Errorf("Expected 'h', got '%c'", c.Get(0))
	}
	if c.Get(5) != ' ' {
		t.Errorf("Expected ' ', got '%c'", c.Get(5))
	}
}

func TestToCharsUnicode(t *testing.T) {
	input := []byte("hello한글world")
	c := ToChars(input)
	if c.IsBytes() {
		t.Error("Expected rune mode for Unicode input")
	}
	expectedLen := 13 // hello(5) + 한글(2) + world(5) + extra rune count = 5+2+5 = 12 ... wait
	// "hello" = 5 chars, "한" = 1, "글" = 1, "world" = 5 => 12 runes
	expectedLen = 12
	if c.Length() != expectedLen {
		t.Errorf("Expected length %d, got %d", expectedLen, c.Length())
	}
	if c.Get(5) != '한' {
		t.Errorf("Expected '한', got '%c'", c.Get(5))
	}
}

func TestCharsLength(t *testing.T) {
	c := ToChars([]byte("hello 한글 world"))
	if c.IsBytes() {
		t.Error("Expected rune mode")
	}
	// h-e-l-l-o- -한-글- -w-o-r-l-d = 14
	if c.Length() != 14 {
		t.Errorf("Expected 14, got %d", c.Length())
	}
}

func TestCharsToString(t *testing.T) {
	orig := "testing 123"
	c := ToChars([]byte(orig))
	if c.ToString() != orig {
		t.Errorf("Expected '%s', got '%s'", orig, c.ToString())
	}
}

func TestTrimLength(t *testing.T) {
	tests := []struct {
		input    string
		expected uint16
	}{
		{"hello", 5},
		{"  hello  ", 5},
		{"  hello world  ", 11},
		{"   ", 0},
		{"", 0},
		{"a", 1},
	}
	for _, tt := range tests {
		c := ToChars([]byte(tt.input))
		got := c.TrimLength()
		if got != tt.expected {
			t.Errorf("TrimLength(%q) = %d, want %d", tt.input, got, tt.expected)
		}
		// Should be cached
		got2 := c.TrimLength()
		if got2 != tt.expected {
			t.Errorf("Cached TrimLength(%q) = %d, want %d", tt.input, got2, tt.expected)
		}
	}
}

func TestLeadingTrailingWhitespaces(t *testing.T) {
	c := ToChars([]byte("  hello  "))
	if c.LeadingWhitespaces() != 2 {
		t.Errorf("Expected 2 leading, got %d", c.LeadingWhitespaces())
	}
	if c.TrailingWhitespaces() != 2 {
		t.Errorf("Expected 2 trailing, got %d", c.TrailingWhitespaces())
	}
}

func TestToRunes(t *testing.T) {
	c := ToChars([]byte("abc"))
	runes := c.ToRunes()
	if len(runes) != 3 {
		t.Errorf("Expected 3 runes, got %d", len(runes))
	}
	if runes[0] != 'a' || runes[1] != 'b' || runes[2] != 'c' {
		t.Errorf("Unexpected runes: %v", runes)
	}
}

func TestCopyRunes(t *testing.T) {
	c := ToChars([]byte("abcdef"))
	dest := make([]rune, 3)
	c.CopyRunes(dest, 2)
	if string(dest) != "cde" {
		t.Errorf("Expected 'cde', got '%s'", string(dest))
	}
}

func TestPrepend(t *testing.T) {
	c := ToChars([]byte("world"))
	c.Prepend("hello ")
	if c.ToString() != "hello world" {
		t.Errorf("Expected 'hello world', got '%s'", c.ToString())
	}
}

func TestFromRunes(t *testing.T) {
	runes := []rune("hello世界")
	c := FromRunes(runes)
	if c.IsBytes() {
		t.Error("Expected rune mode")
	}
	if c.Length() != 7 {
		t.Errorf("Expected length 7, got %d", c.Length())
	}
	if c.Get(5) != '世' {
		t.Errorf("Expected '世', got '%c'", c.Get(5))
	}
}

func TestSlabCreation(t *testing.T) {
	s := NewSlab(100, 50)
	if len(s.I16) != 100 {
		t.Errorf("Expected I16 len 100, got %d", len(s.I16))
	}
	if len(s.I32) != 50 {
		t.Errorf("Expected I32 len 50, got %d", len(s.I32))
	}
}
