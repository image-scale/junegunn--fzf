package charutil

import (
	"unicode"
	"unicode/utf8"
	"unsafe"
)

const (
	hibitMask64 uint64 = 0x8080808080808080
	hibitMask32 uint32 = 0x80808080
)

type Chars struct {
	data            []byte
	asciiMode       bool
	trimLenCached   bool
	cachedTrimLen   uint16
	Index           int32
}

func detectAscii(b []byte) (bool, int) {
	i := 0
	for ; i <= len(b)-8; i += 8 {
		if (hibitMask64 & *(*uint64)(unsafe.Pointer(&b[i]))) > 0 {
			return false, i
		}
	}
	for ; i <= len(b)-4; i += 4 {
		if (hibitMask32 & *(*uint32)(unsafe.Pointer(&b[i]))) > 0 {
			return false, i
		}
	}
	for ; i < len(b); i++ {
		if b[i] >= utf8.RuneSelf {
			return false, i
		}
	}
	return true, 0
}

func ToChars(b []byte) Chars {
	allAscii, firstNonAscii := detectAscii(b)
	if allAscii {
		return Chars{data: b, asciiMode: true}
	}
	runes := make([]rune, firstNonAscii, len(b))
	for i := 0; i < firstNonAscii; i++ {
		runes[i] = rune(b[i])
	}
	for i := firstNonAscii; i < len(b); {
		r, sz := utf8.DecodeRune(b[i:])
		i += sz
		runes = append(runes, r)
	}
	return FromRunes(runes)
}

func FromRunes(runes []rune) Chars {
	return Chars{data: *(*[]byte)(unsafe.Pointer(&runes)), asciiMode: false}
}

func (c *Chars) runeSlice() []rune {
	if c.asciiMode {
		return nil
	}
	return *(*[]rune)(unsafe.Pointer(&c.data))
}

func (c *Chars) IsBytes() bool {
	return c.asciiMode
}

func (c *Chars) Bytes() []byte {
	return c.data
}

func (c *Chars) Get(i int) rune {
	if rs := c.runeSlice(); rs != nil {
		return rs[i]
	}
	return rune(c.data[i])
}

func (c *Chars) Length() int {
	if rs := c.runeSlice(); rs != nil {
		return len(rs)
	}
	return len(c.data)
}

func (c *Chars) TrimLength() uint16 {
	if c.trimLenCached {
		return c.cachedTrimLen
	}
	c.trimLenCached = true
	n := c.Length()
	end := -1
	for i := n - 1; i >= 0; i-- {
		if !unicode.IsSpace(c.Get(i)) {
			end = i
			break
		}
	}
	if end < 0 {
		c.cachedTrimLen = 0
		return 0
	}
	start := 0
	for j := 0; j < n; j++ {
		if !unicode.IsSpace(c.Get(j)) {
			start = j
			break
		}
	}
	val := end - start + 1
	if val > 0xFFFF {
		c.cachedTrimLen = 0xFFFF
	} else if val < 0 {
		c.cachedTrimLen = 0
	} else {
		c.cachedTrimLen = uint16(val)
	}
	return c.cachedTrimLen
}

func (c *Chars) LeadingWhitespaces() int {
	count := 0
	for i := 0; i < c.Length(); i++ {
		if !unicode.IsSpace(c.Get(i)) {
			break
		}
		count++
	}
	return count
}

func (c *Chars) TrailingWhitespaces() int {
	count := 0
	for i := c.Length() - 1; i >= 0; i-- {
		if !unicode.IsSpace(c.Get(i)) {
			break
		}
		count++
	}
	return count
}

func (c *Chars) ToString() string {
	if rs := c.runeSlice(); rs != nil {
		return string(rs)
	}
	return unsafe.String(unsafe.SliceData(c.data), len(c.data))
}

func (c *Chars) ToRunes() []rune {
	if rs := c.runeSlice(); rs != nil {
		return rs
	}
	runes := make([]rune, len(c.data))
	for i, b := range c.data {
		runes[i] = rune(b)
	}
	return runes
}

func (c *Chars) CopyRunes(dest []rune, from int) {
	if rs := c.runeSlice(); rs != nil {
		copy(dest, rs[from:])
		return
	}
	for i, b := range c.data[from:][:len(dest)] {
		dest[i] = rune(b)
	}
}

func (c *Chars) Prepend(prefix string) {
	if rs := c.runeSlice(); rs != nil {
		rs = append([]rune(prefix), rs...)
		c.data = *(*[]byte)(unsafe.Pointer(&rs))
	} else {
		c.data = append([]byte(prefix), c.data...)
	}
}

func (c *Chars) SliceRight(last int) {
	c.data = c.data[:last]
}

func (c *Chars) TrimTrailingWhitespaces(maxIndex int) {
	ws := c.TrailingWhitespaces()
	end := len(c.data) - ws
	if maxIndex > end {
		end = maxIndex
	}
	c.data = c.data[:end]
}

func (c *Chars) TrimSuffix(runes []rune) {
	last := len(c.data)
	first := last - len(runes)
	if first < 0 {
		return
	}
	for i := first; i < last; i++ {
		if c.Get(i) != runes[i-first] {
			return
		}
	}
	c.data = c.data[:first]
}
