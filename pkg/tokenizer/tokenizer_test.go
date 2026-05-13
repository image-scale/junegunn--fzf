package tokenizer

import (
	"testing"
)

func TestParseRange(t *testing.T) {
	{
		i := ".."
		r, ok := ParseRange(&i)
		if !ok || r.Begin != rangeEllipsis || r.End != rangeEllipsis {
			t.Errorf("full range: %v", r)
		}
	}
	{
		i := "3.."
		r, ok := ParseRange(&i)
		if !ok || r.Begin != 3 || r.End != rangeEllipsis {
			t.Errorf("begin-open: %v", r)
		}
	}
	{
		i := "..5"
		r, ok := ParseRange(&i)
		if !ok || r.Begin != rangeEllipsis || r.End != 5 {
			t.Errorf("end-open: %v", r)
		}
	}
	{
		i := "3..5"
		r, ok := ParseRange(&i)
		if !ok || r.Begin != 3 || r.End != 5 {
			t.Errorf("closed range: %v", r)
		}
	}
	{
		i := "-3..-5"
		r, ok := ParseRange(&i)
		if !ok || r.Begin != -3 || r.End != -5 {
			t.Errorf("negative range: %v", r)
		}
	}
	{
		i := "3"
		r, ok := ParseRange(&i)
		if !ok || r.Begin != 3 || r.End != 3 {
			t.Errorf("single field: %v", r)
		}
	}
	{
		// begin=1 normalizes to ellipsis when end != 1
		i := "1..3"
		r, ok := ParseRange(&i)
		if !ok || r.Begin != rangeEllipsis || r.End != 3 {
			t.Errorf("begin=1 normalization: %v", r)
		}
	}
	{
		// end=-1 normalizes to ellipsis
		i := "3..-1"
		r, ok := ParseRange(&i)
		if !ok || r.Begin != 3 || r.End != rangeEllipsis {
			t.Errorf("end=-1 normalization: %v", r)
		}
	}
	{
		// single field 1: newRange(1,1) keeps begin=1 because end==1
		i := "1"
		r, ok := ParseRange(&i)
		if !ok || r.Begin != 1 || r.End != 1 {
			t.Errorf("single 1: %v", r)
		}
	}
	{
		i := "1..3..5"
		if _, ok := ParseRange(&i); ok {
			t.Error("should reject too many dots")
		}
	}
	{
		i := "-3..3"
		if _, ok := ParseRange(&i); ok {
			t.Error("should reject mixed negative/positive")
		}
	}
	{
		i := "0"
		if _, ok := ParseRange(&i); ok {
			t.Error("should reject zero")
		}
	}
	{
		i := "abc"
		if _, ok := ParseRange(&i); ok {
			t.Error("should reject non-numeric")
		}
	}
}

func TestRangeIsFull(t *testing.T) {
	r := Range{Begin: rangeEllipsis, End: rangeEllipsis}
	if !r.IsFull() {
		t.Error("expected full range")
	}
	r2 := Range{Begin: 1, End: 3}
	if r2.IsFull() {
		t.Error("expected not full")
	}
}

func TestCompareRanges(t *testing.T) {
	r1 := []Range{{1, 3}, {4, 5}}
	r2 := []Range{{1, 3}, {4, 5}}
	if !CompareRanges(r1, r2) {
		t.Error("equal ranges should match")
	}
	r3 := []Range{{1, 3}}
	if CompareRanges(r1, r3) {
		t.Error("different length should not match")
	}
	r4 := []Range{{1, 3}, {4, 6}}
	if CompareRanges(r1, r4) {
		t.Error("different values should not match")
	}
}

func TestRangesToString(t *testing.T) {
	ranges := []Range{
		{0, 0},
		{3, 3},
		{3, 5},
		{0, 5},
		{3, 0},
	}
	result := RangesToString(ranges)
	expected := "..,3,3..5,..5,3.."
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestSplitNth(t *testing.T) {
	ranges, err := SplitNth("1,2,3")
	if err != nil {
		t.Fatal(err)
	}
	if len(ranges) != 3 {
		t.Fatalf("expected 3 ranges, got %d", len(ranges))
	}

	ranges2, err := SplitNth("1..2,3,2..")
	if err != nil {
		t.Fatal(err)
	}
	if len(ranges2) != 3 {
		t.Fatalf("expected 3 ranges, got %d", len(ranges2))
	}

	_, err = SplitNth("abc")
	if err == nil {
		t.Error("should reject invalid format")
	}
}

func TestTokenizeAwk(t *testing.T) {
	input := "  abc: \n\t def:  ghi  "
	tokens := Tokenize(input, Delimiter{})
	if len(tokens) < 1 {
		t.Fatal("expected at least 1 token")
	}
	if tokens[0].Text.ToString() != "abc: \n\t " || tokens[0].PrefixLength != 2 {
		t.Errorf("AWK token 0: text=%q prefix=%d", tokens[0].Text.ToString(), tokens[0].PrefixLength)
	}
}

func TestTokenizeStringDelimiter(t *testing.T) {
	input := "  abc: \n\t def:  ghi  "
	delim := DelimiterFromString(":")
	tokens := Tokenize(input, delim)
	if tokens[0].Text.ToString() != "  abc:" || tokens[0].PrefixLength != 0 {
		t.Errorf("string delim token 0: text=%q prefix=%d", tokens[0].Text.ToString(), tokens[0].PrefixLength)
	}
}

func TestTokenizeRegexDelimiter(t *testing.T) {
	input := "  abc: \n\t def:  ghi  "
	delim := DelimiterFromString("\\s+")
	tokens := Tokenize(input, delim)
	if len(tokens) < 4 {
		t.Fatalf("expected 4 tokens, got %d", len(tokens))
	}
	if tokens[0].Text.ToString() != "  " || tokens[0].PrefixLength != 0 {
		t.Errorf("token 0: text=%q prefix=%d", tokens[0].Text.ToString(), tokens[0].PrefixLength)
	}
	if tokens[1].Text.ToString() != "abc: \n\t " || tokens[1].PrefixLength != 2 {
		t.Errorf("token 1: text=%q prefix=%d", tokens[1].Text.ToString(), tokens[1].PrefixLength)
	}
	if tokens[2].Text.ToString() != "def:  " || tokens[2].PrefixLength != 10 {
		t.Errorf("token 2: text=%q prefix=%d", tokens[2].Text.ToString(), tokens[2].PrefixLength)
	}
	if tokens[3].Text.ToString() != "ghi  " || tokens[3].PrefixLength != 16 {
		t.Errorf("token 3: text=%q prefix=%d", tokens[3].Text.ToString(), tokens[3].PrefixLength)
	}
}

func TestTransformAwk(t *testing.T) {
	input := "  abc:  def:  ghi:  jkl"
	tokens := Tokenize(input, Delimiter{})

	{
		ranges, _ := SplitNth("1,2,3")
		tx := Transform(tokens, ranges)
		if JoinTokens(tx) != "abc:  def:  ghi:  " {
			t.Errorf("1,2,3: %q", JoinTokens(tx))
		}
	}
	{
		ranges, _ := SplitNth("1..2,3,2..,1")
		tx := Transform(tokens, ranges)
		joined := JoinTokens(tx)
		if joined != "abc:  def:  ghi:  def:  ghi:  jklabc:  " {
			t.Errorf("complex transform: %q", joined)
		}
		if len(tx) != 4 {
			t.Fatalf("expected 4 tokens, got %d", len(tx))
		}
		if tx[0].Text.ToString() != "abc:  def:  " || tx[0].PrefixLength != 2 {
			t.Errorf("tx[0]: %q prefix=%d", tx[0].Text.ToString(), tx[0].PrefixLength)
		}
		if tx[1].Text.ToString() != "ghi:  " || tx[1].PrefixLength != 14 {
			t.Errorf("tx[1]: %q prefix=%d", tx[1].Text.ToString(), tx[1].PrefixLength)
		}
		if tx[2].Text.ToString() != "def:  ghi:  jkl" || tx[2].PrefixLength != 8 {
			t.Errorf("tx[2]: %q prefix=%d", tx[2].Text.ToString(), tx[2].PrefixLength)
		}
		if tx[3].Text.ToString() != "abc:  " || tx[3].PrefixLength != 2 {
			t.Errorf("tx[3]: %q prefix=%d", tx[3].Text.ToString(), tx[3].PrefixLength)
		}
	}
}

func TestTransformStringDelimiter(t *testing.T) {
	input := "  abc:  def:  ghi:  jkl"
	delim := DelimiterFromString(":")
	tokens := Tokenize(input, delim)

	ranges, _ := SplitNth("1..2,3,2..,1")
	tx := Transform(tokens, ranges)
	joined := JoinTokens(tx)
	if joined != "  abc:  def:  ghi:  def:  ghi:  jkl  abc:" {
		t.Errorf("string delim transform: %q", joined)
	}
	if len(tx) != 4 {
		t.Fatalf("expected 4 tokens, got %d", len(tx))
	}
	if tx[0].Text.ToString() != "  abc:  def:" || tx[0].PrefixLength != 0 {
		t.Errorf("tx[0]: %q prefix=%d", tx[0].Text.ToString(), tx[0].PrefixLength)
	}
	if tx[1].Text.ToString() != "  ghi:" || tx[1].PrefixLength != 12 {
		t.Errorf("tx[1]: %q prefix=%d", tx[1].Text.ToString(), tx[1].PrefixLength)
	}
	if tx[2].Text.ToString() != "  def:  ghi:  jkl" || tx[2].PrefixLength != 6 {
		t.Errorf("tx[2]: %q prefix=%d", tx[2].Text.ToString(), tx[2].PrefixLength)
	}
	if tx[3].Text.ToString() != "  abc:" || tx[3].PrefixLength != 0 {
		t.Errorf("tx[3]: %q prefix=%d", tx[3].Text.ToString(), tx[3].PrefixLength)
	}
}

func TestTransformIndexOutOfBounds(t *testing.T) {
	s, _ := SplitNth("1")
	Transform([]Token{}, s)
}

func TestDelimiterIsAwk(t *testing.T) {
	d := Delimiter{}
	if !d.IsAwk() {
		t.Error("empty delimiter should be AWK")
	}
	d2 := DelimiterFromString(":")
	if d2.IsAwk() {
		t.Error("colon delimiter should not be AWK")
	}
}

func TestStripLastDelimiterAwk(t *testing.T) {
	result := StripLastDelimiter("hello   ", Delimiter{})
	if result != "hello" {
		t.Errorf("expected %q, got %q", "hello", result)
	}
}

func TestStripLastDelimiterString(t *testing.T) {
	delim := DelimiterFromString(":")
	result := StripLastDelimiter("abc:def:", delim)
	if result != "abc:def" {
		t.Errorf("expected %q, got %q", "abc:def", result)
	}
}

func TestStripLastDelimiterRegex(t *testing.T) {
	delim := DelimiterFromString("\\s+")
	result := StripLastDelimiter("abc def  ", delim)
	if result != "abc def" {
		t.Errorf("expected %q, got %q", "abc def", result)
	}
}

func TestGetLastDelimiter(t *testing.T) {
	delim := DelimiterFromString(":")
	if got := GetLastDelimiter("abc:def:", delim); got != ":" {
		t.Errorf("expected %q, got %q", ":", got)
	}
	if got := GetLastDelimiter("abc:def", delim); got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

func TestJoinTokens(t *testing.T) {
	input := "abc def ghi"
	tokens := Tokenize(input, Delimiter{})
	joined := JoinTokens(tokens)
	if joined != input {
		t.Errorf("expected %q, got %q", input, joined)
	}
}
