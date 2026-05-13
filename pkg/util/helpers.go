package util

import (
	"cmp"
	"math"
	"strconv"
	"strings"
	"unicode/utf8"
)

func Constrain[T cmp.Ordered](val, minimum, maximum T) T {
	return max(min(val, maximum), minimum)
}

func AsUint16(val int) uint16 {
	if val > math.MaxUint16 {
		return math.MaxUint16
	} else if val < 0 {
		return 0
	}
	return uint16(val)
}

func Once(firstVal bool) func() bool {
	state := firstVal
	return func() bool {
		prev := state
		state = !firstVal
		return prev
	}
}

func RunOnce(f func()) func() {
	gate := Once(true)
	return func() {
		if gate() {
			f()
		}
	}
}

func RuneDisplayWidth(runes []rune, prefixWidth int, tabstop int, limit int) (int, int) {
	width := 0
	for i, r := range runes {
		var w int
		if r == '\t' {
			w = tabstop - (prefixWidth+width)%tabstop
		} else if r < 0x20 || r == 0x7f {
			w = 0
		} else if r >= 0x1100 && isWideRune(r) {
			w = 2
		} else {
			w = 1
		}
		width += w
		if width > limit {
			return width, i
		}
	}
	return width, -1
}

func isWideRune(r rune) bool {
	if r < 0x1100 {
		return false
	}
	if r <= 0x115f {
		return true
	}
	if r == 0x2329 || r == 0x232a {
		return true
	}
	if r >= 0x2e80 && r <= 0x303e {
		return true
	}
	if r >= 0x3040 && r <= 0x33bf {
		return true
	}
	if r >= 0x3400 && r <= 0x4dbf {
		return true
	}
	if r >= 0x4e00 && r <= 0xa4cf {
		return true
	}
	if r >= 0xa960 && r <= 0xa97c {
		return true
	}
	if r >= 0xac00 && r <= 0xd7a3 {
		return true
	}
	if r >= 0xf900 && r <= 0xfaff {
		return true
	}
	if r >= 0xfe10 && r <= 0xfe6f {
		return true
	}
	if r >= 0xff01 && r <= 0xff60 {
		return true
	}
	if r >= 0xffe0 && r <= 0xffe6 {
		return true
	}
	if r >= 0x1f000 && r <= 0x1fbff {
		return true
	}
	if r >= 0x20000 && r <= 0x2fffd {
		return true
	}
	if r >= 0x30000 && r <= 0x3fffd {
		return true
	}
	return false
}

func TruncateString(input string, limit int) ([]rune, int) {
	runes := []rune{}
	width := 0
	for _, r := range input {
		var w int
		if r >= 0x1100 && isWideRune(r) {
			w = 2
		} else {
			w = 1
		}
		if width+w > limit {
			return runes, width
		}
		width += w
		runes = append(runes, r)
	}
	return runes, width
}

func CompareVersions(v1, v2 string) int {
	parts1 := strings.Split(v1, ".")
	parts2 := strings.Split(v2, ".")
	parseNum := func(s string) int {
		n, err := strconv.Atoi(s)
		if err != nil {
			return 0
		}
		return n
	}
	length := max(len(parts1), len(parts2))
	for i := 0; i < length; i++ {
		var a, b int
		if i < len(parts1) {
			a = parseNum(parts1[i])
		}
		if i < len(parts2) {
			b = parseNum(parts2[i])
		}
		if a > b {
			return 1
		} else if a < b {
			return -1
		}
	}
	return 0
}

func RepeatToFill(str string, length int, limit int) string {
	times := limit / length
	rest := limit % length
	output := strings.Repeat(str, times)
	if rest > 0 {
		for _, r := range str {
			w := 1
			if r >= 0x1100 && isWideRune(r) {
				w = 2
			}
			rest -= w
			if rest < 0 {
				break
			}
			output += string(r)
			if rest == 0 {
				break
			}
		}
	}
	return output
}

func ToKebabCase(s string) string {
	result := ""
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result += "-"
		}
		result += string(r)
	}
	return strings.ToLower(result)
}

func StringWidth(s string) int {
	width := 0
	for _, r := range s {
		if r == '\n' || r == '\r' {
			width++
		} else if r >= 0x1100 && isWideRune(r) {
			width += 2
		} else if r >= 0x20 && r != 0x7f {
			width++
		}
	}
	return width
}

func CountRunes(s string) int {
	return utf8.RuneCountInString(s)
}
