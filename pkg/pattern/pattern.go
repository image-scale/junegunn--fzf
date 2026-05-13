package pattern

import (
	"fmt"
	"regexp"
	"strings"
	"unsafe"

	"github.com/fzf/finder/pkg/charutil"
	"github.com/fzf/finder/pkg/chunk"
	"github.com/fzf/finder/pkg/result"
	"github.com/fzf/finder/pkg/scoring"
	"github.com/fzf/finder/pkg/tokenizer"
)

type TermType int

const (
	TermFuzzy TermType = iota
	TermExact
	TermExactBoundary
	TermPrefix
	TermSuffix
	TermEqual
)

type CaseMode int

const (
	CaseSmart CaseMode = iota
	CaseIgnore
	CaseRespect
)

type Revision struct {
	Major int
	Minor int
}

type Term struct {
	Typ           TermType
	Inv           bool
	Text          []rune
	CaseSensitive bool
	Normalize     bool
}

func (t Term) String() string {
	return fmt.Sprintf("Term{typ: %d, inv: %v, text: []rune(%q), caseSensitive: %v}", t.Typ, t.Inv, string(t.Text), t.CaseSensitive)
}

type TermSet []Term

type Transformed struct {
	Rev    Revision
	Tokens []tokenizer.Token
}

type Pattern struct {
	Fuzzy         bool
	FuzzyAlgo     scoring.MatchFunc
	Extended      bool
	CaseSensitive bool
	Normalize     bool
	Forward       bool
	WithPos       bool
	Text          []rune
	TermSets      []TermSet
	Sortable      bool
	Cacheable     bool
	cacheKey      string
	Delimiter     tokenizer.Delimiter
	Nth           []tokenizer.Range
	Rev           Revision
	procFun       [6]scoring.MatchFunc
	Cache         *chunk.ChunkCache
	Denylist      map[int32]struct{}
	StartIndex    int32
	directAlgo    scoring.MatchFunc
	directTerm    *Term
}

var splitRegex = regexp.MustCompile(" +")

func BuildPattern(cache *chunk.ChunkCache, patternCache map[string]*Pattern, fuzzy bool, fuzzyAlgo scoring.MatchFunc, extended bool, caseMode CaseMode, normalize bool, forward bool,
	withPos bool, cacheable bool, nth []tokenizer.Range, delimiter tokenizer.Delimiter, rev Revision, runes []rune, denylist map[int32]struct{}, startIndex int32) *Pattern {

	var asString string
	if extended {
		asString = strings.TrimLeft(string(runes), " ")
		for strings.HasSuffix(asString, " ") && !strings.HasSuffix(asString, "\\ ") {
			asString = asString[:len(asString)-1]
		}
	} else {
		asString = string(runes)
	}

	cached, found := patternCache[asString]
	if found {
		return cached
	}

	caseSensitive := true
	sortable := true
	termSets := []TermSet{}

	if extended {
		termSets = ParseTerms(fuzzy, caseMode, normalize, asString)
		sortable = false
	Loop:
		for _, termSet := range termSets {
			for idx, term := range termSet {
				if !term.Inv {
					sortable = true
				}
				if !cacheable || idx > 0 || term.Inv || fuzzy && term.Typ != TermFuzzy || !fuzzy && term.Typ != TermExact {
					cacheable = false
					if sortable {
						break Loop
					}
				}
			}
		}
	} else {
		lowerString := strings.ToLower(asString)
		normalize = normalize &&
			lowerString == string(scoring.NormalizeRunes([]rune(lowerString)))
		caseSensitive = caseMode == CaseRespect ||
			caseMode == CaseSmart && lowerString != asString
		if !caseSensitive {
			asString = lowerString
		}
	}

	ptr := &Pattern{
		Fuzzy:         fuzzy,
		FuzzyAlgo:     fuzzyAlgo,
		Extended:      extended,
		CaseSensitive: caseSensitive,
		Normalize:     normalize,
		Forward:       forward,
		WithPos:       withPos,
		Text:          []rune(asString),
		TermSets:      termSets,
		Sortable:      sortable,
		Cacheable:     cacheable,
		Nth:           nth,
		Rev:           rev,
		Delimiter:     delimiter,
		Cache:         cache,
		Denylist:      denylist,
		StartIndex:    startIndex,
	}

	ptr.cacheKey = ptr.buildCacheKey()
	ptr.directAlgo, ptr.directTerm = ptr.buildDirectAlgo(fuzzyAlgo)
	ptr.procFun[TermFuzzy] = fuzzyAlgo
	ptr.procFun[TermEqual] = scoring.EqualMatch
	ptr.procFun[TermExact] = scoring.ExactMatchNaive
	ptr.procFun[TermExactBoundary] = scoring.ExactMatchBoundary
	ptr.procFun[TermPrefix] = scoring.PrefixMatch
	ptr.procFun[TermSuffix] = scoring.SuffixMatch

	patternCache[asString] = ptr
	return ptr
}

func ParseTerms(fuzzy bool, caseMode CaseMode, normalize bool, str string) []TermSet {
	str = strings.ReplaceAll(str, "\\ ", "\t")
	tokens := splitRegex.Split(str, -1)
	sets := []TermSet{}
	set := TermSet{}
	switchSet := false
	afterBar := false
	for _, token := range tokens {
		typ, inv, text := TermFuzzy, false, strings.ReplaceAll(token, "\t", " ")
		lowerText := strings.ToLower(text)
		caseSensitive := caseMode == CaseRespect ||
			caseMode == CaseSmart && text != lowerText
		normalizeTerm := normalize &&
			lowerText == string(scoring.NormalizeRunes([]rune(lowerText)))
		if !caseSensitive {
			text = lowerText
		}
		if !fuzzy {
			typ = TermExact
		}

		if len(set) > 0 && !afterBar && text == "|" {
			switchSet = false
			afterBar = true
			continue
		}
		afterBar = false

		if strings.HasPrefix(text, "!") {
			inv = true
			typ = TermExact
			text = text[1:]
		}

		if text != "$" && strings.HasSuffix(text, "$") {
			typ = TermSuffix
			text = text[:len(text)-1]
		}

		if len(text) > 2 && strings.HasPrefix(text, "'") && strings.HasSuffix(text, "'") {
			typ = TermExactBoundary
			text = text[1 : len(text)-1]
		} else if strings.HasPrefix(text, "'") {
			if fuzzy && !inv {
				typ = TermExact
			} else {
				typ = TermFuzzy
			}
			text = text[1:]
		} else if strings.HasPrefix(text, "^") {
			if typ == TermSuffix {
				typ = TermEqual
			} else {
				typ = TermPrefix
			}
			text = text[1:]
		}

		if len(text) > 0 {
			if switchSet {
				sets = append(sets, set)
				set = TermSet{}
			}
			textRunes := []rune(text)
			if normalizeTerm {
				textRunes = scoring.NormalizeRunes(textRunes)
			}
			set = append(set, Term{
				Typ:           typ,
				Inv:           inv,
				Text:          textRunes,
				CaseSensitive: caseSensitive,
				Normalize:     normalizeTerm})
			switchSet = true
		}
	}
	if len(set) > 0 {
		sets = append(sets, set)
	}
	return sets
}

func (p *Pattern) IsEmpty() bool {
	if len(p.Denylist) > 0 {
		return false
	}
	if !p.Extended {
		return len(p.Text) == 0
	}
	return len(p.TermSets) == 0
}

func (p *Pattern) AsString() string {
	return string(p.Text)
}

func (p *Pattern) buildCacheKey() string {
	if !p.Extended {
		return p.AsString()
	}
	cacheableTerms := []string{}
	for _, termSet := range p.TermSets {
		if len(termSet) == 1 && !termSet[0].Inv && (p.Fuzzy || termSet[0].Typ == TermExact) {
			cacheableTerms = append(cacheableTerms, string(termSet[0].Text))
		}
	}
	return strings.Join(cacheableTerms, "\t")
}

func (p *Pattern) buildDirectAlgo(fuzzyAlgo scoring.MatchFunc) (scoring.MatchFunc, *Term) {
	if !p.Extended || len(p.Nth) > 0 {
		return nil, nil
	}
	if len(p.TermSets) == 1 && len(p.TermSets[0]) == 1 {
		t := &p.TermSets[0][0]
		if !t.Inv && t.Typ == TermFuzzy {
			return fuzzyAlgo, t
		}
	}
	return nil, nil
}

func (p *Pattern) CacheKey() string {
	return p.cacheKey
}

func (p *Pattern) Match(c *chunk.Chunk, slab *charutil.Slab) []result.Result {
	cacheKey := p.CacheKey()

	var cachedBitmap *chunk.ChunkBitmap
	if p.Cacheable {
		cachedBitmap = p.Cache.Lookup(c, cacheKey)
	}
	if cachedBitmap == nil {
		cachedBitmap = p.Cache.Search(c, cacheKey)
	}

	matches, bitmap := p.MatchChunk(c, cachedBitmap, slab)

	if p.Cacheable {
		p.Cache.Add(c, cacheKey, bitmap, len(matches))
	}
	return matches
}

func (p *Pattern) MatchChunk(c *chunk.Chunk, cachedBitmap *chunk.ChunkBitmap, slab *charutil.Slab) ([]result.Result, chunk.ChunkBitmap) {
	matches := []result.Result{}
	var bitmap chunk.ChunkBitmap

	startIdx := 0
	if p.StartIndex > 0 && c.Count > 0 && c.Items[0].Index() < p.StartIndex {
		startIdx = int(p.StartIndex - c.Items[0].Index())
		if startIdx >= c.Count {
			return matches, bitmap
		}
	}

	hasCachedBitmap := cachedBitmap != nil

	if p.directAlgo != nil && len(p.Denylist) == 0 {
		t := p.directTerm
		for idx := startIdx; idx < c.Count; idx++ {
			if hasCachedBitmap && cachedBitmap[idx/64]&(uint64(1)<<(idx%64)) == 0 {
				continue
			}
			res, _ := p.directAlgo(t.CaseSensitive, t.Normalize, p.Forward,
				&c.Items[idx].Text, t.Text, p.WithPos, slab)
			if res.Start >= 0 {
				bitmap[idx/64] |= uint64(1) << (idx % 64)
				matches = append(matches, result.BuildResultFromBounds(
					&c.Items[idx], res.Score,
					int(res.Start), int(res.End), int(res.End), true))
			}
		}
		return matches, bitmap
	}

	if len(p.Denylist) == 0 {
		for idx := startIdx; idx < c.Count; idx++ {
			if hasCachedBitmap && cachedBitmap[idx/64]&(uint64(1)<<(idx%64)) == 0 {
				continue
			}
			if match, _, _ := p.MatchItem(&c.Items[idx], p.WithPos, slab); match.Item != nil {
				bitmap[idx/64] |= uint64(1) << (idx % 64)
				matches = append(matches, match)
			}
		}
		return matches, bitmap
	}

	for idx := startIdx; idx < c.Count; idx++ {
		if hasCachedBitmap && cachedBitmap[idx/64]&(uint64(1)<<(idx%64)) == 0 {
			continue
		}
		if _, prs := p.Denylist[c.Items[idx].Index()]; prs {
			continue
		}
		if match, _, _ := p.MatchItem(&c.Items[idx], p.WithPos, slab); match.Item != nil {
			bitmap[idx/64] |= uint64(1) << (idx % 64)
			matches = append(matches, match)
		}
	}
	return matches, bitmap
}

func (p *Pattern) MatchItem(item *chunk.Item, withPos bool, slab *charutil.Slab) (result.Result, []result.Offset, *[]int) {
	if p.Extended {
		if offsets, bonus, pos := p.extendedMatch(item, withPos, slab); len(offsets) == len(p.TermSets) {
			return result.BuildResult(item, offsets, bonus), offsets, pos
		}
		return result.Result{}, nil, nil
	}
	offset, bonus, pos := p.basicMatch(item, withPos, slab)
	if sidx := offset[0]; sidx >= 0 {
		offsets := []result.Offset{offset}
		return result.BuildResult(item, offsets, bonus), offsets, pos
	}
	return result.Result{}, nil, nil
}

func (p *Pattern) basicMatch(item *chunk.Item, withPos bool, slab *charutil.Slab) (result.Offset, int, *[]int) {
	var input []tokenizer.Token
	if len(p.Nth) == 0 {
		input = []tokenizer.Token{{Text: &item.Text, PrefixLength: 0}}
	} else {
		input = p.transformInput(item)
	}
	if p.Fuzzy {
		return p.iter(p.FuzzyAlgo, input, p.CaseSensitive, p.Normalize, p.Forward, p.Text, withPos, slab)
	}
	return p.iter(scoring.ExactMatchNaive, input, p.CaseSensitive, p.Normalize, p.Forward, p.Text, withPos, slab)
}

func (p *Pattern) extendedMatch(item *chunk.Item, withPos bool, slab *charutil.Slab) ([]result.Offset, int, *[]int) {
	var input []tokenizer.Token
	if len(p.Nth) == 0 {
		input = []tokenizer.Token{{Text: &item.Text, PrefixLength: 0}}
	} else {
		input = p.transformInput(item)
	}
	offsets := []result.Offset{}
	var totalScore int
	var allPos *[]int
	if withPos {
		allPos = &[]int{}
	}
	for _, termSet := range p.TermSets {
		var offset result.Offset
		var currentScore int
		matched := false
		for _, term := range termSet {
			pfun := p.procFun[term.Typ]
			off, score, pos := p.iter(pfun, input, term.CaseSensitive, term.Normalize, p.Forward, term.Text, withPos, slab)
			if sidx := off[0]; sidx >= 0 {
				if term.Inv {
					continue
				}
				offset, currentScore = off, score
				matched = true
				if withPos {
					if pos != nil {
						*allPos = append(*allPos, *pos...)
					} else {
						for idx := off[0]; idx < off[1]; idx++ {
							*allPos = append(*allPos, int(idx))
						}
					}
				}
				break
			} else if term.Inv {
				offset, currentScore = result.Offset{0, 0}, 0
				matched = true
				continue
			}
		}
		if matched {
			offsets = append(offsets, offset)
			totalScore += currentScore
		}
	}
	return offsets, totalScore, allPos
}

func (p *Pattern) transformInput(item *chunk.Item) []tokenizer.Token {
	if item.Transformed != nil {
		if trans, ok := item.Transformed.(*Transformed); ok {
			if trans.Rev == p.Rev {
				return trans.Tokens
			}
		}
	}

	tokens := tokenizer.Tokenize(item.Text.ToString(), p.Delimiter)
	ret := tokenizer.Transform(tokens, p.Nth)
	if len(ret) > 0 && !p.Delimiter.IsAwk() {
		chars := ret[len(ret)-1].Text
		stripped := tokenizer.StripLastDelimiter(chars.ToString(), p.Delimiter)
		newChars := charutil.ToChars(stringBytes(stripped))
		ret[len(ret)-1].Text = &newChars
	}
	item.Transformed = &Transformed{p.Rev, ret}
	return ret
}

func (p *Pattern) iter(pfun scoring.MatchFunc, tokens []tokenizer.Token, caseSensitive bool, normalize bool, forward bool, pat []rune, withPos bool, slab *charutil.Slab) (result.Offset, int, *[]int) {
	for _, part := range tokens {
		if res, pos := pfun(caseSensitive, normalize, forward, part.Text, pat, withPos, slab); res.Start >= 0 {
			sidx := int32(res.Start) + part.PrefixLength
			eidx := int32(res.End) + part.PrefixLength
			if pos != nil {
				for idx := range *pos {
					(*pos)[idx] += int(part.PrefixLength)
				}
			}
			return result.Offset{sidx, eidx}, res.Score, pos
		}
	}
	return result.Offset{-1, -1}, 0, nil
}

func stringBytes(s string) []byte {
	return unsafe.Slice(unsafe.StringData(s), len(s))
}
