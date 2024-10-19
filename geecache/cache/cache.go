package cache

import (
	"geecache/internal/lru"
	"sync"
)

// 负责对lru的并发读写

type Cache struct {
	mu        sync.Mutex
	lru       *lru.Cache
	maxBytes  int64
	onRemoved func(key string, value lru.Value)
}

func (c *Cache) Put(key string, value []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.lru == nil {
		c.lru = lru.New(c.maxBytes, c.onRemoved)
	}
	c.lru.Put(key, byteView{value})
}

func (c *Cache) Get(key string) (value []byte, ok bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.lru == nil {
		return
	}

	if v, ok := c.lru.Get(key); ok {
		return v.(byteView).Bytes(), true
	}
	return value, false
}

func New(maxBytes int64, onRemoved func(key string, value lru.Value)) *Cache {
	return &Cache{
		maxBytes:  maxBytes,
		onRemoved: onRemoved,
	}
}
