package ansi

import (
	"strconv"
	"strings"
	"unicode/utf8"
	"unsafe"
)

func unsafeBytes(s string) []byte {
	return unsafe.Slice(unsafe.StringData(s), len(s))
}

func isDigit(c uint8) bool {
	return c >= '0' && c <= '9'
}

func isControlStart(c uint8) bool {
	return c == '\\' || c == '[' || c == '(' || c == ')'
}

func findByteEitherAnsi(s []byte, a, b byte) int {
	for i, c := range s {
		if c == a || c == b {
			return i
		}
	}
	return -1
}

func matchOSC(s string, start int) int {
	i := start
	idx := findByteEitherAnsi(unsafeBytes(s[i:]), '\x07', '\x1b')
	if idx < 0 {
		return -1
	}
	i += idx
	if s[i] == '\x07' {
		return i + 1
	}
	if i < len(s)-1 && s[i+1] == '\\' {
		return i + 2
	}
	if s[:i+1] == "\x1b]8;;\x1b" {
		return i + 1
	}
	return -1
}

func matchCSI(s string) int {
	i := 2
	for ; i < len(s); i++ {
		c := s[i]
		switch c {
		case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9', ';', ':', '?':
			continue
		default:
			if c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c == '@' {
				return i + 1
			}
			return -1
		}
	}
	return -1
}

func FindEscapeSequence(s string) (int, int) {
	i := 0
	for ; i < len(s); i++ {
		switch s[i] {
		case '\x0e', '\x0f', '\x1b', '\x08', '\n':
			goto scan
		}
	}
	return -1, -1

scan:
	for ; i < len(s); i++ {
		switch s[i] {
		case '\n':
			return i, i + 1
		case '\x08':
			if i > 0 && s[i-1] != '\n' {
				if s[i-1] < utf8.RuneSelf {
					return i - 1, i + 1
				}
				_, n := utf8.DecodeLastRuneInString(s[:i])
				return i - n, i + 1
			}
		case '\x1b':
			if i+2 < len(s) && isControlStart(s[i+1]) {
				if j := matchCSI(s[i:]); j != -1 {
					return i, i + j
				}
			}
			if i+5 < len(s) && s[i+1] == ']' {
				j := 2
				for ; i+j < len(s) && isDigit(s[i+j]); j++ {
				}
				if j > 2 && i+j+1 < len(s) && (s[i+j] == ';' || s[i+j] == ':') && s[i+j+1] >= '\x20' {
					if k := matchOSC(s[i:], j+2); k != -1 {
						return i, i + k
					}
				}
			}
			if i+1 < len(s) && s[i+1] != '\n' {
				if s[i+1] < utf8.RuneSelf {
					return i, i + 2
				}
				_, n := utf8.DecodeRuneInString(s[i+1:])
				return i, i + n + 1
			}
		case '\x0e', '\x0f':
			return i, i + 1
		}
	}
	return -1, -1
}

func ExtractColors(str string, state *State, proc func(string, *State) bool) (string, *[]ColorRange, *State) {
	offsets := make([]ColorRange, 0, 32)
	if state != nil {
		offsets = append(offsets, ColorRange{[2]int32{0, 0}, *state})
	}

	var pstate *State
	var output strings.Builder
	prevIdx := 0
	runeCount := 0

	for idx := 0; idx < len(str); {
		start, end := FindEscapeSequence(str[idx:])
		if start == -1 {
			break
		}
		start += idx
		idx += end

		prev := str[prevIdx:start]
		if proc != nil && !proc(prev, state) {
			return "", nil, nil
		}
		prevIdx = idx

		if len(prev) != 0 {
			runeCount += utf8.RuneCountInString(prev)
			if output.Cap() == 0 {
				output.Grow(len(str))
			}
			output.WriteString(prev)
		}

		code := str[start:idx]
		newState := InterpretCode(code, state)
		if code == "\n" || !newState.Equals(state) {
			if state != nil {
				offsets[len(offsets)-1].Span[1] = int32(runeCount)
			}
			if code == "\n" {
				output.WriteRune('\n')
				runeCount++
				if newState.LineBg >= 0 {
					marker := newState
					marker.Attr |= StyleFullBg
					offsets = append(offsets, ColorRange{
						[2]int32{int32(runeCount), int32(runeCount)},
						marker,
					})
					newState.LineBg = -1
				}
			}
			if newState.HasColor() {
				if pstate == nil {
					pstate = &State{}
				}
				*pstate = newState
				state = pstate
				offsets = append(offsets, ColorRange{
					[2]int32{int32(runeCount), int32(runeCount)},
					newState,
				})
			} else {
				state = nil
			}
		}
	}

	var rest string
	var trimmed string
	if prevIdx == 0 {
		rest = str
		trimmed = str
	} else {
		rest = str[prevIdx:]
		output.WriteString(rest)
		trimmed = output.String()
	}
	if proc != nil {
		proc(rest, state)
	}
	if len(offsets) > 0 {
		if len(rest) > 0 && state != nil {
			runeCount += utf8.RuneCountInString(rest)
			offsets[len(offsets)-1].Span[1] = int32(runeCount)
		}
		a := make([]ColorRange, len(offsets))
		copy(a, offsets)
		return trimmed, &a, state
	}
	return trimmed, nil, state
}

func ParseAnsiCode(s string) (int, byte, string) {
	var remaining string
	var sep byte
	i := -1
	for j := 0; j < len(s); j++ {
		if s[j] == ';' || s[j] == ':' {
			i = j
			break
		}
	}
	if i >= 0 {
		sep = s[i]
		remaining = s[i+1:]
		s = s[:i]
	}
	if len(s) > 0 {
		code := 0
		for _, ch := range unsafeBytes(s) {
			ch -= '0'
			if ch > 9 {
				return -1, sep, remaining
			}
			code = code*10 + int(ch)
		}
		return code, sep, remaining
	}
	return -1, sep, remaining
}

func InterpretCode(code string, prevState *State) State {
	if code == "\n" {
		if prevState != nil {
			return *prevState
		}
		return EmptyState()
	}

	var st State
	if prevState == nil {
		st = EmptyState()
	} else {
		st = State{prevState.Fg, prevState.Bg, prevState.Ul, prevState.Attr, prevState.LineBg, prevState.Link}
	}

	if code[0] != '\x1b' || code[1] != '[' || code[len(code)-1] != 'm' {
		if prevState != nil && (strings.HasSuffix(code, "0K") || strings.HasSuffix(code, "[K")) {
			st.LineBg = prevState.Bg
		} else if strings.HasPrefix(code, "\x1b]8;") && (strings.HasSuffix(code, "\x1b\\") || strings.HasSuffix(code, "\a")) {
			stLen := 2
			if strings.HasSuffix(code, "\a") {
				stLen = 1
			}
			if len(code) == 5+stLen && code[4] == ';' {
				st.Link = nil
			} else if pEnd := strings.IndexRune(code[4:], ';'); pEnd >= 0 {
				params := code[4 : 4+pEnd]
				uri := code[5+pEnd : len(code)-stLen]
				st.Link = &Hyperlink{URI: uri, Params: params}
			}
		}
		return st
	}

	resetAll := func() {
		st.Fg = -1
		st.Bg = -1
		st.Ul = -1
		st.Attr = 0
	}

	if len(code) <= 3 {
		resetAll()
		return st
	}
	body := code[2 : len(code)-1]

	colorState := 0
	ptr := &st.Fg

	count := 0
	for len(body) != 0 {
		var num int
		var sep byte
		num, sep, body = ParseAnsiCode(body)
		if num != -1 {
			count++
			switch colorState {
			case 0:
				switch num {
				case 38:
					ptr = &st.Fg
					colorState++
				case 48:
					ptr = &st.Bg
					colorState++
				case 58:
					ptr = &st.Ul
					colorState++
				case 39:
					st.Fg = -1
				case 49:
					st.Bg = -1
				case 59:
					st.Ul = -1
				case 1:
					st.Attr |= StyleBold
				case 2:
					st.Attr |= StyleDim
				case 3:
					st.Attr |= StyleItalic
				case 4:
					if sep == ':' {
						var subNum int
						subNum, _, body = ParseAnsiCode(body)
						st.Attr &^= UlStyleMask
						switch subNum {
						case 0:
							st.Attr &^= StyleUnderline
						case 1:
							st.Attr |= StyleUnderline
						case 2:
							st.Attr |= StyleUnderline | UlDouble
						case 3:
							st.Attr |= StyleUnderline | UlCurly
						case 4:
							st.Attr |= StyleUnderline | UlDotted
						case 5:
							st.Attr |= StyleUnderline | UlDashed
						default:
							st.Attr |= StyleUnderline
						}
					} else {
						st.Attr |= StyleUnderline
					}
				case 5:
					st.Attr |= StyleBlink
				case 7:
					st.Attr |= StyleReverse
				case 9:
					st.Attr |= StyleStrike
				case 22:
					st.Attr &^= StyleBold
					st.Attr &^= StyleDim
				case 23:
					st.Attr &^= StyleItalic
				case 24:
					st.Attr &^= StyleUnderline
					st.Attr &^= UlStyleMask
				case 25:
					st.Attr &^= StyleBlink
				case 27:
					st.Attr &^= StyleReverse
				case 29:
					st.Attr &^= StyleStrike
				case 0:
					resetAll()
					colorState = 0
				default:
					if num >= 30 && num <= 37 {
						st.Fg = ColorValue(num - 30)
					} else if num >= 40 && num <= 47 {
						st.Bg = ColorValue(num - 40)
					} else if num >= 90 && num <= 97 {
						st.Fg = ColorValue(num - 90 + 8)
					} else if num >= 100 && num <= 107 {
						st.Bg = ColorValue(num - 100 + 8)
					}
				}
			case 1:
				switch num {
				case 2:
					colorState = 10
				case 5:
					colorState++
				default:
					colorState = 0
				}
			case 2:
				*ptr = ColorValue(num)
				colorState = 0
			case 10:
				*ptr = ColorValue(1<<24) | ColorValue(num<<16)
				colorState++
			case 11:
				*ptr = *ptr | ColorValue(num<<8)
				colorState++
			case 12:
				*ptr = *ptr | ColorValue(num)
				colorState = 0
			}
		}
	}

	if count == 0 {
		resetAll()
	}
	if colorState > 0 {
		*ptr = -1
	}
	return st
}

func (s *State) ToString() string {
	if !s.HasColor() {
		return ""
	}
	ret := ""
	if s.Attr&StyleBold > 0 || s.Attr&StyleBoldLock > 0 {
		ret += "1;"
	}
	if s.Attr&StyleDim > 0 {
		ret += "2;"
	}
	if s.Attr&StyleItalic > 0 {
		ret += "3;"
	}
	if s.Attr&StyleUnderline > 0 {
		switch s.Attr.UnderlineKind() {
		case UlDouble:
			ret += "4:2;"
		case UlCurly:
			ret += "4:3;"
		case UlDotted:
			ret += "4:4;"
		case UlDashed:
			ret += "4:5;"
		default:
			ret += "4;"
		}
	}
	if s.Attr&StyleBlink > 0 {
		ret += "5;"
	}
	if s.Attr&StyleReverse > 0 {
		ret += "7;"
	}
	if s.Attr&StyleStrike > 0 {
		ret += "9;"
	}
	ret += fgBgString(s.Fg, 30) + fgBgString(s.Bg, 40)
	if s.Ul != -1 {
		ret += ulColorString(s.Ul)
	}
	ret = "\x1b[" + strings.TrimSuffix(ret, ";") + "m"
	return ret
}

func fgBgString(color ColorValue, offset int) string {
	col := int(color)
	ret := ""
	if col == -1 {
		ret += strconv.Itoa(offset + 9)
	} else if col < 8 {
		ret += strconv.Itoa(offset + col)
	} else if col < 16 {
		ret += strconv.Itoa(offset - 30 + 90 + col - 8)
	} else if col < 256 {
		ret += strconv.Itoa(offset+8) + ";5;" + strconv.Itoa(col)
	} else if col >= (1 << 24) {
		r := strconv.Itoa((col >> 16) & 0xff)
		g := strconv.Itoa((col >> 8) & 0xff)
		b := strconv.Itoa(col & 0xff)
		ret += strconv.Itoa(offset+8) + ";2;" + r + ";" + g + ";" + b
	}
	return ret + ";"
}

func ulColorString(color ColorValue) string {
	col := int(color)
	if col < 0 {
		return ""
	}
	if col >= (1 << 24) {
		r := strconv.Itoa((col >> 16) & 0xff)
		g := strconv.Itoa((col >> 8) & 0xff)
		b := strconv.Itoa(col & 0xff)
		return "58;2;" + r + ";" + g + ";" + b + ";"
	}
	return "58;5;" + strconv.Itoa(col) + ";"
}
