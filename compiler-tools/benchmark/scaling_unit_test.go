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
	"errors"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEvaluateScaling(t *testing.T) {
	cases := []struct {
		name        string
		startup     float64
		unit        float64
		scaled      float64
		wantRatio   float64
		wantVerdict scalingVerdict
	}{
		{name: "linear_passes", startup: 0.02, unit: 0.52, scaled: 2.02, wantRatio: 4, wantVerdict: verdictPass},
		{name: "quadratic_fails", startup: 0.02, unit: 0.52, scaled: 8.02, wantRatio: 16, wantVerdict: verdictFail},
		{name: "ratio_at_threshold_passes", startup: 0.02, unit: 0.52, scaled: 4.02, wantRatio: 8, wantVerdict: verdictPass},
		{name: "ratio_just_above_threshold_fails", startup: 0.02, unit: 0.52, scaled: 4.03, wantRatio: 8.02, wantVerdict: verdictFail},
		{name: "startup_is_subtracted_before_the_ratio", startup: 0.5, unit: 0.6, scaled: 1.4, wantRatio: 9, wantVerdict: verdictFail},
		{name: "work_below_floor_is_too_fast", startup: 0.02, unit: 0.069, scaled: 10, wantVerdict: verdictTooFast},
		{name: "work_at_floor_is_measured", startup: 0, unit: minScalingWork, scaled: 4 * minScalingWork, wantRatio: 4, wantVerdict: verdictPass},
		{name: "unit_faster_than_startup_is_too_fast", startup: 0.05, unit: 0.04, scaled: 0.2, wantVerdict: verdictTooFast},
		{name: "scaled_faster_than_unit_passes", startup: 0.02, unit: 1.02, scaled: 0.52, wantRatio: 0.5, wantVerdict: verdictPass},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ratio, verdict := evaluateScaling(tc.startup, tc.unit, tc.scaled, 8)
			if verdict != tc.wantVerdict {
				t.Fatalf("verdict = %v, want %v", verdict, tc.wantVerdict)
			}
			if math.Abs(ratio-tc.wantRatio) > 1e-6 {
				t.Fatalf("ratio = %v, want %v", ratio, tc.wantRatio)
			}
		})
	}
}

func scalingTestReport(cases ...scalingCase) *scalingReport {
	return &scalingReport{
		ref:      headRef,
		factor:   4,
		maxRatio: 8,
		startup:  benchResult{Mean: 0.02, Stddev: 0.001},
		cases:    cases,
	}
}

func scalingTestCase(label string, unit, scaled float64) scalingCase {
	return scalingCase{label: label, unit: benchResult{Mean: unit, Stddev: 0.01}, scaled: benchResult{Mean: scaled, Stddev: 0.02}}
}

func TestScalingSummaryListsEveryCaseWithVerdict(t *testing.T) {
	rep := scalingTestReport(
		scalingTestCase("linear.bal", 0.52, 2.02),
		scalingTestCase("quadratic.bal", 0.52, 8.02),
		scalingTestCase("tiny.bal", 0.03, 0.05),
	)
	summary := rep.renderSummary()
	for _, want := range []string{
		"| `linear.bal` | 520.0 ± 10.0 ms | 2020.0 ± 20.0 ms | 4.00 | pass |",
		"| `quadratic.bal` | 520.0 ± 10.0 ms | 8020.0 ± 20.0 ms | 16.00 | **fail** |",
		"| `tiny.bal` | 30.0 ± 10.0 ms | 50.0 ± 20.0 ms | n/a | too fast to measure |",
		"**1 of 3 case(s) scale super-linearly.**",
		"**1 case(s) too fast to measure**",
		"Startup (empty `main`, 20.0 ± 1.0 ms)",
		"ratio near 4 and quadratic work near 16",
	} {
		if !strings.Contains(summary, want) {
			t.Fatalf("summary does not contain %q:\n%s", want, summary)
		}
	}
	if strings.ContainsAny(summary, "<>") {
		t.Fatalf("summary contains angle brackets:\n%s", summary)
	}
}

func TestScalingSummaryOnPassingReport(t *testing.T) {
	summary := scalingTestReport(scalingTestCase("linear.bal", 0.52, 2.02)).renderSummary()
	if !strings.Contains(summary, "No case exceeds ratio 8.0.") {
		t.Fatalf("summary does not state that no case failed:\n%s", summary)
	}
	if strings.Contains(summary, "too fast to measure") {
		t.Fatalf("summary mentions too-fast cases when there are none:\n%s", summary)
	}
}

func TestFinishScaling(t *testing.T) {
	cases := []struct {
		name       string
		rep        *scalingReport
		wantFailed bool
	}{
		{name: "passing_report_succeeds", rep: scalingTestReport(scalingTestCase("linear.bal", 0.52, 2.02))},
		{name: "too_fast_case_does_not_fail", rep: scalingTestReport(scalingTestCase("tiny.bal", 0.03, 1))},
		{name: "failing_case_reports_exceeded", rep: scalingTestReport(scalingTestCase("quadratic.bal", 0.52, 8.02)), wantFailed: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			summaryPath := filepath.Join(t.TempDir(), "summary.md")
			err := finishScaling(tc.rep, summaryPath)
			if got := errors.Is(err, errScalingExceeded); got != tc.wantFailed {
				t.Fatalf("finishScaling() error = %v, want exceeded %v", err, tc.wantFailed)
			}
			if !tc.wantFailed && err != nil {
				t.Fatalf("finishScaling() returned error: %v", err)
			}
			if got := string(readFileOrFail(t, summaryPath)); got != tc.rep.renderSummary() {
				t.Fatalf("exported summary differs from rendered summary:\n%s", got)
			}
		})
	}
}

func TestFinishScalingWithoutSummaryPathWritesNothing(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := finishScaling(scalingTestReport(scalingTestCase("linear.bal", 0.52, 2.02)), ""); err != nil {
		t.Fatalf("finishScaling() returned error: %v", err)
	}
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("finishScaling() wrote files without a summary path: %v", entries)
	}
}

func TestMeasureScalingPassesScaleThroughEnvironment(t *testing.T) {
	skipHyperfineStubUnsupported(t)
	cases := []struct {
		exponent    string
		wantVerdict scalingVerdict
	}{
		{exponent: "1", wantVerdict: verdictPass},
		{exponent: "2", wantVerdict: verdictFail},
	}
	for _, tc := range cases {
		t.Run("exponent_"+tc.exponent, func(t *testing.T) {
			tmp := t.TempDir()
			writeScalingHyperfine(t, tmp)
			prependPath(t, tmp)
			t.Setenv("FAKE_SCALING_EXPONENT", tc.exponent)

			target := &benchmarkTarget{mode: singleFileMode, label: "case.bal", paths: []string{"case.bal"}}
			b := &benchmark{config: config{headRef: headRef, mode: scalingMode, runs: 1, factor: 4, maxRatio: 8}}
			rep, err := b.measureScaling("bal", target, t.TempDir())
			if err != nil {
				t.Fatalf("measureScaling() returned error: %v", err)
			}
			if rep.startup.Mean != 0.02 {
				t.Fatalf("startup mean = %v, want 0.02 (no %s set)", rep.startup.Mean, scaleEnvVar)
			}
			if len(rep.cases) != 1 || rep.cases[0].label != "case.bal" {
				t.Fatalf("unexpected cases: %+v", rep.cases)
			}
			if _, verdict := rep.evaluate(&rep.cases[0]); verdict != tc.wantVerdict {
				t.Fatalf("verdict = %v, want %v (case %+v)", verdict, tc.wantVerdict, rep.cases[0])
			}
		})
	}
}

func TestMeasureScalingReturnsHyperfineError(t *testing.T) {
	skipHyperfineStubUnsupported(t)
	tmp := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmp, "hyperfine"), []byte("#!/bin/sh\nexit 1\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	prependPath(t, tmp)

	target := &benchmarkTarget{mode: singleFileMode, label: "case.bal", paths: []string{"case.bal"}}
	b := &benchmark{config: config{headRef: headRef, mode: scalingMode, runs: 1, factor: 4, maxRatio: 8}}
	if _, err := b.measureScaling("bal", target, t.TempDir()); err == nil || !strings.Contains(err.Error(), "failed to run hyperfine") {
		t.Fatalf("measureScaling() error = %v", err)
	}
}

// writeScalingHyperfine installs a hyperfine stub whose mean time is
// 0.02 + 0.1 * BAL_SCALE^FAKE_SCALING_EXPONENT seconds, or 0.02 when
// BAL_SCALE is unset (the startup measurement).
func writeScalingHyperfine(t *testing.T, dir string) {
	t.Helper()
	script := `#!/bin/sh
out=""
while [ "$#" -gt 0 ]; do
  if [ "$1" = "--export-json" ]; then
    shift
    out="$1"
  fi
  shift
done
mean=$(awk -v s="${BAL_SCALE:-0}" -v e="$FAKE_SCALING_EXPONENT" 'BEGIN { if (s == 0) print 0.02; else print 0.02 + 0.1 * s ^ e }')
printf '{"results":[{"command":"c","mean":%s,"stddev":0.001,"median":%s}]}' "$mean" "$mean" > "$out"
`
	if err := os.WriteFile(filepath.Join(dir, "hyperfine"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
}

func TestConfigValidateScalingMode(t *testing.T) {
	target := writeTempBalFile(t)
	valid := config{headRef: headRef, target: target, mode: scalingMode, warmup: 1, runs: 3, factor: 4, maxRatio: 8}
	cases := []struct {
		name    string
		mutate  func(*config)
		wantErr string
	}{
		{name: "allows_missing_base_ref", mutate: func(*config) {}},
		{name: "requires_ref", mutate: func(c *config) { c.headRef = "" }, wantErr: "headRef is required"},
		{name: "rejects_factor_below_two", mutate: func(c *config) { c.factor = 1 }, wantErr: "factor must be at least 2"},
		{name: "rejects_max_ratio_of_one", mutate: func(c *config) { c.maxRatio = 1 }, wantErr: "max-ratio must be greater than 1"},
		{name: "rejects_html_export", mutate: func(c *config) { c.exportPath = "out.html" }, wantErr: "export-html is not supported in scaling mode"},
		{name: "rejects_zero_runs", mutate: func(c *config) { c.runs = 0 }, wantErr: "runs must be greater than zero"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := valid
			tc.mutate(&cfg)
			err := cfg.validate()
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("validate() returned error: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("validate() error = %v, want containing %q", err, tc.wantErr)
			}
		})
	}
}

func TestParseConfigScalingMode(t *testing.T) {
	originalArgs := os.Args
	t.Cleanup(func() { os.Args = originalArgs })

	os.Args = []string{"bal-bench", "-mode", "scaling", "-factor", "8", "-max-ratio", "20", headRef, "target.bal"}
	cfg, err := parseConfig()
	if err != nil {
		t.Fatalf("parseConfig() returned error: %v", err)
	}
	if cfg.baseRef != "" || cfg.headRef != headRef || cfg.target != "target.bal" {
		t.Fatalf("positional arguments = (%q, %q, %q), want (\"\", %q, \"target.bal\")", cfg.baseRef, cfg.headRef, cfg.target, headRef)
	}
	if cfg.factor != 8 || cfg.maxRatio != 20 {
		t.Fatalf("factor = %d, max-ratio = %v, want 8 and 20", cfg.factor, cfg.maxRatio)
	}

	os.Args = []string{"bal-bench", "-mode", "scaling", headRef}
	if _, err := parseConfig(); err == nil || !strings.Contains(err.Error(), "missing required arguments") {
		t.Fatalf("parseConfig() error = %v, want missing arguments", err)
	}
}

func TestParseConfigScalingDefaults(t *testing.T) {
	originalArgs := os.Args
	t.Cleanup(func() { os.Args = originalArgs })

	os.Args = []string{"bal-bench", "-mode", "scaling", headRef, "target.bal"}
	cfg, err := parseConfig()
	if err != nil {
		t.Fatalf("parseConfig() returned error: %v", err)
	}
	if cfg.factor != 4 || cfg.maxRatio != 8 {
		t.Fatalf("factor = %d, max-ratio = %v, want defaults 4 and 8", cfg.factor, cfg.maxRatio)
	}
}
