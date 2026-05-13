package chunk

import "testing"

func TestChunkCache(t *testing.T) {
	cache := NewChunkCache()
	chunk1p := &Chunk{}
	chunk2p := &Chunk{Count: ChunkSize}
	bm1 := ChunkBitmap{1}
	bm2 := ChunkBitmap{1, 2}
	cache.Add(chunk1p, "foo", bm1, 1)
	cache.Add(chunk2p, "foo", bm1, 1)
	cache.Add(chunk2p, "bar", bm2, 2)

	{
		cached := cache.Lookup(chunk1p, "foo")
		if cached != nil {
			t.Error("caching disabled for non-full chunks", cached)
		}
	}
	{
		cached := cache.Lookup(chunk2p, "foo")
		if cached == nil || cached[0] != 1 {
			t.Error("expected bitmap cached", cached)
		}
	}
	{
		cached := cache.Lookup(chunk2p, "bar")
		if cached == nil || cached[1] != 2 {
			t.Error("expected bitmap cached", cached)
		}
	}
	{
		cached := cache.Lookup(chunk1p, "foobar")
		if cached != nil {
			t.Error("expected nil cached", cached)
		}
	}
}

func TestChunkCacheSearch(t *testing.T) {
	cache := NewChunkCache()
	c := &Chunk{Count: ChunkSize}
	bm := ChunkBitmap{42}
	cache.Add(c, "foob", bm, 1)

	found := cache.Search(c, "foobar")
	if found == nil || found[0] != 42 {
		t.Error("expected to find prefix match")
	}

	notFound := cache.Search(c, "x")
	if notFound != nil {
		t.Error("expected nil for unrelated key")
	}
}

func TestChunkCacheEmptyKey(t *testing.T) {
	cache := NewChunkCache()
	c := &Chunk{Count: ChunkSize}
	bm := ChunkBitmap{1}
	cache.Add(c, "", bm, 1)
	if cache.Lookup(c, "") != nil {
		t.Error("empty key should not be cached")
	}
}

func TestChunkCacheHighMatchCount(t *testing.T) {
	cache := NewChunkCache()
	c := &Chunk{Count: ChunkSize}
	bm := ChunkBitmap{1}
	cache.Add(c, "foo", bm, QueryCacheMax+1)
	if cache.Lookup(c, "foo") != nil {
		t.Error("high match count should not be cached")
	}
}

func TestChunkCacheClearAndRetire(t *testing.T) {
	cache := NewChunkCache()
	c := &Chunk{Count: ChunkSize}
	bm := ChunkBitmap{1}
	cache.Add(c, "foo", bm, 1)

	cache.Retire(c)
	if cache.Lookup(c, "foo") != nil {
		t.Error("should be nil after retire")
	}

	cache.Add(c, "bar", bm, 1)
	cache.Clear()
	if cache.Lookup(c, "bar") != nil {
		t.Error("should be nil after clear")
	}
}
