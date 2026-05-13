package result

import (
	"math"
	"math/rand"
	"slices"
	"sort"
	"testing"

	"github.com/fzf/finder/pkg/charutil"
	"github.com/fzf/finder/pkg/chunk"
)

func withIndex(item *chunk.Item, index int) *chunk.Item {
	item.Text.Index = int32(index)
	return item
}

func TestOffsetSort(t *testing.T) {
	offsets := []Offset{
		{3, 5}, {2, 7},
		{1, 3}, {2, 9}}
	slices.SortFunc(offsets, CompareOffsets)

	if offsets[0][0] != 1 || offsets[0][1] != 3 ||
		offsets[1][0] != 2 || offsets[1][1] != 7 ||
		offsets[2][0] != 2 || offsets[2][1] != 9 ||
		offsets[3][0] != 3 || offsets[3][1] != 5 {
		t.Error("Invalid order:", offsets)
	}
}

func TestRankComparison(t *testing.T) {
	rank := func(vals ...uint16) Result {
		return Result{
			Points: [4]uint16{vals[0], vals[1], vals[2], vals[3]},
			Item:   &chunk.Item{Text: charutil.Chars{Index: int32(vals[4])}}}
	}
	if CompareRanks(rank(3, 0, 0, 0, 5), rank(2, 0, 0, 0, 7), false) ||
		!CompareRanks(rank(3, 0, 0, 0, 5), rank(3, 0, 0, 0, 6), false) ||
		!CompareRanks(rank(1, 2, 0, 0, 3), rank(1, 3, 0, 0, 2), false) ||
		!CompareRanks(rank(0, 0, 0, 0, 0), rank(0, 0, 0, 0, 0), false) {
		t.Error("Invalid order")
	}

	if CompareRanks(rank(3, 0, 0, 0, 5), rank(2, 0, 0, 0, 7), true) ||
		!CompareRanks(rank(3, 0, 0, 0, 5), rank(3, 0, 0, 0, 6), false) ||
		!CompareRanks(rank(1, 2, 0, 0, 3), rank(1, 3, 0, 0, 2), true) ||
		!CompareRanks(rank(0, 0, 0, 0, 0), rank(0, 0, 0, 0, 0), false) {
		t.Error("Invalid order (tac)")
	}
}

func TestResultRank(t *testing.T) {
	SortCriteria = []Criterion{ByScore, ByLength}

	str := []rune("foo")
	item1 := BuildResult(
		withIndex(&chunk.Item{Text: charutil.FromRunes(str)}, 1), []Offset{}, 2)
	if item1.Points[3] != math.MaxUint16-2 ||
		item1.Points[2] != 3 ||
		item1.Points[1] != 0 ||
		item1.Points[0] != 0 ||
		item1.Item.Index() != 1 {
		t.Error(item1)
	}

	item2 := BuildResult(&chunk.Item{Text: charutil.FromRunes(str)}, []Offset{}, 2)

	items := []Result{item1, item2}
	sort.Sort(ByRelevance(items))
	if items[0] != item2 || items[1] != item1 {
		t.Error(items)
	}

	items = []Result{item2, item1, item1, item2}
	sort.Sort(ByRelevance(items))
	if items[0] != item2 || items[1] != item2 ||
		items[2] != item1 || items[3] != item1 {
		t.Error(items)
	}

	item3 := BuildResult(
		withIndex(&chunk.Item{}, 2), []Offset{{1, 3}, {5, 7}}, 3)
	item4 := BuildResult(
		withIndex(&chunk.Item{}, 2), []Offset{{1, 2}, {6, 7}}, 4)
	item5 := BuildResult(
		withIndex(&chunk.Item{}, 2), []Offset{{1, 3}, {5, 7}}, 5)
	item6 := BuildResult(
		withIndex(&chunk.Item{}, 2), []Offset{{1, 2}, {6, 7}}, 6)
	items = []Result{item1, item2, item3, item4, item5, item6}
	sort.Sort(ByRelevance(items))
	if !(items[0] == item6 && items[1] == item5 &&
		items[2] == item4 && items[3] == item3 &&
		items[4] == item2 && items[5] == item1) {
		t.Error(items)
	}
}

func TestChunkTiebreak(t *testing.T) {
	SortCriteria = []Criterion{ByScore, ByChunk}

	score := 100
	test := func(input string, offset Offset, chunkStr string) {
		item := BuildResult(withIndex(&chunk.Item{Text: charutil.FromRunes([]rune(input))}, 1), []Offset{offset}, score)
		if !(item.Points[3] == math.MaxUint16-uint16(score) && item.Points[2] == uint16(len(chunkStr))) {
			t.Error(item.Points)
		}
	}
	test("hello foobar goodbye", Offset{8, 9}, "foobar")
	test("hello foobar goodbye", Offset{7, 18}, "foobar goodbye")
	test("hello foobar goodbye", Offset{0, 1}, "hello")
	test("hello foobar goodbye", Offset{5, 7}, "hello foobar")
}

func TestMinRank(t *testing.T) {
	r := MinRank()
	if r.Points[0] != math.MaxUint16 {
		t.Error("expected MaxUint16 in points[0]")
	}
	if r.Item.Index() != math.MinInt32 {
		t.Error("expected MinInt32 index")
	}
}

func TestRadixSortResults(t *testing.T) {
	SortCriteria = []Criterion{ByScore, ByLength}

	rng := rand.New(rand.NewSource(42))

	for _, n := range []int{128, 256, 500, 1000} {
		for _, tac := range []bool{false, true} {
			items := make([]*chunk.Item, n)
			for i := range items {
				items[i] = &chunk.Item{Text: charutil.Chars{Index: int32(i)}}
			}

			results := make([]Result, n)
			for i := range results {
				results[i] = Result{
					Item: items[i],
					Points: [4]uint16{
						uint16(rng.Intn(256)),
						uint16(rng.Intn(256)),
						uint16(rng.Intn(256)),
						uint16(rng.Intn(256)),
					},
				}
			}

			for i := 0; i < n/4; i++ {
				j := rng.Intn(n)
				k := rng.Intn(n)
				results[j].Points = results[k].Points
			}

			expected := make([]Result, n)
			copy(expected, results)
			if tac {
				sort.Sort(ByRelevanceTac(expected))
			} else {
				sort.Sort(ByRelevance(expected))
			}

			var scratch []Result
			scratch = RadixSortResults(results, tac, scratch)
			_ = scratch

			for i := range results {
				if results[i] != expected[i] {
					t.Errorf("n=%d tac=%v: mismatch at index %d: got item %d, want item %d",
						n, tac, i, results[i].Item.Index(), expected[i].Item.Index())
					break
				}
			}
		}
	}
}
