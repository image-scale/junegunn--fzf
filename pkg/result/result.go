package result

import (
	"math"
	"slices"
	"sort"
	"unicode"

	"github.com/fzf/finder/pkg/chunk"
	"github.com/fzf/finder/pkg/util"
)

type Offset [2]int32

type Criterion int

const (
	ByScore    Criterion = iota
	ByChunk
	ByLength
	ByBegin
	ByEnd
	ByPathname
)

type Result struct {
	Item   *chunk.Item
	Points [4]uint16
}

var SortCriteria []Criterion

func BuildResult(item *chunk.Item, offsets []Offset, score int) Result {
	if len(offsets) > 1 {
		slices.SortFunc(offsets, CompareOffsets)
	}

	minBegin := math.MaxUint16
	minEnd := math.MaxUint16
	maxEnd := 0
	validOffsetFound := false
	for _, offset := range offsets {
		b, e := int(offset[0]), int(offset[1])
		if b < e {
			minBegin = min(b, minBegin)
			minEnd = min(e, minEnd)
			maxEnd = max(e, maxEnd)
			validOffsetFound = true
		}
	}

	return BuildResultFromBounds(item, score, minBegin, minEnd, maxEnd, validOffsetFound)
}

func BuildResultFromBounds(item *chunk.Item, score int, minBegin, minEnd, maxEnd int, validOffsetFound bool) Result {
	result := Result{Item: item}
	numChars := item.Text.Length()

	for idx, criterion := range SortCriteria {
		val := uint16(math.MaxUint16)
		switch criterion {
		case ByScore:
			val = math.MaxUint16 - util.AsUint16(score)
		case ByChunk:
			if validOffsetFound {
				b := minBegin
				e := maxEnd
				for ; b >= 1; b-- {
					if unicode.IsSpace(item.Text.Get(b - 1)) {
						break
					}
				}
				for ; e < numChars; e++ {
					if unicode.IsSpace(item.Text.Get(e)) {
						break
					}
				}
				val = util.AsUint16(e - b)
			}
		case ByLength:
			val = item.TrimLength()
		case ByPathname:
			if validOffsetFound {
				lastDelim := -1
				s := item.Text.ToString()
				for i := len(s) - 1; i >= 0; i-- {
					if s[i] == '/' || s[i] == '\\' {
						lastDelim = i
						break
					}
				}
				if lastDelim <= minBegin {
					val = util.AsUint16(minBegin - lastDelim)
				}
			}
		case ByBegin, ByEnd:
			if validOffsetFound {
				whitePrefixLen := 0
				for i := range numChars {
					r := item.Text.Get(i)
					whitePrefixLen = i
					if i == minBegin || !unicode.IsSpace(r) {
						break
					}
				}
				if criterion == ByBegin {
					val = util.AsUint16(minEnd - whitePrefixLen)
				} else {
					val = util.AsUint16(math.MaxUint16 - math.MaxUint16*(maxEnd-whitePrefixLen)/(int(item.TrimLength())+1))
				}
			}
		}
		result.Points[3-idx] = val
	}

	return result
}

func CompareOffsets(a, b Offset) int {
	if a[0] < b[0] {
		return -1
	}
	if a[0] > b[0] {
		return 1
	}
	if a[1] < b[1] {
		return -1
	}
	if a[1] > b[1] {
		return 1
	}
	return 0
}

func CompareRanks(irank, jrank Result, tac bool) bool {
	for idx := 3; idx >= 0; idx-- {
		left := irank.Points[idx]
		right := jrank.Points[idx]
		if left < right {
			return true
		} else if left > right {
			return false
		}
	}
	return (irank.Item.Index() <= jrank.Item.Index()) != tac
}

func SortKey(r *Result) uint64 {
	return uint64(r.Points[0]) | uint64(r.Points[1])<<16 | uint64(r.Points[2])<<32 | uint64(r.Points[3])<<48
}

func (result *Result) Index() int32 {
	return result.Item.Index()
}

func MinRank() Result {
	return Result{Item: &chunk.MinItem, Points: [4]uint16{math.MaxUint16, 0, 0, 0}}
}

type ByRelevance []Result

func (a ByRelevance) Len() int           { return len(a) }
func (a ByRelevance) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByRelevance) Less(i, j int) bool { return CompareRanks(a[i], a[j], false) }

type ByRelevanceTac []Result

func (a ByRelevanceTac) Len() int           { return len(a) }
func (a ByRelevanceTac) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByRelevanceTac) Less(i, j int) bool { return CompareRanks(a[i], a[j], true) }

func RadixSortResults(a []Result, tac bool, scratch []Result) []Result {
	n := len(a)
	if n < 128 {
		if tac {
			sort.Sort(ByRelevanceTac(a))
		} else {
			sort.Sort(ByRelevance(a))
		}
		return scratch[:0]
	}

	if cap(scratch) < n {
		scratch = make([]Result, n)
	}
	buf := scratch[:n]
	src, dst := a, buf
	scattered := 0

	for pass := range 8 {
		shift := uint(pass) * 8

		var count [256]int
		for i := range src {
			count[byte(SortKey(&src[i])>>shift)]++
		}

		if count[byte(SortKey(&src[0])>>shift)] == n {
			continue
		}

		var offset [256]int
		for i := 1; i < 256; i++ {
			offset[i] = offset[i-1] + count[i-1]
		}

		for i := range src {
			b := byte(SortKey(&src[i]) >> shift)
			dst[offset[b]] = src[i]
			offset[b]++
		}

		src, dst = dst, src
		scattered++
	}

	if scattered%2 == 1 {
		copy(a, src)
	}

	if tac {
		i := 0
		for i < n {
			ki := SortKey(&a[i])
			j := i + 1
			for j < n && SortKey(&a[j]) == ki {
				j++
			}
			if j-i > 1 {
				for l, r := i, j-1; l < r; l, r = l+1, r-1 {
					a[l], a[r] = a[r], a[l]
				}
			}
			i = j
		}
	}
	return scratch
}
