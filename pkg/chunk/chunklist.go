package chunk

import "sync"

type Chunk struct {
	Items [ChunkSize]Item
	Count int
}

type ItemBuilder func(*Item, []byte) bool

type ChunkList struct {
	chunks []*Chunk
	mutex  sync.Mutex
	trans  ItemBuilder
	cache  *ChunkCache
}

func NewChunkList(cache *ChunkCache, trans ItemBuilder) *ChunkList {
	return &ChunkList{
		chunks: []*Chunk{},
		trans:  trans,
		cache:  cache,
	}
}

func (c *Chunk) push(trans ItemBuilder, data []byte) bool {
	if trans(&c.Items[c.Count], data) {
		c.Count++
		return true
	}
	return false
}

func (c *Chunk) IsFull() bool {
	return c.Count == ChunkSize
}

func (c *Chunk) LastIndex(minValue int32) int32 {
	if c.Count == 0 {
		return minValue
	}
	return c.Items[c.Count-1].Index() + 1
}

func (cl *ChunkList) lastChunk() *Chunk {
	return cl.chunks[len(cl.chunks)-1]
}

func GetItems(chunks []*Chunk, n int) []Item {
	items := make([]Item, 0, n)
	for _, chunk := range chunks {
		for i := 0; i < chunk.Count && len(items) < n; i++ {
			items = append(items, chunk.Items[i])
		}
		if len(items) >= n {
			break
		}
	}
	return items
}

func CountItems(cs []*Chunk) int {
	if len(cs) == 0 {
		return 0
	}
	if len(cs) == 1 {
		return cs[0].Count
	}
	return cs[0].Count + ChunkSize*(len(cs)-2) + cs[len(cs)-1].Count
}

func (cl *ChunkList) Push(data []byte) bool {
	cl.mutex.Lock()

	if len(cl.chunks) == 0 || cl.lastChunk().IsFull() {
		cl.chunks = append(cl.chunks, &Chunk{})
	}

	ret := cl.lastChunk().push(cl.trans, data)
	cl.mutex.Unlock()
	return ret
}

func (cl *ChunkList) Clear() {
	cl.mutex.Lock()
	cl.chunks = nil
	cl.mutex.Unlock()
}

func (cl *ChunkList) ForEachItem(fn func(*Item), done func()) {
	cl.mutex.Lock()
	for _, c := range cl.chunks {
		for i := 0; i < c.Count; i++ {
			fn(&c.Items[i])
		}
	}
	if done != nil {
		done()
	}
	cl.mutex.Unlock()
}

func (cl *ChunkList) Snapshot(tail int) ([]*Chunk, int, bool) {
	cl.mutex.Lock()

	changed := false
	if tail > 0 && CountItems(cl.chunks) > tail {
		changed = true
		numChunks := 0
		for left, i := tail, len(cl.chunks)-1; left > 0 && i >= 0; i-- {
			numChunks++
			left -= cl.chunks[i].Count
		}

		ret := make([]*Chunk, numChunks)
		minIndex := len(cl.chunks) - numChunks
		cl.cache.Retire(cl.chunks[:minIndex]...)
		copy(ret, cl.chunks[minIndex:])

		for left, i := tail, len(ret)-1; i >= 0; i-- {
			c := ret[i]
			if c.Count > left {
				newChunk := *c
				newChunk.Count = left
				oldCount := c.Count
				for j := 0; j < left; j++ {
					newChunk.Items[j] = c.Items[oldCount-left+j]
				}
				ret[i] = &newChunk
				cl.cache.Retire(c)
				break
			}
			left -= c.Count
		}
		cl.chunks = ret
	}

	ret := make([]*Chunk, len(cl.chunks))
	copy(ret, cl.chunks)

	if cnt := len(ret); cnt > 0 {
		if tail > 0 && cnt > 1 {
			newChunk := *ret[0]
			ret[0] = &newChunk
		}
		newChunk := *ret[cnt-1]
		ret[cnt-1] = &newChunk
	}

	cl.mutex.Unlock()
	return ret, CountItems(ret), changed
}
