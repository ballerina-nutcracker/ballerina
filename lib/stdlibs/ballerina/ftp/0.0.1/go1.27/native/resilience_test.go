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
)

func TestRetryWaitSaturates(t *testing.T) {
	cases := []struct {
		seconds float64
		want    time.Duration
	}{
		{0.0105, 10 * time.Millisecond},
		{1e10, math.MaxInt64},
		{math.Inf(1), math.MaxInt64},
		{math.NaN(), 0},
	}
	for _, c := range cases {
		if got := retryWait(c.seconds); got != c.want {
			t.Errorf("retryWait(%v) = %v, want %v", c.seconds, got, c.want)
		}
	}
}

func TestMillisSaturates(t *testing.T) {
	cases := []struct {
		seconds float64
		want    int64
	}{
		{2.5, 2500},
		{1e30, math.MaxInt64},
		{-1e30, math.MinInt64},
		{math.NaN(), 0},
	}
	for _, c := range cases {
		if got := millis(c.seconds); got != c.want {
			t.Errorf("millis(%v) = %v, want %v", c.seconds, got, c.want)
		}
	}
}
