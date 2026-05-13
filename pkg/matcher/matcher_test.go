package matcher

import (
	"testing"
	"time"

	"github.com/fzf/finder/pkg/charutil"
	"github.com/fzf/finder/pkg/chunk"
	"github.com/fzf/finder/pkg/pattern"
	"github.com/fzf/finder/pkg/scoring"
	"github.com/fzf/finder/pkg/sync"
	"github.com/fzf/finder/pkg/tokenizer"
)

func init() {
	scoring.Setup("default")
}

func makePatternBuilder() func([]rune) *pattern.Pattern {
	cache := chunk.NewChunkCache()
	patternCache := make(map[string]*pattern.Pattern)
	return func(runes []rune) *pattern.Pattern {
		return pattern.BuildPattern(
			cache, patternCache,
			true, scoring.FuzzyMatchV2,
			false, pattern.CaseSmart,
			true, true, false, true,
			nil, tokenizer.Delimiter{},
			pattern.Revision{}, runes,
			nil, 0,
		)
	}
}

func buildChunks(items []string) ([]*chunk.Chunk, *chunk.ChunkCache) {
	cache := chunk.NewChunkCache()
	idx := int32(0)
	cl := chunk.NewChunkList(cache, func(item *chunk.Item, data []byte) bool {
		item.Text = charutil.ToChars(data)
		item.Text.Index = idx
		idx++
		return true
	})
	for _, s := range items {
		cl.Push([]byte(s))
	}
	chunks, _, _ := cl.Snapshot(0)
	return chunks, cache
}

func TestMatcherEmptyPattern(t *testing.T) {
	items := []string{"foo", "bar", "baz"}
	chunks, cache := buildChunks(items)

	eventBox := sync.NewEventBus()
	builder := makePatternBuilder()

	m := NewMatcher(cache, builder, true, false, eventBox, pattern.Revision{}, 2)
	go m.Loop()
	defer m.Stop()

	m.Reset(chunks, []rune{}, false, true, true, pattern.Revision{})

	eventBox.WaitFor(EvtSearchFin)
	eventBox.Wait(func(events *sync.EventMap) {
		val := (*events)[EvtSearchFin]
		res, ok := val.(MatchResult)
		if !ok {
			t.Fatal("Expected MatchResult")
		}
		if res.Cancelled {
			t.Fatal("Should not be cancelled")
		}
		if res.Merger.Length() != 3 {
			t.Errorf("Expected 3 items, got %d", res.Merger.Length())
		}
		events.Clear()
	})
}

func TestMatcherWithPattern(t *testing.T) {
	items := []string{"apple", "banana", "pineapple", "grape", "papaya"}
	chunks, cache := buildChunks(items)

	eventBox := sync.NewEventBus()
	builder := makePatternBuilder()

	m := NewMatcher(cache, builder, true, false, eventBox, pattern.Revision{}, 2)
	go m.Loop()
	defer m.Stop()

	m.Reset(chunks, []rune("apple"), false, true, true, pattern.Revision{})

	eventBox.WaitFor(EvtSearchFin)
	eventBox.Wait(func(events *sync.EventMap) {
		val := (*events)[EvtSearchFin]
		res := val.(MatchResult)
		if res.Cancelled {
			t.Fatal("Should not be cancelled")
		}
		if res.Merger.Length() != 2 {
			t.Errorf("Expected 2 matches (apple, pineapple), got %d", res.Merger.Length())
		}
		if res.PassMerger.Length() != 5 {
			t.Errorf("Expected 5 pass items, got %d", res.PassMerger.Length())
		}
		events.Clear()
	})
}

func TestMatcherNoMatch(t *testing.T) {
	items := []string{"foo", "bar", "baz"}
	chunks, cache := buildChunks(items)

	eventBox := sync.NewEventBus()
	builder := makePatternBuilder()

	m := NewMatcher(cache, builder, true, false, eventBox, pattern.Revision{}, 2)
	go m.Loop()
	defer m.Stop()

	m.Reset(chunks, []rune("zzzzz"), false, true, true, pattern.Revision{})

	eventBox.WaitFor(EvtSearchFin)
	eventBox.Wait(func(events *sync.EventMap) {
		val := (*events)[EvtSearchFin]
		res := val.(MatchResult)
		if res.Merger.Length() != 0 {
			t.Errorf("Expected 0 matches, got %d", res.Merger.Length())
		}
		events.Clear()
	})
}

func TestMatcherEmptyChunks(t *testing.T) {
	eventBox := sync.NewEventBus()
	builder := makePatternBuilder()
	cache := chunk.NewChunkCache()

	m := NewMatcher(cache, builder, true, false, eventBox, pattern.Revision{}, 2)
	go m.Loop()
	defer m.Stop()

	m.Reset(nil, []rune("test"), false, true, true, pattern.Revision{})

	eventBox.WaitFor(EvtSearchFin)
	eventBox.Wait(func(events *sync.EventMap) {
		val := (*events)[EvtSearchFin]
		res := val.(MatchResult)
		if res.Merger.Length() != 0 {
			t.Errorf("Expected 0 matches, got %d", res.Merger.Length())
		}
		events.Clear()
	})
}

func TestMatcherCaching(t *testing.T) {
	items := []string{"hello", "world", "help"}
	chunks, cache := buildChunks(items)

	eventBox := sync.NewEventBus()
	builder := makePatternBuilder()

	m := NewMatcher(cache, builder, true, false, eventBox, pattern.Revision{}, 2)
	go m.Loop()
	defer m.Stop()

	m.Reset(chunks, []rune("hel"), false, true, true, pattern.Revision{})
	eventBox.WaitFor(EvtSearchFin)
	eventBox.Wait(func(events *sync.EventMap) {
		val := (*events)[EvtSearchFin]
		res := val.(MatchResult)
		if res.Merger.Length() != 2 {
			t.Errorf("Expected 2 matches, got %d", res.Merger.Length())
		}
		events.Clear()
	})

	m.Reset(chunks, []rune("hel"), false, true, true, pattern.Revision{})
	eventBox.WaitFor(EvtSearchFin)
	eventBox.Wait(func(events *sync.EventMap) {
		val := (*events)[EvtSearchFin]
		res := val.(MatchResult)
		if res.Merger.Length() != 2 {
			t.Errorf("Expected 2 cached matches, got %d", res.Merger.Length())
		}
		events.Clear()
	})
}

func TestMatcherRevisionChange(t *testing.T) {
	items := []string{"alpha", "beta", "gamma"}
	chunks, cache := buildChunks(items)

	eventBox := sync.NewEventBus()
	builder := makePatternBuilder()

	rev1 := pattern.Revision{Major: 1, Minor: 1}
	m := NewMatcher(cache, builder, true, false, eventBox, rev1, 2)
	go m.Loop()
	defer m.Stop()

	m.Reset(chunks, []rune("a"), false, true, true, rev1)
	eventBox.WaitFor(EvtSearchFin)
	eventBox.Wait(func(events *sync.EventMap) {
		val := (*events)[EvtSearchFin]
		res := val.(MatchResult)
		if res.Merger.Length() != 3 {
			t.Errorf("Expected 3 matches, got %d", res.Merger.Length())
		}
		events.Clear()
	})

	rev2 := pattern.Revision{Major: 2, Minor: 1}
	m.Reset(chunks, []rune("a"), false, true, true, rev2)
	eventBox.WaitFor(EvtSearchFin)
	eventBox.Wait(func(events *sync.EventMap) {
		val := (*events)[EvtSearchFin]
		res := val.(MatchResult)
		if res.Cancelled {
			t.Fatal("Should not be cancelled")
		}
		if res.Merger.Length() != 3 {
			t.Errorf("Expected 3 matches after rev change, got %d", res.Merger.Length())
		}
		events.Clear()
	})
}

func TestMatcherTacMode(t *testing.T) {
	items := []string{"aaa", "bbb", "aab"}
	chunks, cache := buildChunks(items)

	eventBox := sync.NewEventBus()
	builder := makePatternBuilder()

	m := NewMatcher(cache, builder, true, true, eventBox, pattern.Revision{}, 2)
	go m.Loop()
	defer m.Stop()

	m.Reset(chunks, []rune{}, false, true, true, pattern.Revision{})
	eventBox.WaitFor(EvtSearchFin)
	eventBox.Wait(func(events *sync.EventMap) {
		val := (*events)[EvtSearchFin]
		res := val.(MatchResult)
		if res.Merger.Length() != 3 {
			t.Errorf("Expected 3 items in tac mode, got %d", res.Merger.Length())
		}
		events.Clear()
	})
}

func TestMatcherStop(t *testing.T) {
	eventBox := sync.NewEventBus()
	builder := makePatternBuilder()
	cache := chunk.NewChunkCache()

	m := NewMatcher(cache, builder, true, false, eventBox, pattern.Revision{}, 2)

	done := make(chan struct{})
	go func() {
		m.Loop()
		close(done)
	}()

	m.Stop()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Loop did not exit after Stop")
	}
}

func TestMatcherCancelScanResumeScan(t *testing.T) {
	items := []string{"foo", "bar"}
	chunks, cache := buildChunks(items)

	eventBox := sync.NewEventBus()
	builder := makePatternBuilder()

	m := NewMatcher(cache, builder, true, false, eventBox, pattern.Revision{}, 2)
	go m.Loop()
	defer m.Stop()

	m.CancelScan()

	go func() {
		time.Sleep(50 * time.Millisecond)
		m.ResumeScan()
	}()

	m.Reset(chunks, []rune("foo"), false, true, true, pattern.Revision{})
	eventBox.WaitFor(EvtSearchFin)
	eventBox.Wait(func(events *sync.EventMap) {
		val := (*events)[EvtSearchFin]
		res := val.(MatchResult)
		if res.Merger.Length() != 1 {
			t.Errorf("Expected 1 match, got %d", res.Merger.Length())
		}
		events.Clear()
	})
}

func TestMatcherMultipleWorkers(t *testing.T) {
	n := 2048
	items := make([]string, n)
	for i := range n {
		if i%2 == 0 {
			items[i] = "abcdef"
		} else {
			items[i] = "ghijkl"
		}
	}
	chunks, cache := buildChunks(items)

	eventBox := sync.NewEventBus()
	builder := makePatternBuilder()

	m := NewMatcher(cache, builder, true, false, eventBox, pattern.Revision{}, 4)
	go m.Loop()
	defer m.Stop()

	m.Reset(chunks, []rune("abc"), false, true, true, pattern.Revision{})
	eventBox.WaitFor(EvtSearchFin)
	eventBox.Wait(func(events *sync.EventMap) {
		val := (*events)[EvtSearchFin]
		res := val.(MatchResult)
		if res.Merger.Length() != n/2 {
			t.Errorf("Expected %d matches, got %d", n/2, res.Merger.Length())
		}
		events.Clear()
	})
}

func TestMatcherSortToggle(t *testing.T) {
	items := []string{"xyz", "xyzabc", "abc"}
	chunks, cache := buildChunks(items)

	eventBox := sync.NewEventBus()
	builder := makePatternBuilder()

	m := NewMatcher(cache, builder, true, false, eventBox, pattern.Revision{}, 2)
	go m.Loop()
	defer m.Stop()

	m.Reset(chunks, []rune("abc"), false, true, true, pattern.Revision{})
	eventBox.WaitFor(EvtSearchFin)
	eventBox.Wait(func(events *sync.EventMap) {
		events.Clear()
	})

	m.Reset(chunks, []rune("abc"), false, true, false, pattern.Revision{})
	eventBox.WaitFor(EvtSearchFin)
	eventBox.Wait(func(events *sync.EventMap) {
		val := (*events)[EvtSearchFin]
		res := val.(MatchResult)
		if res.Cancelled {
			t.Fatal("Should not be cancelled")
		}
		events.Clear()
	})
}

func TestMatchResultCacheable(t *testing.T) {
	mr := MatchResult{Merger: nil}
	if mr.cacheable() {
		t.Error("nil merger should not be cacheable")
	}
}

func TestMatcherSequentialPatterns(t *testing.T) {
	items := []string{"alpha", "beta", "gamma", "delta"}
	chunks, cache := buildChunks(items)

	eventBox := sync.NewEventBus()
	builder := makePatternBuilder()

	m := NewMatcher(cache, builder, true, false, eventBox, pattern.Revision{}, 2)
	go m.Loop()
	defer m.Stop()

	m.Reset(chunks, []rune("a"), true, true, true, pattern.Revision{})
	eventBox.WaitFor(EvtSearchFin)
	eventBox.Wait(func(events *sync.EventMap) {
		val := (*events)[EvtSearchFin]
		res := val.(MatchResult)
		if res.Merger.Length() != 4 {
			t.Errorf("Expected 4 matches for 'a', got %d", res.Merger.Length())
		}
		events.Clear()
	})

	m.Reset(chunks, []rune("eta"), true, true, true, pattern.Revision{})
	eventBox.WaitFor(EvtSearchFin)
	eventBox.Wait(func(events *sync.EventMap) {
		val := (*events)[EvtSearchFin]
		res := val.(MatchResult)
		if res.Merger.Length() != 2 {
			t.Errorf("Expected 2 matches for 'eta', got %d", res.Merger.Length())
		}
		events.Clear()
	})
}
