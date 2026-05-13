package chunk

import "sync"

const (
	ChunkSize     = 1024
	ChunkBitWords = (ChunkSize + 63) / 64
	QueryCacheMax = ChunkSize / 2
)

type ChunkBitmap [ChunkBitWords]uint64

type queryCache map[string]ChunkBitmap

type ChunkCache struct {
	mutex sync.Mutex
	cache map[*Chunk]*queryCache
}

func NewChunkCache() *ChunkCache {
	return &ChunkCache{cache: make(map[*Chunk]*queryCache)}
}

func (cc *ChunkCache) Clear() {
	cc.mutex.Lock()
	cc.cache = make(map[*Chunk]*queryCache)
	cc.mutex.Unlock()
}

func (cc *ChunkCache) Retire(chunks ...*Chunk) {
	cc.mutex.Lock()
	for _, c := range chunks {
		delete(cc.cache, c)
	}
	cc.mutex.Unlock()
}

func (cc *ChunkCache) Add(chunk *Chunk, key string, bitmap ChunkBitmap, matchCount int) {
	if len(key) == 0 || !chunk.IsFull() || matchCount > QueryCacheMax {
		return
	}

	cc.mutex.Lock()
	defer cc.mutex.Unlock()

	qc, ok := cc.cache[chunk]
	if !ok {
		cc.cache[chunk] = &queryCache{}
		qc = cc.cache[chunk]
	}
	(*qc)[key] = bitmap
}

func (cc *ChunkCache) Lookup(chunk *Chunk, key string) *ChunkBitmap {
	if len(key) == 0 || !chunk.IsFull() {
		return nil
	}

	cc.mutex.Lock()
	defer cc.mutex.Unlock()

	qc, ok := cc.cache[chunk]
	if ok {
		if bm, ok := (*qc)[key]; ok {
			return &bm
		}
	}
	return nil
}

func (cc *ChunkCache) Search(chunk *Chunk, key string) *ChunkBitmap {
	if len(key) == 0 || !chunk.IsFull() {
		return nil
	}

	cc.mutex.Lock()
	defer cc.mutex.Unlock()

	qc, ok := cc.cache[chunk]
	if !ok {
		return nil
	}

	for idx := 1; idx < len(key); idx++ {
		prefix := key[:len(key)-idx]
		suffix := key[idx:]
		for _, substr := range [2]string{prefix, suffix} {
			if bm, found := (*qc)[substr]; found {
				return &bm
			}
		}
	}
	return nil
}
