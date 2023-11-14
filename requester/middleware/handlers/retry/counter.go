package retry

import (
	"sync"
)

type Counter struct {
	cache map[uint64]uint64
	mu    sync.Mutex
}

func (c *Counter) Inc(id uint64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cache[id] += 1
}

func (c *Counter) Clear(id uint64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.cache, id)
}

func (c *Counter) Peek(id uint64) (uint64, bool) {
	v, ok := c.cache[id]
	return v, ok
}
