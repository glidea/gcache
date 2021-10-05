package cache

import (
	"sync"
	"time"
)

// Cache wrapper around an eviction implementation, and then adds some features.
type Cache struct {
	mu               sync.Mutex
	eviction         eviction
	timeout          time.Duration
	nBytes, maxBytes int64
}

func New(eviction eviction, timeout time.Duration, maxBytes int64) *Cache {
	return &Cache{
		eviction: eviction,
		timeout:  timeout,
		maxBytes: maxBytes,
	}
}

func (c *Cache) Add(k string, v []byte) (ok bool) {
	vwNew := &vWarp{
		v:           v,
		expiredTime: time.Now().Add(c.timeout),
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	var overBytes int64
	if vw, ok := c.eviction.get(k); ok {
		incrBytes := int64(len(vwNew.v) - len(vw.v))
		if overBytes, ok = c.updateBytes(incrBytes); !ok {
			return false
		}
		c.eviction.update(k, vwNew)
	} else {
		incrBytes := int64(len(k) + len(vwNew.v))
		if overBytes, ok = c.updateBytes(incrBytes); !ok {
			return false
		}
		c.eviction.insert(k, vwNew)
	}

	if overBytes > 0 {
		decrBytes := c.eviction.onFull(overBytes)
		c.nBytes -= decrBytes
	}
	return true
}

func (c *Cache) Get(k string) (v []byte, ok bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	vw, ok := c.eviction.get(k)
	if !ok {
		return []byte{}, false
	}
	if time.Now().After(vw.expiredTime) {
		c.eviction.remove(k)
		c.nBytes -= int64(len(k) + len(vw.v))
		return []byte{}, false
	}
	return vw.v, true
}

func (c *Cache) updateBytes(incrBytes int64) (overBytes int64, ok bool) {
	if incrBytes > c.maxBytes {
		return 0, false
	}
	c.nBytes += incrBytes
	return c.nBytes - c.maxBytes, true
}

// eviction algorithm interface. such as lru、LFU...
type eviction interface {
	insert(k string, vw *vWarp)
	remove(k string)
	update(k string, vw *vWarp)
	get(k string) (vw *vWarp, ok bool)
	onFull(overBytes int64) (decrBytes int64)
}

// vWarp as a value type of eviction
type vWarp struct {
	v           []byte
	expiredTime time.Time
}

// TODO 性能优化。time.Now()、并发度、更高效的LRU
