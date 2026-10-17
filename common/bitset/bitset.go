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

// Package bitset provides an unbounded bit set whose positions are assigned by a Context.
package bitset

const wordBits = 64

// Pos is a bit position: word index and single-bit mask within the word.
type Pos struct {
	word int
	mask uint64
}

// Context assigns dense bit positions to keys. Not safe for concurrent use.
type Context[K comparable] struct {
	indices map[K]int
}

func NewContext[K comparable]() *Context[K] {
	return &Context[K]{indices: make(map[K]int)}
}

// Intern returns the position of k, assigning the next free index if k is new.
func (c *Context[K]) Intern(k K) Pos {
	i, ok := c.indices[k]
	if !ok {
		i = len(c.indices)
		c.indices[k] = i
	}
	return posOf(i)
}

// Lookup returns the position of k if it has been interned. It never interns.
func (c *Context[K]) Lookup(k K) (Pos, bool) {
	i, ok := c.indices[k]
	if !ok {
		return Pos{}, false
	}
	return posOf(i), true
}

func (c *Context[K]) Len() int {
	return len(c.indices)
}

// New returns an empty BitSet with capacity for every position interned so far.
func (c *Context[K]) New() BitSet {
	return BitSet{words: make([]uint64, 0, (c.Len()+wordBits-1)/wordBits)}
}

func posOf(i int) Pos {
	return Pos{word: i / wordBits, mask: 1 << (i % wordBits)}
}

// BitSet is an unbounded set of positions; missing words read as zero.
// Each bit set is meaningful only within the context from which it was created.
// We are keeping these structs separately in order to allow creating any number of
// bitsets from a given context. BitSet itself does not track the Context from which
// it was created, leaving that responsibility to the user (who can implement this more
// efficiently than us).
type BitSet struct {
	words []uint64
}

func (s *BitSet) Has(p Pos) bool {
	return p.word < len(s.words) && s.words[p.word]&p.mask != 0
}

func (s *BitSet) Set(p Pos) {
	s.grow(p.word + 1)
	s.words[p.word] |= p.mask
}

func (s *BitSet) Or(o *BitSet) {
	s.grow(len(o.words))
	for i, w := range o.words {
		s.words[i] |= w
	}
}

// And intersects s with o. Words missing from o count as zero.
func (s *BitSet) And(o *BitSet) {
	n := min(len(s.words), len(o.words))
	for i := range n {
		s.words[i] &= o.words[i]
	}
	s.words = s.words[:n]
}

// CopyFrom makes s equal to o, reusing the capacity of s. The result shares no memory with o.
func (s *BitSet) CopyFrom(o *BitSet) {
	s.words = append(s.words[:0], o.words...)
}

func (s *BitSet) IsEmpty() bool {
	for _, w := range s.words {
		if w != 0 {
			return false
		}
	}
	return true
}

func (s *BitSet) grow(n int) {
	if n <= len(s.words) {
		return
	}
	if n <= cap(s.words) {
		old := len(s.words)
		s.words = s.words[:n]
		clear(s.words[old:])
		return
	}
	s.words = append(s.words, make([]uint64, n-len(s.words))...)
}
