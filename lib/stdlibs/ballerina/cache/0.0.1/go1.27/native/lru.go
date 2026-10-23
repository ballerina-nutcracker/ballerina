// Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
//
// WSO2 LLC. licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except
// in compliance with the License.
//
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package native

import (
	"container/list"
	"math"
	"sync"

	"github.com/ballerina-nutcracker/ballerina/decimal"
	"github.com/ballerina-nutcracker/ballerina/values"
)

const expTimeField = "expTime"

type lruCache struct {
	mu             sync.Mutex
	capacity       int64
	evictionFactor float64
	// Front is the least recently used entry.
	order   *list.List
	entries map[string]*list.Element
}

type cacheItem struct {
	key   string
	entry *values.Map
}

func newLRUCache(capacity int64, evictionFactor float64) *lruCache {
	return &lruCache{
		capacity:       capacity,
		evictionFactor: evictionFactor,
		order:          list.New(),
		entries:        make(map[string]*list.Element),
	}
}

func (c *lruCache) put(key string, entry *values.Map) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if int64(len(c.entries)) >= c.capacity {
		c.evictTo(c.capacity - c.evictionCount())
	}
	if elem, ok := c.entries[key]; ok {
		elem.Value.(*cacheItem).entry = entry
		c.order.MoveToBack(elem)
		return
	}
	c.entries[key] = c.order.PushBack(&cacheItem{key: key, entry: entry})
}

// evictionCount matches jBallerina, which computes `capacity * evictionFactor` in 32-bit float arithmetic.
func (c *lruCache) evictionCount() int64 {
	product := float32(c.capacity) * float32(c.evictionFactor)
	return int64(math.Ceil(float64(product)))
}

func (c *lruCache) evictTo(size int64) {
	for int64(len(c.entries)) > size {
		c.removeElement(c.order.Front())
	}
}

func (c *lruCache) get(key string, now int64) *values.Map {
	c.mu.Lock()
	defer c.mu.Unlock()
	elem, ok := c.entries[key]
	if !ok {
		return nil
	}
	item := elem.Value.(*cacheItem)
	if isExpired(item.entry, now) {
		c.removeElement(elem)
		return nil
	}
	c.order.MoveToBack(elem)
	return item.entry
}

func (c *lruCache) remove(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if elem, ok := c.entries[key]; ok {
		c.removeElement(elem)
	}
}

func (c *lruCache) removeAll() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.order.Init()
	clear(c.entries)
}

func (c *lruCache) hasKey(key string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	_, ok := c.entries[key]
	return ok
}

func (c *lruCache) keys() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	keys := make([]string, 0, len(c.entries))
	for elem := c.order.Front(); elem != nil; elem = elem.Next() {
		keys = append(keys, elem.Value.(*cacheItem).key)
	}
	return keys
}

func (c *lruCache) size() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.entries)
}

func (c *lruCache) cleanUp(now int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for elem := c.order.Front(); elem != nil; {
		next := elem.Next()
		if isExpired(elem.Value.(*cacheItem).entry, now) {
			c.removeElement(elem)
		}
		elem = next
	}
}

func (c *lruCache) removeElement(elem *list.Element) {
	delete(c.entries, elem.Value.(*cacheItem).key)
	c.order.Remove(elem)
}

func isExpired(entry *values.Map, now int64) bool {
	v, _ := entry.Get(expTimeField)
	expTime := truncate(v.(*decimal.Decimal))
	return expTime != -1 && expTime <= now
}
