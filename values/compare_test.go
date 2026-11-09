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

import (
	"math"
	"testing"

	"github.com/ballerina-nutcracker/ballerina/semtypes"
)

func TestCompareASpecialValues(t *testing.T) {
	nan := math.NaN()

	assertCompareResult(t, compareA(int64(2), nil), CmpLT, "compareA(2,())")
	assertCompareResult(t, compareA(nil, int64(2)), CmpGT, "compareA((),2)")
	assertCompareResult(t, compareA(float64(2), nan), CmpLT, "compareA(2.0,NaN)")
	assertCompareResult(t, compareA(nan, float64(2)), CmpGT, "compareA(NaN,2.0)")
	assertCompareResult(t, compareA(nan, nil), CmpLT, "compareA(NaN,())")
	assertCompareResult(t, compareA(nan, nan), CmpEQ, "compareA(NaN,NaN)")
	assertCompareResult(t, compareA(nil, nil), CmpEQ, "compareA((),())")
}

func TestCompareDSpecialValues(t *testing.T) {
	nan := math.NaN()

	assertCompareResult(t, compareD(nil, int64(2)), CmpLT, "compareD((),2)")
	assertCompareResult(t, compareD(int64(2), nil), CmpGT, "compareD(2,())")
	assertCompareResult(t, compareD(nan, float64(2)), CmpLT, "compareD(NaN,2.0)")
	assertCompareResult(t, compareD(float64(2), nan), CmpGT, "compareD(2.0,NaN)")
	assertCompareResult(t, compareD(nil, nan), CmpLT, "compareD((),NaN)")
	assertCompareResult(t, compareD(nan, nan), CmpEQ, "compareD(NaN,NaN)")
	assertCompareResult(t, compareD(nil, nil), CmpEQ, "compareD((),())")
}

func TestCompareAAndDLists(t *testing.T) {
	nan := math.NaN()
	short := newList(int64(1))
	plain := newList(int64(1), float64(2))
	withNaN := newList(int64(1), nan)
	withNil := newList(int64(1), nil)

	assertCompareResult(t, compareA(short, plain), CmpLT, "compareA([1],[1,2.0])")
	assertCompareResult(t, compareA(plain, withNaN), CmpLT, "compareA([1,2.0],[1,NaN])")
	assertCompareResult(t, compareA(withNaN, withNil), CmpLT, "compareA([1,NaN],[1,()])")

	assertCompareResult(t, compareD(withNil, withNaN), CmpLT, "compareD([1,()],[1,NaN])")
	assertCompareResult(t, compareD(withNaN, plain), CmpLT, "compareD([1,NaN],[1,2.0])")
	assertCompareResult(t, compareD(short, plain), CmpLT, "compareD([1],[1,2.0])")
}

func TestCompareK(t *testing.T) {
	nan := math.NaN()

	assertCompareResult(t, CompareK(float64(2), nil, true), CmpLT, "CompareK(2.0,(),ascending)")
	assertCompareResult(t, CompareK(float64(2), nil, false), CmpLT, "CompareK(2.0,(),descending)")
	assertCompareResult(t, CompareK(nan, nil, true), CmpLT, "CompareK(NaN,(),ascending)")
	assertCompareResult(t, CompareK(nan, float64(2), false), CmpGT, "CompareK(NaN,2.0,descending)")
}

func newList(values ...BalValue) *List {
	return NewList(semtypes.List, &semtypes.ListAtomicInner, false, nil, 0, values)
}

func assertCompareResult(t *testing.T, got CompareResult, want CompareResult, label string) {
	t.Helper()
	if got != want {
		t.Fatalf("%s: got %v, want %v", label, got, want)
	}
}
