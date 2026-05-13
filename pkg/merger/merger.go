package merger

import (
	"fmt"

	"github.com/fzf/finder/pkg/chunk"
	"github.com/fzf/finder/pkg/pattern"
	"github.com/fzf/finder/pkg/result"
)

const MergerCacheMax = 100000

type Revision = pattern.Revision

func EmptyMerger(rev Revision) *Merger {
	return NewMerger(nil, [][]result.Result{}, false, false, rev, 0, 0)
}

type Merger struct {
	Pattern    *pattern.Pattern
	lists      [][]result.Result
	merged     []result.Result
	chunks     *[]*chunk.Chunk
	cursors    []int
	sorted     bool
	tac        bool
	final      bool
	count      int
	pass       bool
	startIndex int
	Rev        Revision
	MinIndex   int32
	MaxIndex   int32
}

func PassMerger(chunks *[]*chunk.Chunk, tac bool, rev Revision, startIndex int32) *Merger {
	var minIndex, maxIndex int32
	if len(*chunks) > 0 {
		minIndex = (*chunks)[0].Items[0].Index()
		maxIndex = (*chunks)[len(*chunks)-1].LastIndex(minIndex)
	}
	si := int(startIndex)
	mg := Merger{
		Pattern:    nil,
		chunks:     chunks,
		tac:        tac,
		count:      0,
		pass:       true,
		startIndex: si,
		Rev:        rev,
		MinIndex:   minIndex + startIndex,
		MaxIndex:   maxIndex}

	for _, c := range *mg.chunks {
		mg.count += c.Count
	}
	mg.count = max(0, mg.count-si)
	return &mg
}

func NewMerger(pat *pattern.Pattern, lists [][]result.Result, sorted bool, tac bool, rev Revision, minIndex int32, maxIndex int32) *Merger {
	mg := Merger{
		Pattern:  pat,
		lists:    lists,
		merged:   []result.Result{},
		chunks:   nil,
		cursors:  make([]int, len(lists)),
		sorted:   sorted,
		tac:      tac,
		final:    false,
		count:    0,
		Rev:      rev,
		MinIndex: minIndex,
		MaxIndex: maxIndex}

	for _, list := range mg.lists {
		mg.count += len(list)
	}
	return &mg
}

func (mg *Merger) Revision() Revision {
	return mg.Rev
}

func (mg *Merger) Length() int {
	return mg.count
}

func (mg *Merger) First() result.Result {
	if mg.tac && !mg.sorted {
		return mg.Get(mg.count - 1)
	}
	return mg.Get(0)
}

func (mg *Merger) FindIndex(itemIndex int32) int {
	index := -1
	if mg.pass {
		index = int(itemIndex - mg.MinIndex)
		if mg.tac {
			index = mg.count - index - 1
		}
	} else {
		for i := 0; i < mg.count; i++ {
			if mg.Get(i).Item.Index() == itemIndex {
				index = i
				break
			}
		}
	}
	return index
}

func (mg *Merger) Get(idx int) result.Result {
	if mg.chunks != nil {
		if mg.tac {
			idx = mg.count - idx - 1
		}
		idx += mg.startIndex
		firstChunk := (*mg.chunks)[0]
		if firstChunk.Count < chunk.ChunkSize && idx >= firstChunk.Count {
			idx -= firstChunk.Count
			c := (*mg.chunks)[idx/chunk.ChunkSize+1]
			return result.Result{Item: &c.Items[idx%chunk.ChunkSize]}
		}
		c := (*mg.chunks)[idx/chunk.ChunkSize]
		return result.Result{Item: &c.Items[idx%chunk.ChunkSize]}
	}

	if mg.sorted {
		return mg.mergedGet(idx)
	}

	if mg.tac {
		idx = mg.count - idx - 1
	}
	return mg.mergedGet(idx)
}

func (mg *Merger) ToMap() map[int32]result.Result {
	ret := make(map[int32]result.Result, mg.count)
	for i := 0; i < mg.count; i++ {
		r := mg.Get(i)
		ret[r.Index()] = r
	}
	return ret
}

func (mg *Merger) Cacheable() bool {
	return mg.count < MergerCacheMax
}

func (mg *Merger) mergedGet(idx int) result.Result {
	for i := len(mg.merged); i <= idx; i++ {
		minRank := result.MinRank()
		minIdx := -1
		for listIdx, list := range mg.lists {
			cursor := mg.cursors[listIdx]
			if cursor < 0 || cursor == len(list) {
				mg.cursors[listIdx] = -1
				continue
			}
			if cursor >= 0 {
				rank := list[cursor]
				if minIdx < 0 || mg.sorted && result.CompareRanks(rank, minRank, mg.tac) || !mg.sorted && rank.Item.Index() < minRank.Item.Index() {
					minRank = rank
					minIdx = listIdx
				}
			}
		}

		if minIdx >= 0 {
			chosen := mg.lists[minIdx]
			mg.merged = append(mg.merged, chosen[mg.cursors[minIdx]])
			mg.cursors[minIdx]++
		} else {
			panic(fmt.Sprintf("Index out of bounds (sorted, %d/%d)", i, mg.count))
		}
	}
	return mg.merged[idx]
}
