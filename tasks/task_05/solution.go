package main

type Cache[K comparable, V any] struct {
	capacity int
	items    map[K]V
}

func NewCache[K comparable, V any](capacity int) *Cache[K, V] {
	cache := Cache[K, V]{
		capacity,
		make(map[K]V),
	}
	return &cache
}
func (c *Cache[K, V]) Get(k K) (v V, ok bool) {
	val, ok := c.items[k]
	return val, ok
}
func (c *Cache[K, V]) Set(k K, v V) bool {
	_, has := c.items[k]
	if len(c.items) >= c.capacity && !has {
		return false
	}
	c.items[k] = v
	return true
}
