package chunk

import (
	"math"

	"github.com/fzf/finder/pkg/ansi"
	"github.com/fzf/finder/pkg/charutil"
)

type Item struct {
	Text     charutil.Chars
	OrigText *[]byte
	Colors   *[]ansi.ColorRange
}

func (item *Item) Index() int32 {
	return item.Text.Index
}

var MinItem = Item{Text: charutil.Chars{Index: math.MinInt32}}

func (item *Item) TrimLength() uint16 {
	return item.Text.TrimLength()
}

func (item *Item) GetColors() []ansi.ColorRange {
	if item.Colors == nil {
		return []ansi.ColorRange{}
	}
	return *item.Colors
}

func (item *Item) AsString(stripAnsi bool) string {
	if item.OrigText != nil {
		if stripAnsi {
			trimmed, _, _ := ansi.ExtractColors(string(*item.OrigText), nil, nil)
			return trimmed
		}
		return string(*item.OrigText)
	}
	return item.Text.ToString()
}
