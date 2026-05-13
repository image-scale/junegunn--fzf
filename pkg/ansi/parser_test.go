package ansi

import (
	"regexp"
	"strings"
	"testing"
)

var escapeRegex = regexp.MustCompile("(?:\x1b[\\[()][0-9;:]*[a-zA-Z@]|\x1b][0-9][;:][[:print:]]+(?:\x1b\\\\|\x07)|\x1b.|[\x0e\x0f]|.\x08|\n)")

func verifyParser(t testing.TB, str string) {
	t.Helper()
	toSlice := func(start, end int) []int {
		if start == -1 {
			return nil
		}
		return []int{start, end}
	}
	s := str
	for i := 0; ; i++ {
		got := toSlice(FindEscapeSequence(s))
		exp := escapeRegex.FindStringIndex(s)
		equal := len(got) == len(exp)
		if equal {
			for i := range got {
				if got[i] != exp[i] {
					equal = false
					break
				}
			}
		}
		if !equal {
			t.Errorf("%d: %q: got: %v want: %v", i, s, got, exp)
			return
		}
		if len(exp) == 0 {
			return
		}
		s = s[exp[1]:]
	}
}

const benchStr = "\x1b[38;5;81m\x1b[01;31m\x1b[Kkernel/\x1b[0m\x1b[38:5:81mbpf/" +
	"\x1b[0m\x1b[38:5:81mpreload/\x1b[0m\x1b[38;5;81miterators/" +
	"\x1b[0m\x1b[38:5:149mMakefile\x1b[m\x1b[K\x1b[0m"

func TestFindEscapeSequence(t *testing.T) {
	testStrs := []string{
		"\x1b[0mhello world",
		"\x1b[1mhello world",
		"椙\x1b[1m椙",
		"椙\x1b[1椙m椙",
		"\x1b[1mhello \x1b[mw\x1b7o\x1b8r\x1b(Bl\x1b[2@d",
		"\x1b[1mhello \x1b[Kworld",
		"hello \x1b[34;45;1mworld",
		"hello \x1b[34;45;1mwor\x1b[34;45;1mld",
		"hello \x1b[34;45;1mwor\x1b[0mld",
		"hello \x1b[34;48;5;233;1mwo\x1b[38;5;161mr\x1b[0ml\x1b[38;5;161md",
		"hello \x1b[38;5;38;48;5;48;1mwor\x1b[38;5;48;48;5;38ml\x1b[0md",
		"hello \x1b[32;1mworld",
		"hello world",
		"hello \x1b[0;38;5;200;48;5;100mworld",
		"\x1b椙",
		"椙\x08",
		"\n\x08",
		"X\x08",
		"",
		"\x1b]4;3;rgb:aa/bb/cc\x07 ",
		"\x1b]4;3;rgb:aa/bb/cc\x1b\\ ",
		benchStr,
	}
	for _, s := range testStrs {
		verifyParser(t, s)
	}
}

func TestExtractColors(t *testing.T) {
	assertRange := func(cr ColorRange, b, e int32, fg, bg ColorValue, bold bool) {
		t.Helper()
		var attr TextStyle
		if bold {
			attr = StyleBold
		}
		if cr.Span[0] != b || cr.Span[1] != e ||
			cr.Style.Fg != fg || cr.Style.Bg != bg || cr.Style.Attr != attr {
			t.Errorf("got %+v, want b=%d e=%d fg=%d bg=%d attr=%d", cr, b, e, fg, bg, attr)
		}
	}

	src := "hello world"
	var state *State
	check := func(assertion func(*[]ColorRange, *State)) {
		output, ranges, newState := ExtractColors(src, state, nil)
		state = newState
		if output != "hello world" {
			t.Errorf("Invalid output: %s", output)
		}
		assertion(ranges, state)
	}

	// Plain text
	check(func(r *[]ColorRange, s *State) {
		if r != nil {
			t.Error("Expected nil ranges for plain text")
		}
	})

	// Reset only
	state = nil
	src = "\x1b[0mhello world"
	check(func(r *[]ColorRange, s *State) {
		if r != nil {
			t.Error("Expected nil after reset")
		}
	})

	// Bold
	state = nil
	src = "\x1b[1mhello world"
	check(func(r *[]ColorRange, s *State) {
		if len(*r) != 1 {
			t.Fatalf("Expected 1 range, got %d", len(*r))
		}
		assertRange((*r)[0], 0, 11, -1, -1, true)
	})

	// Bold with non-SGR sequences mixed in
	state = nil
	src = "\x1b[1mhello \x1b[mw\x1b7o\x1b8r\x1b(Bl\x1b[2@d"
	check(func(r *[]ColorRange, s *State) {
		if len(*r) != 1 {
			t.Fatalf("Expected 1 range, got %d", len(*r))
		}
		assertRange((*r)[0], 0, 6, -1, -1, true)
	})

	// Bold with erase-to-EOL
	state = nil
	src = "\x1b[1mhello \x1b[Kworld"
	check(func(r *[]ColorRange, s *State) {
		if len(*r) != 1 {
			t.Fatalf("Expected 1 range, got %d", len(*r))
		}
		assertRange((*r)[0], 0, 11, -1, -1, true)
	})

	// fg + bg + bold
	state = nil
	src = "hello \x1b[34;45;1mworld"
	check(func(r *[]ColorRange, s *State) {
		if len(*r) != 1 {
			t.Fatalf("Expected 1 range, got %d", len(*r))
		}
		assertRange((*r)[0], 6, 11, 4, 5, true)
	})

	// Duplicate same state
	state = nil
	src = "hello \x1b[34;45;1mwor\x1b[34;45;1mld"
	check(func(r *[]ColorRange, s *State) {
		if len(*r) != 1 {
			t.Fatalf("Expected 1 range, got %d", len(*r))
		}
		assertRange((*r)[0], 6, 11, 4, 5, true)
	})

	// Color then reset
	state = nil
	src = "hello \x1b[34;45;1mwor\x1b[0mld"
	check(func(r *[]ColorRange, s *State) {
		if len(*r) != 1 {
			t.Fatalf("Expected 1 range, got %d", len(*r))
		}
		assertRange((*r)[0], 6, 9, 4, 5, true)
	})

	// 256-color and multiple segments
	state = nil
	src = "hello \x1b[34;48;5;233;1mwo\x1b[38;5;161mr\x1b[0ml\x1b[38;5;161md"
	check(func(r *[]ColorRange, s *State) {
		if len(*r) != 3 {
			t.Fatalf("Expected 3 ranges, got %d", len(*r))
		}
		assertRange((*r)[0], 6, 8, 4, 233, true)
		assertRange((*r)[1], 8, 9, 161, 233, true)
		assertRange((*r)[2], 10, 11, 161, -1, false)
	})

	// {38,48};5;{38,48}
	state = nil
	src = "hello \x1b[38;5;38;48;5;48;1mwor\x1b[38;5;48;48;5;38ml\x1b[0md"
	check(func(r *[]ColorRange, s *State) {
		if len(*r) != 2 {
			t.Fatalf("Expected 2 ranges, got %d", len(*r))
		}
		assertRange((*r)[0], 6, 9, 38, 48, true)
		assertRange((*r)[1], 9, 10, 48, 38, true)
	})

	// State carry-over
	src = "hello \x1b[32;1mworld"
	check(func(r *[]ColorRange, s *State) {
		if len(*r) != 1 {
			t.Fatalf("Expected 1 range, got %d", len(*r))
		}
		if s.Fg != 2 || s.Bg != -1 || s.Attr == 0 {
			t.Error("State not carried over correctly")
		}
		assertRange((*r)[0], 6, 11, 2, -1, true)
	})

	src = "hello world"
	check(func(r *[]ColorRange, s *State) {
		if len(*r) != 1 {
			t.Fatalf("Expected 1 range, got %d", len(*r))
		}
		if s.Fg != 2 || s.Bg != -1 || s.Attr == 0 {
			t.Error("State not carried over")
		}
		assertRange((*r)[0], 0, 11, 2, -1, true)
	})

	// 256-color with reset
	src = "hello \x1b[0;38;5;200;48;5;100mworld"
	check(func(r *[]ColorRange, s *State) {
		if len(*r) != 2 {
			t.Fatalf("Expected 2 ranges, got %d", len(*r))
		}
		if s.Fg != 200 || s.Bg != 100 || s.Attr > 0 {
			t.Error("Unexpected state")
		}
		assertRange((*r)[0], 0, 6, 2, -1, true)
		assertRange((*r)[1], 6, 11, 200, 100, false)
	})

	// 24-bit color
	state = nil
	var trueColor ColorValue = (1 << 24) + (180 << 16) + (190 << 8) + 254
	src = "\x1b[1mhello \x1b[22;1;38:2:180:190:254mworld"
	check(func(r *[]ColorRange, s *State) {
		if len(*r) != 2 {
			t.Fatalf("Expected 2 ranges, got %d", len(*r))
		}
		if s.Fg != trueColor || s.Attr != 1 {
			t.Errorf("Unexpected state: fg=%d attr=%d", s.Fg, s.Attr)
		}
		assertRange((*r)[0], 0, 6, -1, -1, true)
		assertRange((*r)[1], 6, 11, trueColor, -1, true)
	})

	// OSC 133 shell integration
	src = "\x1b]133;A\x1b\\hello \x1b]133;C\x1b\\world"
	check(func(r *[]ColorRange, s *State) {
		if len(*r) != 1 {
			t.Fatalf("Expected 1 range, got %d", len(*r))
		}
		assertRange((*r)[0], 0, 11, trueColor, -1, true)
	})
}

func TestCodeStringConversion(t *testing.T) {
	verify := func(code string, prevState *State, expected string) {
		t.Helper()
		st := InterpretCode(code, prevState)
		got := st.ToString()
		if expected != got {
			t.Errorf("code=%q: expected %q, got %q",
				strings.ReplaceAll(code, "\x1b[", "\\x1b["),
				strings.ReplaceAll(expected, "\x1b[", "\\x1b["),
				strings.ReplaceAll(got, "\x1b[", "\\x1b["))
		}
	}
	verify("\x1b[m", nil, "")
	verify("\x1b[m", &State{Attr: StyleBlink, Ul: -1, LineBg: -1}, "")
	verify("\x1b[0m", &State{Fg: 4, Bg: 4, Ul: -1, LineBg: -1}, "")
	verify("\x1b[;m", &State{Fg: 4, Bg: 4, Ul: -1, LineBg: -1}, "")
	verify("\x1b[;;m", &State{Fg: 4, Bg: 4, Ul: -1, LineBg: -1}, "")

	verify("\x1b[31m", nil, "\x1b[31;49m")
	verify("\x1b[41m", nil, "\x1b[39;41m")
	verify("\x1b[92m", nil, "\x1b[92;49m")
	verify("\x1b[102m", nil, "\x1b[39;102m")

	verify("\x1b[31m", &State{Fg: 4, Bg: 4, Ul: -1, LineBg: -1}, "\x1b[31;44m")
	verify("\x1b[1;2;31m", &State{Fg: 2, Bg: -1, Ul: -1, Attr: StyleReverse, LineBg: -1}, "\x1b[1;2;7;31;49m")
	verify("\x1b[38;5;100;48;5;200m", nil, "\x1b[38;5;100;48;5;200m")
	verify("\x1b[38:5:100:48:5:200m", nil, "\x1b[38;5;100;48;5;200m")
	verify("\x1b[48;5;100;38;5;200m", nil, "\x1b[38;5;200;48;5;100m")
	verify("\x1b[48;5;100;38;2;10;20;30;1m", nil, "\x1b[1;38;2;10;20;30;48;5;100m")
	verify("\x1b[48;5;100;38;2;10;20;30;7m",
		&State{Attr: StyleDim | StyleItalic, Fg: 1, Bg: 1, Ul: -1},
		"\x1b[2;3;7;38;2;10;20;30;48;5;100m")

	// Underline styles
	verify("\x1b[4:3m", nil, "\x1b[4:3;39;49m")
	verify("\x1b[4:2m", nil, "\x1b[4:2;39;49m")
	verify("\x1b[4:4m", nil, "\x1b[4:4;39;49m")
	verify("\x1b[4:5m", nil, "\x1b[4:5;39;49m")
	verify("\x1b[4:1m", nil, "\x1b[4;39;49m")

	// Underline color
	verify("\x1b[4;58;5;100m", nil, "\x1b[4;39;49;58;5;100m")
	verify("\x1b[4;58;2;255;0;128m", nil, "\x1b[4;39;49;58;2;255;0;128m")
	verify("\x1b[4:3;58;2;255;0;0m", nil, "\x1b[4:3;39;49;58;2;255;0;0m")
	verify("\x1b[59m", &State{Fg: 1, Bg: -1, Ul: 100, LineBg: -1}, "\x1b[31;49m")
}

func TestParseAnsiCode(t *testing.T) {
	tests := []struct {
		in   string
		num  int
		sep  byte
		rest string
	}{
		{"123", 123, 0, ""},
		{"1a", -1, 0, ""},
		{"1a;12", -1, ';', "12"},
		{"12;a", 12, ';', "a"},
		{"-2", -1, 0, ""},
		{"4:3", 4, ':', "3"},
		{"4:3;31", 4, ':', "3;31"},
		{"38:2:255:0:0", 38, ':', "2:255:0:0"},
		{"58:5:200", 58, ':', "5:200"},
		{"4;38:2:0:0:0", 4, ';', "38:2:0:0:0"},
	}
	for _, tt := range tests {
		n, sep, s := ParseAnsiCode(tt.in)
		if n != tt.num || s != tt.rest || sep != tt.sep {
			t.Errorf("%q: got (%d, %q, %q), want (%d, %q, %q)",
				tt.in, n, string(sep), s, tt.num, string(tt.sep), tt.rest)
		}
	}
}

func TestUnderlineStyles(t *testing.T) {
	// 4:0 = no underline
	st := InterpretCode("\x1b[4:0m", nil)
	if st.Attr&StyleUnderline != 0 {
		t.Error("4:0 should not set underline")
	}
	// 4:1 = single
	st = InterpretCode("\x1b[4:1m", nil)
	if st.Attr&StyleUnderline == 0 {
		t.Error("4:1 should set underline")
	}
	// 4:3 = curly, not italic
	st = InterpretCode("\x1b[4:3m", nil)
	if st.Attr&StyleUnderline == 0 {
		t.Error("4:3 should set underline")
	}
	if st.Attr.UnderlineKind() != UlCurly {
		t.Error("4:3 should be curly")
	}
	if st.Attr&StyleItalic != 0 {
		t.Error("4:3 should not set italic")
	}
	// 4:2;31 = double underline + red
	st = InterpretCode("\x1b[4:2;31m", nil)
	if st.Attr&StyleUnderline == 0 {
		t.Error("should set underline")
	}
	if st.Fg != 1 {
		t.Errorf("should set fg to 1, got %d", st.Fg)
	}
	if st.Attr&StyleDim != 0 {
		t.Error("should not set dim")
	}
	// 4;2 (semicolon) = underline + dim
	st = InterpretCode("\x1b[4;2m", nil)
	if st.Attr&StyleUnderline == 0 {
		t.Error("should set underline")
	}
	if st.Attr&StyleDim == 0 {
		t.Error("should set dim")
	}
}

func TestUnderlineColor(t *testing.T) {
	// 58:2:R:G:B
	st := InterpretCode("\x1b[58:2:255:0:0m", nil)
	if st.Fg != -1 || st.Bg != -1 {
		t.Errorf("should not affect fg/bg, got fg=%d bg=%d", st.Fg, st.Bg)
	}
	// 58:5:200
	st = InterpretCode("\x1b[58:5:200m", nil)
	if st.Fg != -1 || st.Bg != -1 {
		t.Errorf("should not affect fg/bg, got fg=%d bg=%d", st.Fg, st.Bg)
	}
	// 58:2 + 38:2
	st = InterpretCode("\x1b[58:2:255:0:0;38:2:0:255:0m", nil)
	expectedFg := ColorValue(1<<24 | 0<<16 | 255<<8 | 0)
	if st.Fg != expectedFg {
		t.Errorf("expected fg=%d, got %d", expectedFg, st.Fg)
	}
	if st.Bg != -1 {
		t.Errorf("bg should be -1, got %d", st.Bg)
	}
}
