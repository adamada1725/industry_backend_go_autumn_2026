package main

import (
	"container/list"
	"sync"
)

type LRU[K comparable, V any] interface {
	Get(key K) (value V, ok bool)
	Set(key K, value V)
}
type entry[K comparable, V any] struct {
	key   K
	value V
}
type LRUCache[K comparable, V any] struct {
	capacity int
	mu       sync.Mutex
	ll       list.List
	items    map[K]*list.Element
}

func NewLRUCache[K comparable, V any](capacity int) *LRUCache[K, V] {
	return &LRUCache[K, V]{
		capacity: capacity,
		items:    map[K]*list.Element{},
	}
}
func (c *LRUCache[K, V]) Get(key K) (value V, ok bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	v, o := c.items[key]

	if !o {
		var zero V
		return zero, false
	}

	c.ll.MoveToFront(v)

	entry := v.Value.(entry[K, V])
	return entry.value, true
}
func (c *LRUCache[K, V]) Set(key K, value V) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.capacity <= 0 {
		return
	}

	if v, o := c.items[key]; o { // hit
		e := v.Value.(entry[K, V])
		e.value = value
		v.Value = e
		return
	}

	if c.ll.Len() >= c.capacity { // eviction
		last := c.ll.Back()
		e := last.Value.(entry[K, V])
		delete(c.items, e.key)
		c.ll.Remove(last)
	}

	entry := entry[K, V]{key: key, value: value}
	new := c.ll.PushFront(entry)
	c.items[key] = new
}
