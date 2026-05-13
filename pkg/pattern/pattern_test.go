package pattern

import (
	"reflect"
	"runtime"
	"testing"

	"github.com/fzf/finder/pkg/charutil"
	"github.com/fzf/finder/pkg/chunk"
	"github.com/fzf/finder/pkg/result"
	"github.com/fzf/finder/pkg/scoring"
	"github.com/fzf/finder/pkg/tokenizer"
)

var slab *charutil.Slab

func init() {
	slab = charutil.NewSlab(100*1024, 2048)
}

func buildTestPattern(fuzzy bool, fuzzyAlgo scoring.MatchFunc, extended bool, caseMode CaseMode, normalize bool, forward bool,
	withPos bool, cacheable bool, nth []tokenizer.Range, delimiter tokenizer.Delimiter, runes []rune) *Pattern {
	return BuildPattern(chunk.NewChunkCache(), make(map[string]*Pattern),
		fuzzy, fuzzyAlgo, extended, caseMode, normalize, forward,
		withPos, cacheable, nth, delimiter, Revision{}, runes, nil, 0)
}

func TestParseTermsExtended(t *testing.T) {
	terms := ParseTerms(true, CaseSmart, false,
		"aaa 'bbb ^ccc ddd$ !eee !'fff !^ggg !hhh$ | ^iii$ ^xxx | 'yyy | zzz$ | !ZZZ |")
	if len(terms) != 9 ||
		terms[0][0].Typ != TermFuzzy || terms[0][0].Inv ||
		terms[1][0].Typ != TermExact || terms[1][0].Inv ||
		terms[2][0].Typ != TermPrefix || terms[2][0].Inv ||
		terms[3][0].Typ != TermSuffix || terms[3][0].Inv ||
		terms[4][0].Typ != TermExact || !terms[4][0].Inv ||
		terms[5][0].Typ != TermFuzzy || !terms[5][0].Inv ||
		terms[6][0].Typ != TermPrefix || !terms[6][0].Inv ||
		terms[7][0].Typ != TermSuffix || !terms[7][0].Inv ||
		terms[7][1].Typ != TermEqual || terms[7][1].Inv ||
		terms[8][0].Typ != TermPrefix || terms[8][0].Inv ||
		terms[8][1].Typ != TermExact || terms[8][1].Inv ||
		terms[8][2].Typ != TermSuffix || terms[8][2].Inv ||
		terms[8][3].Typ != TermExact || !terms[8][3].Inv {
		t.Errorf("%v", terms)
	}
	for _, termSet := range terms[:8] {
		term := termSet[0]
		if len(term.Text) != 3 {
			t.Errorf("%v", term)
		}
	}
}

func TestParseTermsExtendedExact(t *testing.T) {
	terms := ParseTerms(false, CaseSmart, false,
		"aaa 'bbb ^ccc ddd$ !eee !'fff !^ggg !hhh$")
	if len(terms) != 8 ||
		terms[0][0].Typ != TermExact || terms[0][0].Inv || len(terms[0][0].Text) != 3 ||
		terms[1][0].Typ != TermFuzzy || terms[1][0].Inv || len(terms[1][0].Text) != 3 ||
		terms[2][0].Typ != TermPrefix || terms[2][0].Inv || len(terms[2][0].Text) != 3 ||
		terms[3][0].Typ != TermSuffix || terms[3][0].Inv || len(terms[3][0].Text) != 3 ||
		terms[4][0].Typ != TermExact || !terms[4][0].Inv || len(terms[4][0].Text) != 3 ||
		terms[5][0].Typ != TermFuzzy || !terms[5][0].Inv || len(terms[5][0].Text) != 3 ||
		terms[6][0].Typ != TermPrefix || !terms[6][0].Inv || len(terms[6][0].Text) != 3 ||
		terms[7][0].Typ != TermSuffix || !terms[7][0].Inv || len(terms[7][0].Text) != 3 {
		t.Errorf("%v", terms)
	}
}

func TestParseTermsEmpty(t *testing.T) {
	terms := ParseTerms(true, CaseSmart, false, "' ^ !' !^")
	if len(terms) != 0 {
		t.Errorf("%v", terms)
	}
}

func TestExact(t *testing.T) {
	pat := buildTestPattern(true, scoring.FuzzyMatchV2, true, CaseSmart, false, true, false, true,
		[]tokenizer.Range{}, tokenizer.Delimiter{}, []rune("'abc"))
	chars := charutil.ToChars([]byte("aabbcc abc"))
	res, pos := scoring.ExactMatchNaive(
		pat.CaseSensitive, pat.Normalize, pat.Forward, &chars, pat.TermSets[0][0].Text, true, nil)
	if res.Start != 7 || res.End != 10 {
		t.Errorf("%v / %d / %d", pat.TermSets, res.Start, res.End)
	}
	if pos != nil {
		t.Errorf("pos is expected to be nil")
	}
}

func TestEqual(t *testing.T) {
	pat := buildTestPattern(true, scoring.FuzzyMatchV2, true, CaseSmart, false, true, false, true, []tokenizer.Range{}, tokenizer.Delimiter{}, []rune("^AbC$"))

	match := func(str string, sidxExpected int, eidxExpected int) {
		chars := charutil.ToChars([]byte(str))
		res, pos := scoring.EqualMatch(
			pat.CaseSensitive, pat.Normalize, pat.Forward, &chars, pat.TermSets[0][0].Text, true, nil)
		if res.Start != sidxExpected || res.End != eidxExpected {
			t.Errorf("%v / %d / %d", pat.TermSets, res.Start, res.End)
		}
		if pos != nil {
			t.Errorf("pos is expected to be nil")
		}
	}
	match("ABC", -1, -1)
	match("AbC", 0, 3)
	match("AbC  ", 0, 3)
	match(" AbC ", 1, 4)
	match("  AbC", 2, 5)
}

func TestCaseSensitivity(t *testing.T) {
	pat1 := buildTestPattern(true, scoring.FuzzyMatchV2, false, CaseSmart, false, true, false, true, []tokenizer.Range{}, tokenizer.Delimiter{}, []rune("abc"))
	pat2 := buildTestPattern(true, scoring.FuzzyMatchV2, false, CaseSmart, false, true, false, true, []tokenizer.Range{}, tokenizer.Delimiter{}, []rune("Abc"))
	pat3 := buildTestPattern(true, scoring.FuzzyMatchV2, false, CaseIgnore, false, true, false, true, []tokenizer.Range{}, tokenizer.Delimiter{}, []rune("abc"))
	pat4 := buildTestPattern(true, scoring.FuzzyMatchV2, false, CaseIgnore, false, true, false, true, []tokenizer.Range{}, tokenizer.Delimiter{}, []rune("Abc"))
	pat5 := buildTestPattern(true, scoring.FuzzyMatchV2, false, CaseRespect, false, true, false, true, []tokenizer.Range{}, tokenizer.Delimiter{}, []rune("abc"))
	pat6 := buildTestPattern(true, scoring.FuzzyMatchV2, false, CaseRespect, false, true, false, true, []tokenizer.Range{}, tokenizer.Delimiter{}, []rune("Abc"))

	if string(pat1.Text) != "abc" || pat1.CaseSensitive != false ||
		string(pat2.Text) != "Abc" || pat2.CaseSensitive != true ||
		string(pat3.Text) != "abc" || pat3.CaseSensitive != false ||
		string(pat4.Text) != "abc" || pat4.CaseSensitive != false ||
		string(pat5.Text) != "abc" || pat5.CaseSensitive != true ||
		string(pat6.Text) != "Abc" || pat6.CaseSensitive != true {
		t.Error("Invalid case conversion")
	}
}

func TestOrigTextAndTransformed(t *testing.T) {
	pat := buildTestPattern(true, scoring.FuzzyMatchV2, true, CaseSmart, false, true, false, true, []tokenizer.Range{}, tokenizer.Delimiter{}, []rune("jg"))
	tokens := tokenizer.Tokenize("junegunn", tokenizer.Delimiter{})
	trans := tokenizer.Transform(tokens, []tokenizer.Range{{Begin: 1, End: 1}})

	origBytes := []byte("junegunn.choi")
	for _, extended := range []bool{false, true} {
		c := chunk.Chunk{Count: 1}
		c.Items[0] = chunk.Item{
			Text:        charutil.ToChars([]byte("junegunn")),
			OrigText:    &origBytes,
			Transformed: &Transformed{pat.Rev, trans}}
		pat.Extended = extended
		matches, _ := pat.MatchChunk(&c, nil, slab)
		if !(matches[0].Item.Text.ToString() == "junegunn" &&
			string(*matches[0].Item.OrigText) == "junegunn.choi" &&
			reflect.DeepEqual(matches[0].Item.Transformed.(*Transformed).Tokens, trans)) {
			t.Error("Invalid match result", matches)
		}

		match, offsets, pos := pat.MatchItem(&c.Items[0], true, slab)
		if !(match.Item.Text.ToString() == "junegunn" &&
			string(*match.Item.OrigText) == "junegunn.choi" &&
			offsets[0][0] == 0 && offsets[0][1] == 5 &&
			reflect.DeepEqual(match.Item.Transformed.(*Transformed).Tokens, trans)) {
			t.Error("Invalid match result", match, offsets, extended)
		}
		if !((*pos)[0] == 4 && (*pos)[1] == 0) {
			t.Error("Invalid pos array", *pos)
		}
	}
}

func TestCacheKey(t *testing.T) {
	test := func(extended bool, patStr string, expected string, cacheable bool) {
		pat := buildTestPattern(true, scoring.FuzzyMatchV2, extended, CaseSmart, false, true, false, true, []tokenizer.Range{}, tokenizer.Delimiter{}, []rune(patStr))
		if pat.CacheKey() != expected {
			t.Errorf("Expected: %s, actual: %s", expected, pat.CacheKey())
		}
		if pat.Cacheable != cacheable {
			t.Errorf("Expected: %t, actual: %t (%s)", cacheable, pat.Cacheable, patStr)
		}
	}
	test(false, "foo !bar", "foo !bar", true)
	test(false, "foo | bar !baz", "foo | bar !baz", true)
	test(true, "foo  bar  baz", "foo\tbar\tbaz", true)
	test(true, "foo !bar", "foo", false)
	test(true, "foo !bar   baz", "foo\tbaz", false)
	test(true, "foo | bar baz", "baz", false)
	test(true, "foo | bar | baz", "", false)
	test(true, "foo | bar !baz", "", false)
	test(true, "| | foo", "", false)
	test(true, "| | | foo", "foo", false)
}

func TestCacheable(t *testing.T) {
	test := func(fuzzy bool, str string, expected string, cacheable bool) {
		pat := buildTestPattern(fuzzy, scoring.FuzzyMatchV2, true, CaseSmart, true, true, false, true, []tokenizer.Range{}, tokenizer.Delimiter{}, []rune(str))
		if pat.CacheKey() != expected {
			t.Errorf("Expected: %s, actual: %s", expected, pat.CacheKey())
		}
		if cacheable != pat.Cacheable {
			t.Errorf("Invalid Pattern.Cacheable for %q: %v (expected: %v)", str, pat.Cacheable, cacheable)
		}
	}
	test(true, "foo bar", "foo\tbar", true)
	test(true, "foo 'bar", "foo\tbar", false)
	test(true, "foo !bar", "foo", false)

	test(false, "foo bar", "foo\tbar", true)
	test(false, "foo 'bar", "foo", false)
	test(false, "foo '", "foo", true)
	test(false, "foo 'bar", "foo", false)
	test(false, "foo !bar", "foo", false)
}

func buildChunks(numChunks int) []*chunk.Chunk {
	chunks := make([]*chunk.Chunk, numChunks)
	words := []string{
		"src/main/java/com/example/service/UserService.java",
		"src/test/java/com/example/service/UserServiceTest.java",
		"docs/api/reference/endpoints.md",
		"lib/internal/utils/string_helper.go",
		"pkg/server/http/handler/auth.go",
		"build/output/release/app.exe",
		"config/production/database.yml",
		"scripts/deploy/kubernetes/setup.sh",
		"vendor/github.com/junegunn/fzf/src/core.go",
		"node_modules/.cache/babel/transform.js",
	}
	for ci := range numChunks {
		chunks[ci] = &chunk.Chunk{Count: chunk.ChunkSize}
		for i := range chunk.ChunkSize {
			text := words[(ci*chunk.ChunkSize+i)%len(words)]
			chunks[ci].Items[i] = chunk.Item{Text: charutil.ToChars([]byte(text))}
			chunks[ci].Items[i].Text.Index = int32(ci*chunk.ChunkSize + i)
		}
	}
	return chunks
}

func buildPatternWith(cache *chunk.ChunkCache, runes []rune) *Pattern {
	return BuildPattern(cache, make(map[string]*Pattern),
		true, scoring.FuzzyMatchV2, true, CaseSmart, false, true,
		false, true, []tokenizer.Range{}, tokenizer.Delimiter{}, Revision{}, runes, nil, 0)
}

func TestBitmapCacheBenefit(t *testing.T) {
	result.SortCriteria = []result.Criterion{result.ByScore, result.ByLength}

	numChunks := 100
	chunks := buildChunks(numChunks)
	queries := []string{"s", "se", "ser", "serv", "servi"}

	cache := chunk.NewChunkCache()
	for _, q := range queries {
		pat := buildPatternWith(cache, []rune(q))
		for _, c := range chunks {
			pat.Match(c, slab)
		}
	}

	runtime.GC()
	runtime.GC()
	var memWith runtime.MemStats
	runtime.ReadMemStats(&memWith)

	cache.Clear()
	runtime.GC()
	runtime.GC()
	var memWithout runtime.MemStats
	runtime.ReadMemStats(&memWithout)

	cacheMem := int64(memWith.Alloc) - int64(memWithout.Alloc)
	t.Logf("Chunks: %d, Queries: %d", numChunks, len(queries))
	t.Logf("Cache memory: %d bytes (%.1f KB)", cacheMem, float64(cacheMem)/1024)

	cache2 := chunk.NewChunkCache()
	for _, q := range queries {
		pat := buildPatternWith(cache2, []rune(q))
		for _, c := range chunks {
			pat.Match(c, slab)
		}
	}
	for _, q := range queries {
		patCached := buildPatternWith(cache2, []rune(q))
		patFresh := buildPatternWith(chunk.NewChunkCache(), []rune(q))
		var countCached, countFresh int
		for _, c := range chunks {
			countCached += len(patCached.Match(c, slab))
			countFresh += len(patFresh.Match(c, slab))
		}
		if countCached != countFresh {
			t.Errorf("query=%q: cached=%d, fresh=%d", q, countCached, countFresh)
		}
		t.Logf("query=%q: matches=%d", q, countCached)
	}
}
