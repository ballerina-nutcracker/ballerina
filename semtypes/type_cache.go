// Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
//
// WSO2 LLC. licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except
// in compliance with the License.
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

package semtypes

import "sync"

// SemTypeCache caches semtypes keyed by interned semtype values. SemTypeCache is safe for concurrent use.
type SemTypeCache struct {
	mu       sync.Mutex
	interner *SemtypeInterner
	values   map[InternHandle]SemType
}

func NewSemTypeCache() *SemTypeCache {
	return &SemTypeCache{
		interner: NewSemtypeInterner(),
		values:   make(map[InternHandle]SemType),
	}
}

func (c *SemTypeCache) GetOrBuild(key SemType, build func() SemType) SemType {
	c.mu.Lock()
	defer c.mu.Unlock()
	handle := c.interner.Intern(key)
	if value, ok := c.values[handle]; ok {
		return value
	}
	value := build()
	c.values[handle] = value
	return value
}
