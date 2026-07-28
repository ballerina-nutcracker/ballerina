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
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteFileRoundTrip(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "report.md")
	if err := writeFile(path, "# hello"); err != nil {
		t.Fatalf("writeFile: %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil || string(got) != "# hello" {
		t.Errorf("readback = %q, err = %v, want %q", got, err, "# hello")
	}
}

func TestWriteFileFailsForMissingDir(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "missing-dir", "report.md")
	if err := writeFile(path, "x"); err == nil {
		t.Fatal("expected an error writing to a nonexistent directory")
	}
}

func TestWriteResultJSONRoundTrip(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "result.json")
	want := result{Regressed: true, ThroughputBase: 1000, ThroughputHead: 800, ThroughputDelta: -20, ThresholdPct: 10}
	if err := writeResultJSON(path, want); err != nil {
		t.Fatalf("writeResultJSON: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading result.json: %v", err)
	}
	var got result
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got != want {
		t.Errorf("result.json roundtrip = %+v, want %+v", got, want)
	}
}

func TestWriteResultJSONFailsForMissingDir(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "missing-dir", "result.json")
	if err := writeResultJSON(path, result{}); err == nil {
		t.Fatal("expected an error writing to a nonexistent directory")
	}
}

func thrSamples(vals ...float64) []sample {
	out := make([]sample, len(vals))
	for i, v := range vals {
		out[i] = sample{wrkMetrics: wrkMetrics{throughput: v}}
	}
	return out
}

func TestGateFlagsSignificantRegression(t *testing.T) {
	t.Parallel()
	cfg := config{repeats: 3, duration: "10s", conns: 50, threshold: 10}
	base := thrSamples(100000, 101000, 99000) // median 100000
	head := thrSamples(85000, 86000, 84000)   // median 85000 -> -15%
	_, res := buildReport("base", "head", base, head, cfg)
	if !res.Regressed {
		t.Errorf("expected regressed=true for -15%% at threshold 10%%")
	}
	if res.ThroughputDelta != -15 {
		t.Errorf("throughputDelta = %v, want -15", res.ThroughputDelta)
	}
}

func TestGatePassesWithinThreshold(t *testing.T) {
	t.Parallel()
	cfg := config{repeats: 3, duration: "10s", conns: 50, threshold: 10}
	base := thrSamples(100000, 101000, 99000) // median 100000
	head := thrSamples(95000, 96000, 94000)   // median 95000 -> -5%
	_, res := buildReport("base", "head", base, head, cfg)
	if res.Regressed {
		t.Errorf("expected regressed=false for -5%% at threshold 10%%")
	}
	if res.ThroughputDelta != -5 {
		t.Errorf("throughputDelta = %v, want -5", res.ThroughputDelta)
	}
}

func TestGateFlagsRegressionAtExactThreshold(t *testing.T) {
	t.Parallel()
	cfg := config{repeats: 2, duration: "10s", conns: 50, threshold: 10}
	base := thrSamples(100000, 100000)
	head := thrSamples(90000, 90000) // exactly -10% at a 10% threshold
	_, res := buildReport("base", "head", base, head, cfg)
	if !res.Regressed {
		t.Errorf("expected regressed=true when the drop exactly equals the threshold")
	}
}

func TestBuildReportHeaderLabelsBaseAndHead(t *testing.T) {
	t.Parallel()
	cfg := config{repeats: 1, duration: "10s", conns: 50, threshold: 10}
	md, _ := buildReport("base-ref", "head-ref", thrSamples(100000), thrSamples(100000), cfg)
	if !strings.Contains(md, "`base-ref` (base)") {
		t.Errorf("expected the base column to be labeled, report:\n%s", md)
	}
	if !strings.Contains(md, "`head-ref` (head)") {
		t.Errorf("expected the head column to be labeled, report:\n%s", md)
	}
}

func TestGateIgnoresImprovement(t *testing.T) {
	t.Parallel()
	cfg := config{repeats: 2, duration: "10s", conns: 50, threshold: 10}
	base := thrSamples(100000, 100000)
	head := thrSamples(120000, 120000) // +20% — never a regression
	_, res := buildReport("base", "head", base, head, cfg)
	if res.Regressed {
		t.Errorf("an improvement must not be flagged as a regression")
	}
}

func TestGateErrorsOnEmptySamples(t *testing.T) {
	t.Parallel()
	cfg := config{repeats: 2, duration: "330s", conns: 50, threshold: 10}
	_, res := buildReport("base", "head", nil, thrSamples(100000, 100000), cfg)
	if !res.Error {
		t.Errorf("expected Error=true when the base group has no samples")
	}
	if res.Regressed {
		t.Errorf("must not report a regression on invalid samples")
	}
}

func TestGateErrorsOnZeroBaseline(t *testing.T) {
	t.Parallel()
	cfg := config{repeats: 2, duration: "330s", conns: 50, threshold: 10}
	_, res := buildReport("base", "head", thrSamples(0, 0), thrSamples(100000, 100000), cfg)
	if !res.Error {
		t.Errorf("expected Error=true when the base median throughput is 0")
	}
}

func TestComma(t *testing.T) {
	t.Parallel()
	cases := map[float64]string{0: "0", 999: "999", 1000: "1,000", 69303: "69,303", 1234567: "1,234,567"}
	for in, want := range cases {
		if got := comma(in); got != want {
			t.Errorf("comma(%v) = %q, want %q", in, got, want)
		}
	}
}
