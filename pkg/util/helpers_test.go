package util

import (
	"math"
	"testing"
)

func TestConstrain(t *testing.T) {
	if Constrain(5, 1, 10) != 5 {
		t.Error("Within range")
	}
	if Constrain(0, 1, 10) != 1 {
		t.Error("Below min")
	}
	if Constrain(15, 1, 10) != 10 {
		t.Error("Above max")
	}
}

func TestAsUint16(t *testing.T) {
	if AsUint16(5) != 5 {
		t.Error("Normal value")
	}
	if AsUint16(-10) != 0 {
		t.Error("Negative should be 0")
	}
	if AsUint16(math.MaxUint16) != math.MaxUint16 {
		t.Error("MaxUint16")
	}
	if AsUint16(math.MinInt32) != 0 {
		t.Error("MinInt32 should be 0")
	}
	if AsUint16(math.MaxUint16+1) != math.MaxUint16 {
		t.Error("MaxUint16+1 should clamp")
	}
}

func TestOnce(t *testing.T) {
	fn := Once(false)
	if fn() != false {
		t.Error("First call should return false")
	}
	if fn() != true {
		t.Error("Second call should return true")
	}
	if fn() != true {
		t.Error("Third call should return true")
	}

	fn2 := Once(true)
	if fn2() != true {
		t.Error("First call should return true")
	}
	if fn2() != false {
		t.Error("Second call should return false")
	}
}

func TestRuneDisplayWidth(t *testing.T) {
	w, overflow := RuneDisplayWidth([]rune("hello"), 0, 8, 100)
	if w != 5 || overflow != -1 {
		t.Errorf("Expected (5, -1), got (%d, %d)", w, overflow)
	}

	w, overflow = RuneDisplayWidth([]rune("hello"), 0, 8, 3)
	if overflow != 3 {
		t.Errorf("Expected overflow at 3, got %d", overflow)
	}

	w, overflow = RuneDisplayWidth([]rune("hello"), 0, 8, 0)
	if overflow != 0 {
		t.Errorf("Expected overflow at 0, got %d", overflow)
	}
}

func TestTruncateString(t *testing.T) {
	runes, w := TruncateString("가나다라마", 7)
	if string(runes) != "가나다" || w != 6 {
		t.Errorf("Expected '가나다' width 6, got '%s' width %d", string(runes), w)
	}
}

func TestRepeatToFill(t *testing.T) {
	result := RepeatToFill("ab", 2, 10)
	if result != "ababababab" {
		t.Errorf("Expected 'ababababab', got '%s'", result)
	}
}

func TestStringWidth(t *testing.T) {
	if StringWidth("abc") != 3 {
		t.Error("ascii width")
	}
	if StringWidth("가") != 2 {
		t.Error("wide char width")
	}
}

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		v1, v2 string
		want   int
	}{
		{"1.0", "1.0", 0},
		{"1.1", "1.0", 1},
		{"1.0", "1.1", -1},
		{"2.0", "1.9", 1},
		{"1.0.0", "1.0", 0},
		{"", "1.0", -1},
		{"1.0", "", 1},
	}
	for _, tt := range tests {
		got := CompareVersions(tt.v1, tt.v2)
		if got != tt.want {
			t.Errorf("CompareVersions(%q, %q) = %d, want %d", tt.v1, tt.v2, got, tt.want)
		}
	}
}

func TestToKebabCase(t *testing.T) {
	if ToKebabCase("FooBar") != "foo-bar" {
		t.Error("Expected foo-bar")
	}
	if ToKebabCase("fooBarBaz") != "foo-bar-baz" {
		t.Error("Expected foo-bar-baz")
	}
}

func TestRunOnce(t *testing.T) {
	count := 0
	fn := RunOnce(func() { count++ })
	fn()
	fn()
	fn()
	if count != 1 {
		t.Errorf("Expected count 1, got %d", count)
	}
}
