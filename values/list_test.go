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

package values

import "testing"

// TestListRemoveAtReleasesVacatedSlot checks that RemoveAt nils out the slot
// it vacates instead of just shrinking the visible length, which would leave
// the backing array holding a strong reference to the removed value forever
// (as long as the list itself is reachable).
func TestListRemoveAtReleasesVacatedSlot(t *testing.T) {
	removedVal := newList(int64(99))
	l := newList(int64(1), int64(2), removedVal)
	full := l.elems[:cap(l.elems)]

	got := l.RemoveAt(2)
	if got != BalValue(removedVal) {
		t.Fatalf("RemoveAt returned %v, want %v", got, removedVal)
	}
	if l.Len() != 2 {
		t.Fatalf("Len() = %d, want 2", l.Len())
	}
	if last := full[len(full)-1]; last != nil {
		t.Fatalf("vacated slot still holds a reference: %v", last)
	}
}

// TestListClearReleasesAllSlots checks that Clear nils out every element
// before truncating, so previously-held values become unreachable through
// the list's backing array.
func TestListClearReleasesAllSlots(t *testing.T) {
	a := newList(int64(1))
	b := newList(int64(2))
	l := newList(a, b)
	full := l.elems[:cap(l.elems)]

	l.Clear()

	if l.Len() != 0 {
		t.Fatalf("Len() = %d, want 0", l.Len())
	}
	for i, v := range full {
		if v != nil {
			t.Fatalf("slot %d still holds a reference after Clear: %v", i, v)
		}
	}
}
