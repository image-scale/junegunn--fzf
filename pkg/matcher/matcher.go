package matcher

import (
	"fmt"
	"runtime"
	gosync "sync"
	"sync/atomic"
	"time"

	"github.com/fzf/finder/pkg/charutil"
	"github.com/fzf/finder/pkg/chunk"
	"github.com/fzf/finder/pkg/merger"
	"github.com/fzf/finder/pkg/pattern"
	"github.com/fzf/finder/pkg/result"
	fzfsync "github.com/fzf/finder/pkg/sync"
)

const (
	ProgressMinDuration = 200 * time.Millisecond
	Slab16Size          = 100 * 1024
	Slab32Size          = 2048
)

const (
	EvtSearchNew      fzfsync.EventKind = 200
	EvtSearchProgress fzfsync.EventKind = 201
	EvtSearchFin      fzfsync.EventKind = 202
)

const (
	reqRetry fzfsync.EventKind = iota
	reqReset
	reqQuit
)

type MatchRequest struct {
	Chunks  []*chunk.Chunk
	Pattern *pattern.Pattern
	Final   bool
	Sort    bool
	Rev     pattern.Revision
}

type MatchResult struct {
	Merger     *merger.Merger
	PassMerger *merger.Merger
	Cancelled  bool
}

func (mr MatchResult) cacheable() bool {
	return mr.Merger != nil && mr.Merger.Cacheable()
}

type Matcher struct {
	cache          *chunk.ChunkCache
	patternBuilder func([]rune) *pattern.Pattern
	sort           bool
	tac            bool
	eventBox       *fzfsync.EventBus
	reqBox         *fzfsync.EventBus
	partitions     int
	slab           []*charutil.Slab
	sortBuf        [][]result.Result
	mergerCache    map[string]MatchResult
	rev            pattern.Revision
	scanMutex      gosync.Mutex
	cancelScan     *fzfsync.AtomicFlag
}

func NewMatcher(cache *chunk.ChunkCache, patternBuilder func([]rune) *pattern.Pattern,
	sort bool, tac bool, eventBox *fzfsync.EventBus, rev pattern.Revision, threads int) *Matcher {
	partitions := runtime.NumCPU()
	if threads > 0 {
		partitions = threads
	}
	return &Matcher{
		cache:          cache,
		patternBuilder: patternBuilder,
		sort:           sort,
		tac:            tac,
		eventBox:       eventBox,
		reqBox:         fzfsync.NewEventBus(),
		partitions:     partitions,
		slab:           make([]*charutil.Slab, partitions),
		sortBuf:        make([][]result.Result, partitions),
		mergerCache:    make(map[string]MatchResult),
		rev:            rev,
		cancelScan:     fzfsync.NewAtomicFlag(false),
	}
}

func (m *Matcher) Loop() {
	prevCount := 0

	for {
		var request MatchRequest

		stop := false
		m.reqBox.Wait(func(events *fzfsync.EventMap) {
			for t, val := range *events {
				if t == reqQuit {
					stop = true
					return
				}
				switch val := val.(type) {
				case MatchRequest:
					request = val
				default:
					panic(fmt.Sprintf("Unexpected type: %T", val))
				}
			}
			events.Clear()
		})
		if stop {
			break
		}

		cacheCleared := false
		if request.Sort != m.sort || request.Rev != m.rev {
			m.sort = request.Sort
			m.mergerCache = make(map[string]MatchResult)
			if request.Rev.Major != m.rev.Major {
				m.cache.Clear()
			}
			m.rev = request.Rev
			cacheCleared = true
		}

		patternString := request.Pattern.AsString()
		var res MatchResult
		count := chunk.CountItems(request.Chunks)

		if !cacheCleared {
			if count == prevCount {
				if cached, found := m.mergerCache[patternString]; found {
					res = cached
				}
			} else {
				prevCount = count
				m.mergerCache = make(map[string]MatchResult)
			}
		}

		if res.Merger == nil {
			m.scanMutex.Lock()
			res = m.scan(request)
			m.scanMutex.Unlock()
		}

		if !res.Cancelled {
			if res.cacheable() {
				m.mergerCache[patternString] = res
			}
			m.eventBox.Set(EvtSearchFin, res)
		}
	}
}

type partialResult struct {
	index   int
	matches []result.Result
}

func (m *Matcher) scan(request MatchRequest) MatchResult {
	startedAt := time.Now()

	numChunks := len(request.Chunks)
	if numChunks == 0 {
		mg := merger.EmptyMerger(request.Rev)
		return MatchResult{mg, mg, false}
	}
	pat := request.Pattern
	passMerger := merger.PassMerger(&request.Chunks, m.tac, request.Rev, pat.StartIndex)
	if pat.IsEmpty() {
		return MatchResult{passMerger, passMerger, false}
	}

	minIndex := request.Chunks[0].Items[0].Index()
	maxIndex := request.Chunks[numChunks-1].LastIndex(minIndex)
	cancelled := fzfsync.NewAtomicFlag(false)

	numWorkers := min(m.partitions, numChunks)
	var nextChunk atomic.Int32
	resultChan := make(chan partialResult, numWorkers)
	countChan := make(chan int, numChunks)
	waitGroup := gosync.WaitGroup{}

	for idx := range numWorkers {
		waitGroup.Add(1)
		if m.slab[idx] == nil {
			m.slab[idx] = charutil.NewSlab(Slab16Size, Slab32Size)
		}
		go func(idx int, slab *charutil.Slab) {
			defer waitGroup.Done()
			var matches []result.Result
			for {
				ci := int(nextChunk.Add(1)) - 1
				if ci >= numChunks {
					break
				}
				chunkMatches := request.Pattern.Match(request.Chunks[ci], slab)
				matches = append(matches, chunkMatches...)
				if cancelled.Get() {
					return
				}
				countChan <- len(chunkMatches)
			}
			if m.sort && request.Pattern.Sortable {
				m.sortBuf[idx] = result.RadixSortResults(matches, m.tac, m.sortBuf[idx])
			}
			resultChan <- partialResult{idx, matches}
		}(idx, m.slab[idx])
	}

	wait := func() bool {
		cancelled.Set(true)
		waitGroup.Wait()
		return true
	}

	count := 0
	matchCount := 0
	for matchesInChunk := range countChan {
		count++
		matchCount += matchesInChunk

		if count == numChunks {
			break
		}

		if m.cancelScan.Get() || m.reqBox.Peek(reqReset) {
			return MatchResult{nil, nil, wait()}
		}

		if time.Since(startedAt) > ProgressMinDuration {
			m.eventBox.Set(EvtSearchProgress, float32(count)/float32(numChunks))
		}
	}

	partialResults := make([][]result.Result, numWorkers)
	for range numWorkers {
		pr := <-resultChan
		partialResults[pr.index] = pr.matches
	}
	mg := merger.NewMerger(pat, partialResults, m.sort && request.Pattern.Sortable, m.tac, request.Rev, minIndex, maxIndex)
	return MatchResult{mg, passMerger, false}
}

func (m *Matcher) Reset(chunks []*chunk.Chunk, patternRunes []rune, cancel bool, final bool, sort bool, rev pattern.Revision) {
	pat := m.patternBuilder(patternRunes)

	var event fzfsync.EventKind
	if cancel {
		event = reqReset
	} else {
		event = reqRetry
	}
	m.reqBox.Set(event, MatchRequest{chunks, pat, final, sort, rev})
}

func (m *Matcher) CancelScan() {
	m.cancelScan.Set(true)
	m.scanMutex.Lock()
	m.cancelScan.Set(false)
}

func (m *Matcher) ResumeScan() {
	m.scanMutex.Unlock()
}

func (m *Matcher) Stop() {
	m.reqBox.Set(reqQuit, nil)
}
