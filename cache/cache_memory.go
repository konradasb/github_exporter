package cache

import (
	"sync"
	"time"
)

// MemoryCache is an implemtation of Cache that will store items in an in-memory map
type MemoryCache struct {
	mu    *sync.RWMutex
	items map[string]*Item
}

// Get returns an item from cache
func (c *MemoryCache) Get(k string) *Item {
	c.mu.RLock()
	item, ok := c.items[k]
	c.mu.RUnlock()
	if !ok {
		return nil
	}
	return item
}

// Set stores an item to cache
func (c *MemoryCache) Set(k string, x interface{}) {
	c.mu.Lock()
	c.items[k] = &Item{Object: x, Age: time.Now().UnixNano()}
	c.mu.Unlock()
}

// Delete removes an item from cache
func (c *MemoryCache) Delete(k string) {
	c.mu.Lock()
	delete(c.items, k)
	c.mu.Unlock()
}

func (c *MemoryCache) Refresh(k string) {
	c.mu.Lock()
	item, ok := c.items[k]
	if ok {
		item.RefreshAge()
	}
	c.mu.Unlock()
}

// NewMemoryCache returns a new Cache that will store items in an in-memory map
func NewMemoryCache() *MemoryCache {
	c := &MemoryCache{
		items: make(map[string]*Item),
		mu:    &sync.RWMutex{},
	}

	return c
}
