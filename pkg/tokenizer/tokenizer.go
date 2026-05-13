package tokenizer

import (
	"bytes"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"unsafe"

	"github.com/fzf/finder/pkg/charutil"
)

const rangeEllipsis = 0

type Range struct {
	Begin int
	End   int
}

func (r Range) IsFull() bool {
	return r.Begin == rangeEllipsis && r.End == rangeEllipsis
}

func CompareRanges(r1, r2 []Range) bool {
	if len(r1) != len(r2) {
		return false
	}
	for i := range r1 {
		if r1[i] != r2[i] {
			return false
		}
	}
	return true
}

func RangesToString(ranges []Range) string {
	strs := make([]string, 0, len(ranges))
	for _, r := range ranges {
		var s string
		if r.Begin == rangeEllipsis && r.End == rangeEllipsis {
			s = ".."
		} else if r.Begin == r.End {
			s = strconv.Itoa(r.Begin)
		} else {
			if r.Begin != rangeEllipsis {
				s += strconv.Itoa(r.Begin)
			}
			if r.Begin != -1 {
				s += ".."
				if r.End != rangeEllipsis {
					s += strconv.Itoa(r.End)
				}
			}
		}
		strs = append(strs, s)
	}
	return strings.Join(strs, ",")
}

type Token struct {
	Text         *charutil.Chars
	PrefixLength int32
}

func (t Token) String() string {
	return fmt.Sprintf("Token{text: %s, prefixLength: %d}", t.Text.ToString(), t.PrefixLength)
}

type Delimiter struct {
	Regex *regexp.Regexp
	Str   *string
}

func (d Delimiter) IsAwk() bool {
	return d.Regex == nil && d.Str == nil
}

func (d Delimiter) String() string {
	if d.Str != nil {
		return fmt.Sprintf("Delimiter{str: %q}", *d.Str)
	}
	return fmt.Sprintf("Delimiter{regex: %v}", d.Regex)
}

func newRange(begin, end int) Range {
	if begin == 1 && end != 1 {
		begin = rangeEllipsis
	}
	if end == -1 {
		end = rangeEllipsis
	}
	return Range{begin, end}
}

func ParseRange(str *string) (Range, bool) {
	if *str == ".." {
		return newRange(rangeEllipsis, rangeEllipsis), true
	}
	if strings.HasPrefix(*str, "..") {
		end, err := strconv.Atoi((*str)[2:])
		if err != nil || end == 0 {
			return Range{}, false
		}
		return newRange(rangeEllipsis, end), true
	}
	if strings.HasSuffix(*str, "..") {
		begin, err := strconv.Atoi((*str)[:len(*str)-2])
		if err != nil || begin == 0 {
			return Range{}, false
		}
		return newRange(begin, rangeEllipsis), true
	}
	if strings.Contains(*str, "..") {
		ns := strings.Split(*str, "..")
		if len(ns) != 2 {
			return Range{}, false
		}
		begin, err1 := strconv.Atoi(ns[0])
		end, err2 := strconv.Atoi(ns[1])
		if err1 != nil || err2 != nil || begin == 0 || end == 0 || begin < 0 && end > 0 {
			return Range{}, false
		}
		return newRange(begin, end), true
	}

	n, err := strconv.Atoi(*str)
	if err != nil || n == 0 {
		return Range{}, false
	}
	return newRange(n, n), true
}

func SplitNth(str string) ([]Range, error) {
	if match, _ := regexp.MatchString("^[0-9,-.]+$", str); !match {
		return nil, fmt.Errorf("invalid format: %s", str)
	}
	tokens := strings.Split(str, ",")
	ranges := make([]Range, len(tokens))
	for i, s := range tokens {
		r, ok := ParseRange(&s)
		if !ok {
			return nil, fmt.Errorf("invalid format: %s", str)
		}
		ranges[i] = r
	}
	return ranges, nil
}

func DelimiterFromString(str string) Delimiter {
	str = strings.ReplaceAll(str, "\\t", "\t")
	if len([]rune(str)) == 1 {
		return Delimiter{Str: &str}
	}
	if regexp.QuoteMeta(str) == str {
		return Delimiter{Str: &str}
	}
	rx, err := regexp.Compile(str)
	if err != nil {
		return Delimiter{Str: &str}
	}
	return Delimiter{Regex: rx}
}

func stringBytes(s string) []byte {
	return unsafe.Slice(unsafe.StringData(s), len(s))
}

func withPrefixLengths(tokens []string, begin int) []Token {
	ret := make([]Token, len(tokens))
	prefixLength := begin
	for i := range tokens {
		chars := charutil.ToChars(stringBytes(tokens[i]))
		ret[i] = Token{&chars, int32(prefixLength)}
		prefixLength += chars.Length()
	}
	return ret
}

const (
	awkNil = iota
	awkBlack
	awkWhite
)

func awkTokenizer(input string) ([]string, int) {
	var ret []string
	prefixLength := 0
	state := awkNil
	begin := 0
	end := 0
	for idx := 0; idx < len(input); idx++ {
		r := input[idx]
		white := r == 9 || r == 32 || r == 10
		switch state {
		case awkNil:
			if white {
				prefixLength++
			} else {
				state, begin, end = awkBlack, idx, idx+1
			}
		case awkBlack:
			end = idx + 1
			if white {
				state = awkWhite
			}
		case awkWhite:
			if white {
				end = idx + 1
			} else {
				ret = append(ret, input[begin:end])
				state, begin, end = awkBlack, idx, idx+1
			}
		}
	}
	if begin < end {
		ret = append(ret, input[begin:end])
	}
	return ret, prefixLength
}

func Tokenize(text string, delimiter Delimiter) []Token {
	if delimiter.Str == nil && delimiter.Regex == nil {
		tokens, prefixLength := awkTokenizer(text)
		return withPrefixLengths(tokens, prefixLength)
	}

	if delimiter.Str != nil {
		return withPrefixLengths(strings.SplitAfter(text, *delimiter.Str), 0)
	}

	var tokens []string
	if delimiter.Regex != nil {
		locs := delimiter.Regex.FindAllStringIndex(text, -1)
		begin := 0
		tokens = make([]string, len(locs))
		for i, loc := range locs {
			tokens[i] = text[begin:loc[1]]
			begin = loc[1]
		}
		if begin < len(text) {
			tokens = append(tokens, text[begin:])
		}
	}
	return withPrefixLengths(tokens, 0)
}

func StripLastDelimiter(str string, delimiter Delimiter) string {
	if delimiter.Str != nil {
		return strings.TrimSuffix(str, *delimiter.Str)
	}
	if delimiter.Regex != nil {
		locs := delimiter.Regex.FindAllStringIndex(str, -1)
		if len(locs) > 0 {
			last := locs[len(locs)-1]
			if last[1] == len(str) {
				str = str[:last[0]]
			}
		}
		return str
	}
	return strings.TrimRightFunc(str, unicode.IsSpace)
}

func GetLastDelimiter(str string, delimiter Delimiter) string {
	if delimiter.Str != nil {
		if strings.HasSuffix(str, *delimiter.Str) {
			return *delimiter.Str
		}
	} else if delimiter.Regex != nil {
		locs := delimiter.Regex.FindAllStringIndex(str, -1)
		if len(locs) > 0 {
			last := locs[len(locs)-1]
			if last[1] == len(str) {
				return str[last[0]:]
			}
		}
	}
	return ""
}

func JoinTokens(tokens []Token) string {
	var buf bytes.Buffer
	for _, t := range tokens {
		buf.WriteString(t.Text.ToString())
	}
	return buf.String()
}

func Transform(tokens []Token, withNth []Range) []Token {
	transTokens := make([]Token, len(withNth))
	numTokens := len(tokens)
	for idx, r := range withNth {
		var parts []*charutil.Chars
		minIdx := 0
		if r.Begin == r.End {
			i := r.Begin
			if i == rangeEllipsis {
				chars := charutil.ToChars(stringBytes(JoinTokens(tokens)))
				parts = append(parts, &chars)
			} else {
				if i < 0 {
					i += numTokens + 1
				}
				if i >= 1 && i <= numTokens {
					minIdx = i - 1
					parts = append(parts, tokens[i-1].Text)
				}
			}
		} else {
			var begin, end int
			if r.Begin == rangeEllipsis {
				begin, end = 1, r.End
				if end < 0 {
					end += numTokens + 1
				}
			} else if r.End == rangeEllipsis {
				begin, end = r.Begin, numTokens
				if begin < 0 {
					begin += numTokens + 1
				}
			} else {
				begin, end = r.Begin, r.End
				if begin < 0 {
					begin += numTokens + 1
				}
				if end < 0 {
					end += numTokens + 1
				}
			}
			minIdx = max(0, begin-1)
			for i := begin; i <= end; i++ {
				if i >= 1 && i <= numTokens {
					parts = append(parts, tokens[i-1].Text)
				}
			}
		}

		var merged charutil.Chars
		switch len(parts) {
		case 0:
			merged = charutil.ToChars([]byte{})
		case 1:
			merged = *parts[0]
		default:
			var buf bytes.Buffer
			for _, p := range parts {
				buf.WriteString(p.ToString())
			}
			merged = charutil.ToChars(buf.Bytes())
		}

		var prefixLength int32
		if minIdx < numTokens {
			prefixLength = tokens[minIdx].PrefixLength
		}
		transTokens[idx] = Token{&merged, prefixLength}
	}
	return transTokens
}
