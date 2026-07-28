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

package main

import (
	"math"
	"testing"
)

// A representative `wrk --latency` capture (no errors).
const sampleClean = `Running 10s test @ http://127.0.0.1:9090/hello
  4 threads and 50 connections
  Thread Stats   Avg      Stdev     Max   +/- Stdev
    Latency     1.76ms    1.84ms  30.40ms   88.00%
    Req/Sec    17.50k     2.00k    20.10k    70.00%
  Latency Distribution
     50%    1.50ms
     75%    2.00ms
     90%    3.00ms
     99%    8.01ms
  693033 requests in 10.00s, 89.10MB read
Requests/sec:  69303.29
Transfer/sec:      8.91MB`

// A capture with socket errors and non-2xx responses, and second-scale latency.
const sampleErrors = `Running 10s test @ http://127.0.0.1:9090/hello
  2 threads and 50 connections
  Thread Stats   Avg      Stdev     Max   +/- Stdev
    Latency     1.20s     0.50s     2.00s    60.00%
    Req/Sec     1.00k     0.50k     2.00k    55.00%
  Latency Distribution
     99%    1.90s
  10000 requests in 10.00s, 1.00MB read
  Socket errors: connect 0, read 5, write 0, timeout 3
  Non-2xx or 3xx responses: 12
Requests/sec:   1000.00
Transfer/sec:      0.10MB`

func approx(a, b float64) bool { return math.Abs(a-b) < 1e-6 }

func TestParseWrkClean(t *testing.T) {
	t.Parallel()
	m, err := parseWrk(sampleClean)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !approx(m.throughput, 69303.29) {
		t.Errorf("throughput = %v, want 69303.29", m.throughput)
	}
	if !approx(m.avgMs, 1.76) {
		t.Errorf("avgMs = %v, want 1.76", m.avgMs)
	}
	if !approx(m.stdevMs, 1.84) {
		t.Errorf("stdevMs = %v, want 1.84", m.stdevMs)
	}
	if !approx(m.maxMs, 30.40) {
		t.Errorf("maxMs = %v, want 30.40", m.maxMs)
	}
	if !approx(m.p99Ms, 8.01) {
		t.Errorf("p99Ms = %v, want 8.01", m.p99Ms)
	}
	if !approx(m.errPct, 0) {
		t.Errorf("errPct = %v, want 0", m.errPct)
	}
}

func TestParseWrkErrorsAndSecondsUnit(t *testing.T) {
	t.Parallel()
	m, err := parseWrk(sampleErrors)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !approx(m.avgMs, 1200) { // 1.20s -> 1200ms
		t.Errorf("avgMs = %v, want 1200", m.avgMs)
	}
	if !approx(m.p99Ms, 1900) { // 1.90s -> 1900ms
		t.Errorf("p99Ms = %v, want 1900", m.p99Ms)
	}
	// (5 read + 3 timeout socket errors) + 12 non-2xx = 20 of 10000 = 0.2%
	if !approx(m.errPct, 0.2) {
		t.Errorf("errPct = %v, want 0.2", m.errPct)
	}
}

func TestParseWrkNoThroughput(t *testing.T) {
	t.Parallel()
	if _, err := parseWrk("garbage output with no metrics"); err == nil {
		t.Fatal("expected an error when Requests/sec is absent")
	}
}

func TestToMsUnits(t *testing.T) {
	t.Parallel()
	cases := map[string]float64{
		"850.00us": 0.85,
		"1.76ms":   1.76,
		"1.20s":    1200,
		"2.00m":    120000,
	}
	for tok, want := range cases {
		got, err := toMs(tok)
		if err != nil {
			t.Errorf("toMs(%q) error: %v", tok, err)
			continue
		}
		if !approx(got, want) {
			t.Errorf("toMs(%q) = %v, want %v", tok, got, want)
		}
	}
}
