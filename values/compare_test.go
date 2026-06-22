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
	"ballerina-lang-go/semtypes"
	"math"
	"testing"
)

func TestCompareASpecialValues(t *testing.T) {
	nan := math.NaN()

	assertCompareResult(t, CompareA(int64(2), nil), CmpLT, "CompareA(2,())")
	assertCompareResult(t, CompareA(nil, int64(2)), CmpGT, "CompareA((),2)")
	assertCompareResult(t, CompareA(float64(2), nan), CmpLT, "CompareA(2.0,NaN)")
	assertCompareResult(t, CompareA(nan, float64(2)), CmpGT, "CompareA(NaN,2.0)")
	assertCompareResult(t, CompareA(nan, nil), CmpLT, "CompareA(NaN,())")
	assertCompareResult(t, CompareA(nan, nan), CmpEQ, "CompareA(NaN,NaN)")
	assertCompareResult(t, CompareA(nil, nil), CmpEQ, "CompareA((),())")
}

func TestCompareDSpecialValues(t *testing.T) {
	nan := math.NaN()

	assertCompareResult(t, CompareD(nil, int64(2)), CmpLT, "CompareD((),2)")
	assertCompareResult(t, CompareD(int64(2), nil), CmpGT, "CompareD(2,())")
	assertCompareResult(t, CompareD(nan, float64(2)), CmpLT, "CompareD(NaN,2.0)")
	assertCompareResult(t, CompareD(float64(2), nan), CmpGT, "CompareD(2.0,NaN)")
	assertCompareResult(t, CompareD(nil, nan), CmpLT, "CompareD((),NaN)")
	assertCompareResult(t, CompareD(nan, nan), CmpEQ, "CompareD(NaN,NaN)")
	assertCompareResult(t, CompareD(nil, nil), CmpEQ, "CompareD((),())")
}

func TestCompareAAndDLists(t *testing.T) {
	nan := math.NaN()
	short := newList(int64(1))
	plain := newList(int64(1), float64(2))
	withNaN := newList(int64(1), nan)
	withNil := newList(int64(1), nil)

	assertCompareResult(t, CompareA(short, plain), CmpLT, "CompareA([1],[1,2.0])")
	assertCompareResult(t, CompareA(plain, withNaN), CmpLT, "CompareA([1,2.0],[1,NaN])")
	assertCompareResult(t, CompareA(withNaN, withNil), CmpLT, "CompareA([1,NaN],[1,()])")

	assertCompareResult(t, CompareD(withNil, withNaN), CmpLT, "CompareD([1,()],[1,NaN])")
	assertCompareResult(t, CompareD(withNaN, plain), CmpLT, "CompareD([1,NaN],[1,2.0])")
	assertCompareResult(t, CompareD(short, plain), CmpLT, "CompareD([1],[1,2.0])")
}

func TestCompareK(t *testing.T) {
	nan := math.NaN()

	assertCompareResult(t, CompareK(float64(2), nil, true), CmpLT, "CompareK(2.0,(),ascending)")
	assertCompareResult(t, CompareK(float64(2), nil, false), CmpLT, "CompareK(2.0,(),descending)")
	assertCompareResult(t, CompareK(nan, nil, true), CmpLT, "CompareK(NaN,(),ascending)")
	assertCompareResult(t, CompareK(nan, float64(2), false), CmpGT, "CompareK(NaN,2.0,descending)")
}

func newList(values ...BalValue) *List {
	return NewList(semtypes.LIST, &semtypes.LIST_ATOMIC_INNER, false, nil, 0, values)
}

func assertCompareResult(t *testing.T, got CompareResult, want CompareResult, label string) {
	t.Helper()
	if got != want {
		t.Fatalf("%s: got %v, want %v", label, got, want)
	}
}
