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

// These are unit tests because BitSet edge cases (word boundaries, operand length mismatch, capacity reuse) are not
// observable through corpus output.

import "testing"

func TestInternIsIdempotentAndDense(t *testing.T) {
	t.Parallel()
	c := NewContext[string]()
	a := c.Intern("a")
	b := c.Intern("b")
	if got := c.Intern("a"); got != a {
		t.Errorf("Intern(a) = %v, want %v", got, a)
	}
	if a != posOf(0) || b != posOf(1) {
		t.Errorf("positions = %v, %v, want %v, %v", a, b, posOf(0), posOf(1))
	}
	if c.Len() != 2 {
		t.Errorf("Len() = %d, want 2", c.Len())
	}
}

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
	if c.Len() != 1 {
		t.Errorf("Len() = %d after Lookup miss, want 1", c.Len())
	}
}

func TestPositionsAcrossWordBoundaries(t *testing.T) {
	t.Parallel()
	cases := []struct {
		index int
		want  Pos
	}{
		{0, Pos{word: 0, mask: 1}},
		{63, Pos{word: 0, mask: 1 << 63}},
		{64, Pos{word: 1, mask: 1}},
		{128, Pos{word: 2, mask: 1}},
	}
	c := internN(129)
	for _, tc := range cases {
		p, ok := c.Lookup(tc.index)
		if !ok || p != tc.want {
			t.Errorf("Lookup(%d) = %v, %v, want %v, true", tc.index, p, ok, tc.want)
		}
	}
	var s BitSet
	for _, tc := range cases {
		s.Set(tc.want)
	}
	for i := range 129 {
		p, _ := c.Lookup(i)
		want := i == 0 || i == 63 || i == 64 || i == 128
		if s.Has(p) != want {
			t.Errorf("Has(%d) = %v, want %v", i, s.Has(p), want)
		}
	}
}

func TestHasAndSetPastEnd(t *testing.T) {
	t.Parallel()
	var s BitSet
	p := posOf(200)
	if s.Has(p) {
		t.Error("Has past the end returned true")
	}
	s.Set(p)
	if !s.Has(p) {
		t.Error("Has after Set returned false")
	}
	if len(s.words) != p.word+1 {
		t.Errorf("len(words) = %d, want %d", len(s.words), p.word+1)
	}
}

func TestOr(t *testing.T) {
	t.Parallel()
	short := setOf(1)
	long := setOf(2, 130)
	short.Or(&long)
	assertBits(t, "short|long", &short, 1, 2, 130)

	long = setOf(2, 130)
	other := setOf(5)
	long.Or(&other)
	assertBits(t, "long|short", &long, 2, 5, 130)
}

func TestAnd(t *testing.T) {
	t.Parallel()
	long := setOf(1, 2, 130)
	short := setOf(1)
	long.And(&short)
	assertBits(t, "long&short", &long, 1)
	if len(long.words) != len(short.words) {
		t.Errorf("len(words) = %d, want %d", len(long.words), len(short.words))
	}

	short = setOf(1, 2)
	long = setOf(2, 3, 130)
	short.And(&long)
	assertBits(t, "short&long", &short, 2)
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

func TestCopyFrom(t *testing.T) {
	t.Parallel()
	src := setOf(1, 70)
	var dst BitSet
	dst.CopyFrom(&src)
	assertBits(t, "copy", &dst, 1, 70)
	dst.Set(posOf(3))
	src.Set(posOf(4))
	assertBits(t, "src after mutation", &src, 1, 4, 70)
	assertBits(t, "dst after mutation", &dst, 1, 3, 70)

	big := setOf(0, 190)
	backing := &big.words[0]
	big.CopyFrom(&src)
	assertBits(t, "reused", &big, 1, 4, 70)
	if &big.words[0] != backing {
		t.Error("CopyFrom did not reuse the receiver's capacity")
	}
}

func TestIsEmpty(t *testing.T) {
	t.Parallel()
	var s BitSet
	if !s.IsEmpty() {
		t.Error("new set is not empty")
	}
	s.Set(posOf(65))
	if s.IsEmpty() {
		t.Error("set with a bit is empty")
	}
	other := setOf(1)
	s.Or(&other)
	disjoint := setOf(2, 66)
	s.And(&disjoint)
	if !s.IsEmpty() {
		t.Error("set is not empty after clearing And")
	}
}

func TestNew(t *testing.T) {
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
