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

package prettyprint

import (
	"slices"
	"testing"
)

func TestCompareFunctionPrintOrder(t *testing.T) {
	names := []string{
		"$start", "main", "$default$10", "$anonFunc$_2", "helper",
		"$default$2", "$gracefulStop", "$anonFunc$_10", "$default$1", "abc",
	}
	slices.SortStableFunc(names, CompareFunctionPrintOrder)
	expected := []string{
		"main", "helper", "abc",
		"$anonFunc$_2", "$anonFunc$_10",
		"$default$1", "$default$2", "$default$10",
		"$gracefulStop", "$start",
	}
	if !slices.Equal(names, expected) {
		t.Errorf("got %v, want %v", names, expected)
	}
}
