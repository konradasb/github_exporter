package cache

import (
	"time"
)

// Cache stores, retrieves and deletes cacheable items
type Cache interface {
	Get(k string) *Item
	Set(k string, x interface{})
	Delete(k string)
}

// Item is an object which is stored in Cache
type Item struct {
	Object interface{}
	Age    int64
}

// Get age returns the age of the cached object
func (i *Item) GetAge() time.Duration {
	return time.Since(time.Unix(0, i.Age))
}

func (i *Item) RefreshAge() {
	i.Age = time.Now().UnixNano()
}
