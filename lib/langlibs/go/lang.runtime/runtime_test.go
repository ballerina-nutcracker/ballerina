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

// TestSecondsToSleepDuration is a unit test because the clamped ~292-year
// sleep can't run to completion in a corpus test.
func TestSecondsToSleepDuration(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		seconds float64
		want    time.Duration
	}{
		{"typical positive value", 1.5, 1500 * time.Millisecond},
		{"zero is a no-op", 0, 0},
		{"negative is a no-op", -1, 0},
		{"positive infinity clamps to max duration", math.Inf(1), math.MaxInt64},
		{"a value far outside decimal's overlap with float64 clamps to max duration", 1e300, math.MaxInt64},
		{"a value just under int64 nanosecond range converts exactly", 1000, 1000 * time.Second},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := secondsToSleepDuration(tc.seconds)
			if got != tc.want {
				t.Errorf("secondsToSleepDuration(%v) = %v, want %v", tc.seconds, got, tc.want)
			}
		})
	}
}

// TestSleepDeadline guards the clamped max duration from wrapping the deadline
// negative once added to a positive monotonic clock reading.
func TestSleepDeadline(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		now  time.Duration
		dur  time.Duration
		want time.Duration
	}{
		{"typical addition", 5 * time.Second, time.Second, 6 * time.Second},
		{"zero duration", 5 * time.Second, 0, 5 * time.Second},
		{"max duration saturates", 5 * time.Second, math.MaxInt64, math.MaxInt64},
		{"max duration at clock zero", 0, math.MaxInt64, math.MaxInt64},
		{"sum exactly at the limit", math.MaxInt64 - 1, 1, math.MaxInt64},
		{"sum one past the limit saturates", math.MaxInt64 - 1, 2, math.MaxInt64},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := sleepDeadline(tc.now, tc.dur); got != tc.want {
				t.Errorf("sleepDeadline(%v, %v) = %v, want %v", tc.now, tc.dur, got, tc.want)
			}
		})
	}
}
