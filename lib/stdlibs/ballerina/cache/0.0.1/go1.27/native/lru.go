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
	"sync/atomic"

	"github.com/ballerina-nutcracker/ballerina/values"
)

type lruCache struct {
	mu             sync.Mutex
	cleaning       atomic.Bool
	capacity       int64
	evictionFactor float64
	// Front is the least recently used entry.
	order   *list.List
	entries map[string]*list.Element
}

type cacheItem struct {
	key   string
	entry *values.Map
	// expTime is in Unix nanoseconds; math.MaxInt64 means the entry never expires.
	expTime int64
}

func newLRUCache(capacity int64, evictionFactor float64) *lruCache {
	return &lruCache{
		capacity:       capacity,
		evictionFactor: evictionFactor,
		order:          list.New(),
		entries:        make(map[string]*list.Element),
	}
}

func (c *lruCache) put(key string, entry *values.Map, expTime int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if elem, ok := c.entries[key]; ok {
		item := elem.Value.(*cacheItem)
		item.entry = entry
		item.expTime = expTime
		c.order.MoveToBack(elem)
		return
	}
	if int64(len(c.entries)) >= c.capacity {
		c.evictTo(c.capacity - c.evictionCount())
	}
	c.entries[key] = c.order.PushBack(&cacheItem{key: key, entry: entry, expTime: expTime})
}

// evictionCount computes `capacity * evictionFactor` in 32-bit float arithmetic as jBallerina does,
// clamped to [1, capacity] so the cache never exceeds its capacity.
func (c *lruCache) evictionCount() int64 {
	product := float32(c.capacity) * float32(c.evictionFactor)
	return min(max(int64(math.Ceil(float64(product))), 1), c.capacity)
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
	if item.isExpired(now) {
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

// hasKey removes an expired entry and reports it as absent, as get does, without changing recency.
func (c *lruCache) hasKey(key string, now int64) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	elem, ok := c.entries[key]
	if !ok {
		return false
	}
	if elem.Value.(*cacheItem).isExpired(now) {
		c.removeElement(elem)
		return false
	}
	return true
}

func (c *lruCache) keys() []values.BalValue {
	c.mu.Lock()
	defer c.mu.Unlock()
	keys := make([]values.BalValue, 0, len(c.entries))
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

// cleanUp skips a run while another is still scanning, as jBallerina's cleanupInProgress flag does.
func (c *lruCache) cleanUp(now int64) {
	if !c.cleaning.CompareAndSwap(false, true) {
		return
	}
	defer c.cleaning.Store(false)
	c.mu.Lock()
	defer c.mu.Unlock()
	for elem := c.order.Front(); elem != nil; {
		next := elem.Next()
		if elem.Value.(*cacheItem).isExpired(now) {
			c.removeElement(elem)
		}
		elem = next
	}
}

func (c *lruCache) removeElement(elem *list.Element) {
	delete(c.entries, elem.Value.(*cacheItem).key)
	c.order.Remove(elem)
}

func (item *cacheItem) isExpired(now int64) bool {
	return item.expTime <= now
}
