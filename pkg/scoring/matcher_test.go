package scoring

import (
	"math"
	"sort"
	"strings"
	"testing"

	"github.com/fzf/finder/pkg/charutil"
)

func init() {
	Setup("default")
}

func checkMatch(t *testing.T, fn MatchFunc, caseSens, fwd bool, input, pattern string, wantStart, wantEnd, wantScore int) {
	t.Helper()
	checkMatchNormalize(t, fn, caseSens, false, fwd, input, pattern, wantStart, wantEnd, wantScore)
}

func checkMatchNormalize(t *testing.T, fn MatchFunc, caseSens, norm, fwd bool, input, pattern string, wantStart, wantEnd, wantScore int) {
	t.Helper()
	if !caseSens {
		pattern = strings.ToLower(pattern)
	}
	chars := charutil.ToChars([]byte(input))
	res, pos := fn(caseSens, norm, fwd, &chars, []rune(pattern), true, nil)
	var start, end int
	if pos == nil || len(*pos) == 0 {
		start = res.Start
		end = res.End
	} else {
		sort.Ints(*pos)
		start = (*pos)[0]
		end = (*pos)[len(*pos)-1] + 1
	}
	if start != wantStart {
		t.Errorf("Start: got %d, want %d (%s / %s)", start, wantStart, input, pattern)
	}
	if end != wantEnd {
		t.Errorf("End: got %d, want %d (%s / %s)", end, wantEnd, input, pattern)
	}
	if res.Score != wantScore {
		t.Errorf("Score: got %d, want %d (%s / %s)", res.Score, wantScore, input, pattern)
	}
}

func TestFuzzyMatching(t *testing.T) {
	for _, fn := range []MatchFunc{FuzzyMatchV1, FuzzyMatchV2} {
		for _, fwd := range []bool{true, false} {
			checkMatch(t, fn, false, fwd, "fooBarbaz1", "oBZ", 2, 9,
				ScoreMatch*3+BonusCamelCase+ScoreGapStart+ScoreGapExtension*3)
			checkMatch(t, fn, false, fwd, "foo bar baz", "fbb", 0, 9,
				ScoreMatch*3+int(BonusBoundaryWhite)*BonusFirstCharMultiplier+
					int(BonusBoundaryWhite)*2+2*ScoreGapStart+4*ScoreGapExtension)
			checkMatch(t, fn, false, fwd, "/AutomatorDocument.icns", "rdoc", 9, 13,
				ScoreMatch*4+BonusCamelCase+BonusConsecutive*2)
			checkMatch(t, fn, false, fwd, "/man1/zshcompctl.1", "zshc", 6, 10,
				ScoreMatch*4+int(BonusBoundaryDelimiter)*BonusFirstCharMultiplier+int(BonusBoundaryDelimiter)*3)
			checkMatch(t, fn, false, fwd, "/.oh-my-zsh/cache", "zshc", 8, 13,
				ScoreMatch*4+BonusBoundary*BonusFirstCharMultiplier+BonusBoundary*2+ScoreGapStart+int(BonusBoundaryDelimiter))
			checkMatch(t, fn, false, fwd, "ab0123 456", "12356", 3, 10,
				ScoreMatch*5+BonusConsecutive*3+ScoreGapStart+ScoreGapExtension)
			checkMatch(t, fn, false, fwd, "abc123 456", "12356", 3, 10,
				ScoreMatch*5+BonusCamelCase*BonusFirstCharMultiplier+BonusCamelCase*2+BonusConsecutive+ScoreGapStart+ScoreGapExtension)
			checkMatch(t, fn, false, fwd, "foo/bar/baz", "fbb", 0, 9,
				ScoreMatch*3+int(BonusBoundaryWhite)*BonusFirstCharMultiplier+
					int(BonusBoundaryDelimiter)*2+2*ScoreGapStart+4*ScoreGapExtension)
			checkMatch(t, fn, false, fwd, "fooBarBaz", "fbb", 0, 7,
				ScoreMatch*3+int(BonusBoundaryWhite)*BonusFirstCharMultiplier+
					BonusCamelCase*2+2*ScoreGapStart+2*ScoreGapExtension)
			checkMatch(t, fn, false, fwd, "foo barbaz", "fbb", 0, 8,
				ScoreMatch*3+int(BonusBoundaryWhite)*BonusFirstCharMultiplier+int(BonusBoundaryWhite)+
					ScoreGapStart*2+ScoreGapExtension*3)
			checkMatch(t, fn, false, fwd, "fooBar Baz", "foob", 0, 4,
				ScoreMatch*4+int(BonusBoundaryWhite)*BonusFirstCharMultiplier+int(BonusBoundaryWhite)*3)
			checkMatch(t, fn, false, fwd, "xFoo-Bar Baz", "foo-b", 1, 6,
				ScoreMatch*5+BonusCamelCase*BonusFirstCharMultiplier+BonusCamelCase*2+
					BonusNonWord+BonusBoundary)

			checkMatch(t, fn, true, fwd, "fooBarbaz", "oBz", 2, 9,
				ScoreMatch*3+BonusCamelCase+ScoreGapStart+ScoreGapExtension*3)
			checkMatch(t, fn, true, fwd, "Foo/Bar/Baz", "FBB", 0, 9,
				ScoreMatch*3+int(BonusBoundaryWhite)*BonusFirstCharMultiplier+int(BonusBoundaryDelimiter)*2+
					ScoreGapStart*2+ScoreGapExtension*4)
			checkMatch(t, fn, true, fwd, "FooBarBaz", "FBB", 0, 7,
				ScoreMatch*3+int(BonusBoundaryWhite)*BonusFirstCharMultiplier+BonusCamelCase*2+
					ScoreGapStart*2+ScoreGapExtension*2)
			checkMatch(t, fn, true, fwd, "FooBar Baz", "FooB", 0, 4,
				ScoreMatch*4+int(BonusBoundaryWhite)*BonusFirstCharMultiplier+int(BonusBoundaryWhite)*2+
					max(BonusCamelCase, int(BonusBoundaryWhite)))
			checkMatch(t, fn, true, fwd, "foo-bar", "o-ba", 2, 6,
				ScoreMatch*4+BonusBoundary*3)

			// Non-matches
			checkMatch(t, fn, true, fwd, "fooBarbaz", "oBZ", -1, -1, 0)
			checkMatch(t, fn, true, fwd, "Foo Bar Baz", "fbb", -1, -1, 0)
			checkMatch(t, fn, true, fwd, "fooBarbaz", "fooBarbazz", -1, -1, 0)
		}
	}
}

func TestFuzzyMatchDirection(t *testing.T) {
	checkMatch(t, FuzzyMatchV1, false, true, "foobar fb", "fb", 0, 4,
		ScoreMatch*2+int(BonusBoundaryWhite)*BonusFirstCharMultiplier+
			ScoreGapStart+ScoreGapExtension)
	checkMatch(t, FuzzyMatchV1, false, false, "foobar fb", "fb", 7, 9,
		ScoreMatch*2+int(BonusBoundaryWhite)*BonusFirstCharMultiplier+int(BonusBoundaryWhite))
}

func TestExactMatching(t *testing.T) {
	for _, dir := range []bool{true, false} {
		checkMatch(t, ExactMatchNaive, true, dir, "fooBarbaz", "oBA", -1, -1, 0)
		checkMatch(t, ExactMatchNaive, true, dir, "fooBarbaz", "fooBarbazz", -1, -1, 0)

		checkMatch(t, ExactMatchNaive, false, dir, "fooBarbaz", "oBA", 2, 5,
			ScoreMatch*3+BonusCamelCase+BonusConsecutive)
		checkMatch(t, ExactMatchNaive, false, dir, "/AutomatorDocument.icns", "rdoc", 9, 13,
			ScoreMatch*4+BonusCamelCase+BonusConsecutive*2)
		checkMatch(t, ExactMatchNaive, false, dir, "/man1/zshcompctl.1", "zshc", 6, 10,
			ScoreMatch*4+int(BonusBoundaryDelimiter)*(BonusFirstCharMultiplier+3))
		checkMatch(t, ExactMatchNaive, false, dir, "/.oh-my-zsh/cache", "zsh/c", 8, 13,
			ScoreMatch*5+BonusBoundary*(BonusFirstCharMultiplier+3)+int(BonusBoundaryDelimiter))
	}
}

func TestExactMatchDirection(t *testing.T) {
	checkMatch(t, ExactMatchNaive, false, true, "foobar foob", "oo", 1, 3,
		ScoreMatch*2+BonusConsecutive)
	checkMatch(t, ExactMatchNaive, false, false, "foobar foob", "oo", 8, 10,
		ScoreMatch*2+BonusConsecutive)
}

func TestPrefixMatching(t *testing.T) {
	score := ScoreMatch*3 + int(BonusBoundaryWhite)*BonusFirstCharMultiplier + int(BonusBoundaryWhite)*2
	for _, dir := range []bool{true, false} {
		checkMatch(t, PrefixMatch, true, dir, "fooBarbaz", "Foo", -1, -1, 0)
		checkMatch(t, PrefixMatch, false, dir, "fooBarBaz", "baz", -1, -1, 0)
		checkMatch(t, PrefixMatch, false, dir, "fooBarbaz", "Foo", 0, 3, score)
		checkMatch(t, PrefixMatch, false, dir, "foOBarBaZ", "foo", 0, 3, score)
		checkMatch(t, PrefixMatch, false, dir, "f-oBarbaz", "f-o", 0, 3, score)

		checkMatch(t, PrefixMatch, false, dir, " fooBar", "foo", 1, 4, score)
		checkMatch(t, PrefixMatch, false, dir, " fooBar", " fo", 0, 3, score)
		checkMatch(t, PrefixMatch, false, dir, "     fo", "foo", -1, -1, 0)
	}
}

func TestSuffixMatching(t *testing.T) {
	for _, dir := range []bool{true, false} {
		checkMatch(t, SuffixMatch, true, dir, "fooBarbaz", "Baz", -1, -1, 0)
		checkMatch(t, SuffixMatch, false, dir, "fooBarbaz", "Foo", -1, -1, 0)

		checkMatch(t, SuffixMatch, false, dir, "fooBarbaz", "baz", 6, 9,
			ScoreMatch*3+BonusConsecutive*2)
		checkMatch(t, SuffixMatch, false, dir, "fooBarBaZ", "baz", 6, 9,
			(ScoreMatch+BonusCamelCase)*3+BonusCamelCase*(BonusFirstCharMultiplier-1))

		checkMatch(t, SuffixMatch, false, dir, "fooBarbaz ", "baz", 6, 9,
			ScoreMatch*3+BonusConsecutive*2)
		checkMatch(t, SuffixMatch, false, dir, "fooBarbaz ", "baz ", 6, 10,
			ScoreMatch*4+BonusConsecutive*2+int(BonusBoundaryWhite))
	}
}

func TestEmptyPatterns(t *testing.T) {
	for _, dir := range []bool{true, false} {
		checkMatch(t, FuzzyMatchV1, true, dir, "foobar", "", 0, 0, 0)
		checkMatch(t, FuzzyMatchV2, true, dir, "foobar", "", 0, 0, 0)
		checkMatch(t, ExactMatchNaive, true, dir, "foobar", "", 0, 0, 0)
		checkMatch(t, PrefixMatch, true, dir, "foobar", "", 0, 0, 0)
		checkMatch(t, SuffixMatch, true, dir, "foobar", "", 6, 6, 0)
	}
}

func TestNormalization(t *testing.T) {
	test := func(input, pattern string, sidx, eidx, score int, fns ...MatchFunc) {
		for _, fn := range fns {
			checkMatchNormalize(t, fn, false, true, true, input, pattern, sidx, eidx, score)
		}
	}
	test("Só Danço Samba", "So", 0, 2, 62, FuzzyMatchV1, FuzzyMatchV2, PrefixMatch, ExactMatchNaive)
	test("Só Danço Samba", "sodc", 0, 7, 97, FuzzyMatchV1, FuzzyMatchV2)
	test("Danço", "danco", 0, 5, 140, FuzzyMatchV1, FuzzyMatchV2, PrefixMatch, SuffixMatch, ExactMatchNaive, EqualMatch)
}

func TestLongText(t *testing.T) {
	data := make([]byte, math.MaxUint16*2)
	for i := range data {
		data[i] = 'x'
	}
	data[math.MaxUint16] = 'z'
	checkMatch(t, FuzzyMatchV2, true, true, string(data), "zx", math.MaxUint16, math.MaxUint16+2,
		ScoreMatch*2+BonusConsecutive)
}

func TestNormalizeRunes(t *testing.T) {
	runes := []rune("Só Danço")
	normalized := NormalizeRunes(runes)
	expected := []rune("So Danco")
	if string(normalized) != string(expected) {
		t.Errorf("NormalizeRunes: got %q, want %q", string(normalized), string(expected))
	}
}

func TestScoringSchemeSetup(t *testing.T) {
	if !Setup("default") {
		t.Error("Setup('default') should return true")
	}
	if !Setup("path") {
		t.Error("Setup('path') should return true")
	}
	if !Setup("history") {
		t.Error("Setup('history') should return true")
	}
	if Setup("invalid") {
		t.Error("Setup('invalid') should return false")
	}
	Setup("default")
}

func TestEqualMatching(t *testing.T) {
	// Empty pattern
	chars := charutil.ToChars([]byte("foobar"))
	res, _ := EqualMatch(true, false, true, &chars, []rune{}, false, nil)
	if res.Start != -1 || res.End != -1 {
		t.Errorf("Empty equal match should return -1, -1")
	}

	// Exact equality with whitespace trimming
	chars2 := charutil.ToChars([]byte("  AbC  "))
	res2, _ := EqualMatch(false, false, true, &chars2, []rune("abc"), false, nil)
	if res2.Start != 2 || res2.End != 5 {
		t.Errorf("EqualMatch: got [%d,%d], want [2,5]", res2.Start, res2.End)
	}

	// Case sensitive non-match
	chars3 := charutil.ToChars([]byte("AbC"))
	res3, _ := EqualMatch(true, false, true, &chars3, []rune("abc"), false, nil)
	if res3.Start != -1 {
		t.Errorf("Case-sensitive EqualMatch should not match")
	}
}

func TestLongTextWithNormalize(t *testing.T) {
	data := make([]byte, 30000)
	for i := range data {
		data[i] = 'x'
	}
	unicodeString := string(data) + " Minímal example"
	checkMatchNormalize(t, FuzzyMatchV1, false, true, false, unicodeString, "minim", 30001, 30006, 140)
}
