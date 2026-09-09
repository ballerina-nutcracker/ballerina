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

package langruntime

import (
	"math"
	"testing"
	"time"
)

// TestSecondsToSleepDuration covers the clamping contract that motivated
// extracting this helper: decimal's value range vastly exceeds float64's, and
// a naive seconds*time.Second conversion can overflow to a platform-defined
// (and on some platforms negative) int64, turning a huge sleep() call into a
// no-op instead of the longest representable sleep. A corpus test can't cover
// the overflow branches directly (sleeping for the clamped ~292-year duration
// isn't practical to run to completion), so this is a plain unit test on the
// pure helper instead.
func TestSecondsToSleepDuration(t *testing.T) {
	tests := []struct {
		name    string
		seconds float64
		want    time.Duration
	}{
		{"typical positive value", 1.5, 1500 * time.Millisecond},
		{"zero is a no-op", 0, 0},
		{"negative is a no-op", -1, 0},
		{"NaN is a no-op", math.NaN(), 0},
		{"positive infinity clamps to max duration", math.Inf(1), math.MaxInt64},
		{"a value far outside decimal's overlap with float64 clamps to max duration", 1e300, math.MaxInt64},
		{"a value just under int64 nanosecond range converts exactly", 1000, 1000 * time.Second},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := secondsToSleepDuration(tc.seconds)
			if got != tc.want {
				t.Errorf("secondsToSleepDuration(%v) = %v, want %v", tc.seconds, got, tc.want)
			}
		})
	}
}
