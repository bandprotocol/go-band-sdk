package retry

import (
	"sync"
)

type Counter struct {
	cache sync.Map
}

func (c *Counter) Inc(id uint64) {
	v, ok := c.cache.Load(id)
	if !ok {
		c.cache.Store(id, uint64(1))
	} else {
		c.cache.Store(id, v.(uint64)+1)
	}
}

func (c *Counter) Clear(id uint64) {
	c.cache.Delete(id)
}

func (c *Counter) Peek(id uint64) (uint64, bool) {
	v, ok := c.cache.Load(id)
	if !ok {
		v = 0
	}
	return v.(uint64), ok
}
