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

package native

import (
	"math"
	"testing"
	"time"

	"github.com/ballerina-nutcracker/ballerina/decimal"
)

// A saturated timeout can only be observed by waiting it out, so the conversion is pinned here.
func TestDecimalToDuration(t *testing.T) {
	for _, tc := range []struct {
		seconds string
		want    time.Duration
	}{
		{"1.5", 1500 * time.Millisecond},
		{"0", 0},
		{"1e-10", 1},
		{"1e-400", 1},
		{"1e10", math.MaxInt64},
		{"-1e10", math.MinInt64},
	} {
		d, err := decimal.FromString(tc.seconds)
		if err != nil {
			t.Fatalf("FromString(%q): %v", tc.seconds, err)
		}
		if got := decimalToDuration(d); got != tc.want {
			t.Errorf("decimalToDuration(%s) = %d, want %d", tc.seconds, got, tc.want)
		}
	}
}
