package history

import (
	"os"
	"testing"
)

func TestHistory(t *testing.T) {
	maxHistory := 50

	paths := []string{"/etc", "/proc"}
	for _, path := range paths {
		if _, e := NewHistory(path, maxHistory); e == nil {
			t.Error("Error expected for: " + path)
		}
	}

	f, _ := os.CreateTemp("", "fzf-history")
	f.Close()
	defer os.Remove(f.Name())

	{
		h, _ := NewHistory(f.Name(), maxHistory)
		for i := 0; i < maxHistory+10; i++ {
			h.Append("foobar")
		}
	}
	{
		h, _ := NewHistory(f.Name(), maxHistory)
		if len(h.Lines) != maxHistory+1 {
			t.Errorf("Expected: %d, actual: %d", maxHistory+1, len(h.Lines))
		}
		for i := range maxHistory {
			if h.Lines[i] != "foobar" {
				t.Error("Expected: foobar, actual: " + h.Lines[i])
			}
		}
	}
	{
		h, _ := NewHistory(f.Name(), maxHistory)
		h.Append("barfoo")
		h.Append("")
		h.Append("foobarbaz")
	}
	{
		h, _ := NewHistory(f.Name(), maxHistory)
		if len(h.Lines) != maxHistory+1 {
			t.Errorf("Expected: %d, actual: %d", maxHistory+1, len(h.Lines))
		}
		compare := func(idx int, exp string) {
			if h.Lines[idx] != exp {
				t.Errorf("Expected: %s, actual: %s", exp, h.Lines[idx])
			}
		}
		compare(maxHistory-3, "foobar")
		compare(maxHistory-2, "barfoo")
		compare(maxHistory-1, "foobarbaz")
	}
}

func TestHistoryNavigation(t *testing.T) {
	f, _ := os.CreateTemp("", "fzf-history-nav")
	f.Close()
	defer os.Remove(f.Name())

	h, _ := NewHistory(f.Name(), 100)
	h.Append("line1")
	h.Append("line2")
	h.Append("line3")

	h2, _ := NewHistory(f.Name(), 100)

	if h2.Current() != "" {
		t.Errorf("Current should be empty, got %q", h2.Current())
	}

	prev := h2.Previous()
	if prev != "line3" {
		t.Errorf("Expected line3, got %q", prev)
	}

	prev = h2.Previous()
	if prev != "line2" {
		t.Errorf("Expected line2, got %q", prev)
	}

	next := h2.Next()
	if next != "line3" {
		t.Errorf("Expected line3, got %q", next)
	}
}

func TestHistoryOverride(t *testing.T) {
	f, _ := os.CreateTemp("", "fzf-history-override")
	f.Close()
	defer os.Remove(f.Name())

	h, _ := NewHistory(f.Name(), 100)
	h.Append("original")

	h2, _ := NewHistory(f.Name(), 100)
	h2.Previous()
	h2.Override("modified")
	if h2.Current() != "modified" {
		t.Errorf("Expected modified, got %q", h2.Current())
	}

	h2.Next()
	h2.Override("current-line")
	if h2.Current() != "current-line" {
		t.Errorf("Expected current-line, got %q", h2.Current())
	}
}

func TestHistoryNewFile(t *testing.T) {
	f, _ := os.CreateTemp("", "fzf-history-new")
	name := f.Name()
	f.Close()
	os.Remove(name)

	h, err := NewHistory(name, 100)
	defer os.Remove(name)
	if err != nil {
		t.Fatal(err)
	}
	h.Append("test")

	h2, _ := NewHistory(name, 100)
	if len(h2.Lines) != 2 || h2.Lines[0] != "test" {
		t.Errorf("Expected [test, ''], got %v", h2.Lines)
	}
}
