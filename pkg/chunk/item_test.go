package chunk

import (
	"testing"

	"github.com/fzf/finder/pkg/charutil"
)

func TestStringPtr(t *testing.T) {
	orig := []byte("\x1b[34mfoo")
	text := []byte("\x1b[34mbar")
	item := Item{OrigText: &orig, Text: charutil.ToChars(text)}
	if item.AsString(true) != "foo" || item.AsString(false) != string(orig) {
		t.Fail()
	}
	if item.AsString(true) != "foo" {
		t.Fail()
	}
	item.OrigText = nil
	if item.AsString(true) != string(text) || item.AsString(false) != string(text) {
		t.Fail()
	}
}

func TestItemIndex(t *testing.T) {
	item := Item{Text: charutil.Chars{Index: 42}}
	if item.Index() != 42 {
		t.Errorf("expected 42, got %d", item.Index())
	}
}

func TestItemColors(t *testing.T) {
	item := Item{}
	if len(item.GetColors()) != 0 {
		t.Error("nil colors should return empty slice")
	}
}

func TestItemTrimLength(t *testing.T) {
	item := Item{Text: charutil.ToChars([]byte("  hello  "))}
	if item.TrimLength() != 5 {
		t.Errorf("expected 5, got %d", item.TrimLength())
	}
}
