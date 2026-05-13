package ansi

type TextStyle int32

const (
	StyleNone      = TextStyle(0)
	StyleBold      = TextStyle(1)
	StyleDim       = TextStyle(1 << 1)
	StyleItalic    = TextStyle(1 << 2)
	StyleUnderline = TextStyle(1 << 3)
	StyleBlink     = TextStyle(1 << 4)
	StyleReverse   = TextStyle(1 << 6)
	StyleStrike    = TextStyle(1 << 7)

	StyleRegular  = TextStyle(1 << 8)
	StyleClear    = TextStyle(1 << 9)
	StyleBoldLock = TextStyle(1 << 10)
	StyleFullBg   = TextStyle(1 << 11)

	ulStyleShift = 13
	UlStyleMask  = TextStyle(0b111 << ulStyleShift)
	UlDouble     = TextStyle(0b001 << ulStyleShift)
	UlCurly      = TextStyle(0b010 << ulStyleShift)
	UlDotted     = TextStyle(0b011 << ulStyleShift)
	UlDashed     = TextStyle(0b100 << ulStyleShift)
)

func (a TextStyle) UnderlineKind() TextStyle {
	return a & UlStyleMask
}

type ColorValue int32

type Hyperlink struct {
	URI    string
	Params string
}

type State struct {
	Fg     ColorValue
	Bg     ColorValue
	Ul     ColorValue
	Attr   TextStyle
	LineBg ColorValue
	Link   *Hyperlink
}

func (s *State) HasColor() bool {
	return s.Fg != -1 || s.Bg != -1 || s.Ul != -1 || s.Attr > 0 || s.LineBg >= 0 || s.Link != nil
}

func (s *State) Equals(other *State) bool {
	if other == nil {
		return !s.HasColor()
	}
	return s.Fg == other.Fg && s.Bg == other.Bg && s.Ul == other.Ul &&
		s.Attr == other.Attr && s.LineBg == other.LineBg && s.Link == other.Link
}

func EmptyState() State {
	return State{Fg: -1, Bg: -1, Ul: -1, Attr: 0, LineBg: -1, Link: nil}
}

type ColorRange struct {
	Span  [2]int32
	Style State
}
