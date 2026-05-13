package chunk

import (
	"fmt"
	"testing"

	"github.com/fzf/finder/pkg/charutil"
)

func TestChunkList(t *testing.T) {
	cl := NewChunkList(NewChunkCache(), func(item *Item, s []byte) bool {
		item.Text = charutil.ToChars(s)
		return true
	})

	snapshot, count, _ := cl.Snapshot(0)
	if len(snapshot) > 0 || count > 0 {
		t.Error("snapshot should be empty")
	}

	cl.Push([]byte("hello"))
	cl.Push([]byte("world"))

	if len(snapshot) > 0 {
		t.Error("previous snapshot should not have changed")
	}

	snapshot, count, _ = cl.Snapshot(0)
	if len(snapshot) != 1 || count != 2 {
		t.Error("snapshot should have 1 chunk, 2 items")
	}

	chunk1 := snapshot[0]
	if chunk1.Count != 2 {
		t.Error("chunk should have 2 items")
	}
	if chunk1.Items[0].Text.ToString() != "hello" ||
		chunk1.Items[1].Text.ToString() != "world" {
		t.Error("invalid data")
	}
	if chunk1.IsFull() {
		t.Error("chunk should not be full")
	}

	for i := range ChunkSize * 2 {
		cl.Push(fmt.Appendf(nil, "item %d", i))
	}

	if len(snapshot) != 1 {
		t.Error("previous snapshot should stay the same")
	}

	snapshot, count, _ = cl.Snapshot(0)
	if len(snapshot) != 3 || !snapshot[0].IsFull() ||
		!snapshot[1].IsFull() || snapshot[2].IsFull() || count != ChunkSize*2+2 {
		t.Error("expected two full chunks and one partial")
	}
	if snapshot[2].Count != 2 {
		t.Error("unexpected item count in last chunk")
	}

	cl.Push([]byte("hello"))
	cl.Push([]byte("world"))

	lastChunkCount := snapshot[len(snapshot)-1].Count
	if lastChunkCount != 2 {
		t.Error("snapshot immutability violated:", lastChunkCount)
	}
}

func TestChunkListTail(t *testing.T) {
	cl := NewChunkList(NewChunkCache(), func(item *Item, s []byte) bool {
		item.Text = charutil.ToChars(s)
		return true
	})
	total := ChunkSize*2 + ChunkSize/2
	for i := range total {
		cl.Push(fmt.Appendf(nil, "item %d", i))
	}

	snapshot, count, changed := cl.Snapshot(0)
	assertCount := func(expected int, shouldChange bool) {
		if count != expected || CountItems(snapshot) != expected {
			t.Errorf("unexpected count: %d (expected: %d)", count, expected)
		}
		if changed != shouldChange {
			t.Error("unexpected change status")
		}
	}
	assertCount(total, false)

	tail := ChunkSize + ChunkSize/2
	snapshot, count, changed = cl.Snapshot(tail)
	assertCount(tail, true)

	snapshot, count, changed = cl.Snapshot(tail)
	assertCount(tail, false)

	snapshot, count, changed = cl.Snapshot(0)
	assertCount(tail, false)

	tail = ChunkSize / 2
	snapshot, count, changed = cl.Snapshot(tail)
	assertCount(tail, true)
}

func TestGetItems(t *testing.T) {
	cl := NewChunkList(NewChunkCache(), func(item *Item, s []byte) bool {
		item.Text = charutil.ToChars(s)
		return true
	})
	for i := 0; i < 5; i++ {
		cl.Push(fmt.Appendf(nil, "item %d", i))
	}
	snapshot, _, _ := cl.Snapshot(0)
	items := GetItems(snapshot, 3)
	if len(items) != 3 {
		t.Errorf("expected 3 items, got %d", len(items))
	}
	if items[0].Text.ToString() != "item 0" {
		t.Errorf("expected 'item 0', got %q", items[0].Text.ToString())
	}
}

func TestChunkListClear(t *testing.T) {
	cl := NewChunkList(NewChunkCache(), func(item *Item, s []byte) bool {
		item.Text = charutil.ToChars(s)
		return true
	})
	cl.Push([]byte("hello"))
	cl.Clear()
	snapshot, count, _ := cl.Snapshot(0)
	if len(snapshot) != 0 || count != 0 {
		t.Error("should be empty after clear")
	}
}

func TestForEachItem(t *testing.T) {
	cl := NewChunkList(NewChunkCache(), func(item *Item, s []byte) bool {
		item.Text = charutil.ToChars(s)
		return true
	})
	cl.Push([]byte("a"))
	cl.Push([]byte("b"))
	cl.Push([]byte("c"))

	count := 0
	doneRan := false
	cl.ForEachItem(func(item *Item) {
		count++
	}, func() {
		doneRan = true
	})
	if count != 3 {
		t.Errorf("expected 3, got %d", count)
	}
	if !doneRan {
		t.Error("done callback should have run")
	}
}

func TestCountItems(t *testing.T) {
	if CountItems(nil) != 0 {
		t.Error("nil should return 0")
	}
	if CountItems([]*Chunk{}) != 0 {
		t.Error("empty should return 0")
	}
	c := &Chunk{Count: 5}
	if CountItems([]*Chunk{c}) != 5 {
		t.Error("single chunk")
	}
}
