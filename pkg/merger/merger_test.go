package merger

import (
	"fmt"
	"math/rand"
	"sort"
	"testing"

	"github.com/fzf/finder/pkg/charutil"
	"github.com/fzf/finder/pkg/chunk"
	"github.com/fzf/finder/pkg/result"
)

func assertTrue(t *testing.T, cond bool, msg ...string) {
	t.Helper()
	if !cond {
		t.Error(msg)
	}
}

func randResult() result.Result {
	str := fmt.Sprintf("%d", rand.Uint32())
	chars := charutil.ToChars([]byte(str))
	chars.Index = rand.Int31()
	return result.Result{Item: &chunk.Item{Text: chars}}
}

func TestEmptyMerger(t *testing.T) {
	r := Revision{}
	assertTrue(t, EmptyMerger(r).Length() == 0, "Not empty")
	assertTrue(t, EmptyMerger(r).count == 0, "Invalid count")
	assertTrue(t, len(EmptyMerger(r).lists) == 0, "Invalid lists")
	assertTrue(t, len(EmptyMerger(r).merged) == 0, "Invalid merged list")
}

func buildLists(partiallySorted bool) ([][]result.Result, []result.Result) {
	numLists := 4
	lists := make([][]result.Result, numLists)
	for i := range numLists {
		numResults := rand.Int() % 20
		lists[i] = make([]result.Result, numResults)
		for j := range numResults {
			lists[i][j] = randResult()
		}
		if partiallySorted {
			sort.Sort(result.ByRelevance(lists[i]))
		}
	}
	items := []result.Result{}
	for _, list := range lists {
		items = append(items, list...)
	}
	return lists, items
}

func TestMergerUnsorted(t *testing.T) {
	lists, _ := buildLists(false)

	for _, list := range lists {
		sort.Slice(list, func(i, j int) bool {
			return list[i].Item.Index() < list[j].Item.Index()
		})
	}
	items := []result.Result{}
	for _, list := range lists {
		items = append(items, list...)
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].Item.Index() < items[j].Item.Index()
	})
	cnt := len(items)

	mg := NewMerger(nil, lists, false, false, Revision{}, 0, 0)
	assertTrue(t, cnt == mg.Length(), "Invalid Length")
	for i := range cnt {
		assertTrue(t, items[i] == mg.Get(i), "Invalid Get")
	}
}

func TestMergerSorted(t *testing.T) {
	lists, items := buildLists(true)
	cnt := len(items)

	mg := NewMerger(nil, lists, true, false, Revision{}, 0, 0)
	assertTrue(t, cnt == mg.Length(), "Invalid Length")
	sort.Sort(result.ByRelevance(items))
	for i := range cnt {
		if items[i] != mg.Get(i) {
			t.Error("Not sorted", items[i], mg.Get(i))
		}
	}

	mg2 := NewMerger(nil, lists, true, false, Revision{}, 0, 0)
	for i := cnt - 1; i >= 0; i-- {
		if items[i] != mg2.Get(i) {
			t.Error("Not sorted", items[i], mg2.Get(i))
		}
	}
}

func TestMergerCacheable(t *testing.T) {
	mg := NewMerger(nil, [][]result.Result{}, false, false, Revision{}, 0, 0)
	if !mg.Cacheable() {
		t.Error("Empty merger should be cacheable")
	}
}

func TestMergerFindIndex(t *testing.T) {
	items := make([]result.Result, 3)
	for i := range items {
		chars := charutil.ToChars([]byte(fmt.Sprintf("item%d", i)))
		chars.Index = int32(i * 10)
		items[i] = result.Result{Item: &chunk.Item{Text: chars}}
	}
	mg := NewMerger(nil, [][]result.Result{items}, false, false, Revision{}, 0, 0)
	idx := mg.FindIndex(int32(10))
	if idx < 0 {
		t.Error("Should find item with index 10")
	}
}
