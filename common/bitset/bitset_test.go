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

package bitset

// These are unit tests because capacity reuse and Lookup misses are not observable through corpus output.

import "testing"

func TestLookup(t *testing.T) {
	t.Parallel()
	c := NewContext[string]()
	want := c.Intern("a")
	if p, ok := c.Lookup("a"); !ok || p != want {
		t.Errorf("Lookup(a) = %v, %v, want %v, true", p, ok, want)
	}
	if _, ok := c.Lookup("b"); ok {
		t.Error("Lookup(b) found an unknown key")
	}
	if len(c.indices) != 1 {
		t.Errorf("len(indices) = %d after Lookup miss, want 1", len(c.indices))
	}
}

func TestAndThenGrowDoesNotResurrectBits(t *testing.T) {
	t.Parallel()
	s := setOf(1, 130)
	empty := BitSet{}
	s.And(&empty)
	s.Set(posOf(0))
	assertBits(t, "after regrow", &s, 0)
	other := setOf(129)
	s.Or(&other)
	assertBits(t, "after or", &s, 0, 129)
}

func TestCopyFromReusesCapacity(t *testing.T) {
	t.Parallel()
	src := setOf(1, 4, 70)
	dst := setOf(0, 190)
	backing := &dst.words[0]
	dst.CopyFrom(&src)
	assertBits(t, "reused", &dst, 1, 4, 70)
	if &dst.words[0] != backing {
		t.Error("CopyFrom did not reuse the receiver's capacity")
	}
}

func TestNewPreallocatesCapacity(t *testing.T) {
	t.Parallel()
	c := internN(130)
	s := c.New()
	if !s.IsEmpty() {
		t.Error("New returned a non-empty set")
	}
	if cap(s.words) < 3 {
		t.Errorf("cap(words) = %d, want >= 3", cap(s.words))
	}
}

func internN(n int) *Context[int] {
	c := NewContext[int]()
	for i := range n {
		c.Intern(i)
	}
	return c
}

func setOf(indices ...int) BitSet {
	var s BitSet
	for _, i := range indices {
		s.Set(posOf(i))
	}
	return s
}

func assertBits(t *testing.T, name string, s *BitSet, indices ...int) {
	t.Helper()
	want := make(map[int]bool, len(indices))
	for _, i := range indices {
		want[i] = true
	}
	for i := range len(s.words) * wordBits {
		if s.Has(posOf(i)) != want[i] {
			t.Errorf("%s: Has(%d) = %v, want %v", name, i, s.Has(posOf(i)), want[i])
		}
	}
	for i := range want {
		if !s.Has(posOf(i)) {
			t.Errorf("%s: missing bit %d", name, i)
		}
	}
}
