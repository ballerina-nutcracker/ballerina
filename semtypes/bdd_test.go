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

import (
	"testing"
)

// TestBddDiff tests BDD operations
// Ported from SemTypeBddTest.java:bddTest()
func TestBddDiff(t *testing.T) {
	// Create two BDD atoms from different rec atoms
	b1 := bddAtom(new(createRecAtom(1, recAtomKindList)))
	b2 := bddAtom(new(createRecAtom(2, recAtomKindList)))

	// Intersect them
	b1and2 := bddIntersect(b1, b2)

	// Calculate difference: (b1 ∩ b2) - b1
	r := bddDiff(b1and2, b1)

	// Type assert to *bddAllOrNothing
	allOrNothing, ok := r.(*bddAllOrNothing)
	if !ok {
		t.Fatalf("expected *bddAllOrNothing, got %T", r)
	}

	// Assert that the result is not "all" (should be "nothing")
	if allOrNothing.IsAll() {
		t.Error("expected IsAll() to be false")
	}
}
