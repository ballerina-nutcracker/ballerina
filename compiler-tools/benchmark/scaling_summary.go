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
	"fmt"
	"strings"
)

// minScalingWork is the smallest startup-adjusted BAL_SCALE=1 time, in
// seconds, whose ratio is trusted; below it timer noise dominates the ratio.
const minScalingWork = 0.05

type scalingVerdict int

const (
	verdictPass scalingVerdict = iota
	verdictFail
	verdictTooFast
)

type (
	scalingCase struct {
		label  string
		unit   benchResult
		scaled benchResult
	}
	scalingReport struct {
		ref      string
		factor   int
		maxRatio float64
		startup  benchResult
		cases    []scalingCase
	}
)

// evaluateScaling compares the work done above startup at both scales. The
// ratio is zero for verdictTooFast, where it would not be meaningful.
func evaluateScaling(startup, unit, scaled, maxRatio float64) (float64, scalingVerdict) {
	unitWork := unit - startup
	if unitWork < minScalingWork {
		return 0, verdictTooFast
	}
	ratio := (scaled - startup) / unitWork
	if ratio > maxRatio {
		return ratio, verdictFail
	}
	return ratio, verdictPass
}

func (r *scalingReport) evaluate(sc *scalingCase) (float64, scalingVerdict) {
	return evaluateScaling(r.startup.Mean, sc.unit.Mean, sc.scaled.Mean, r.maxRatio)
}

func (r *scalingReport) count(verdict scalingVerdict) int {
	n := 0
	for i := range r.cases {
		if _, v := r.evaluate(&r.cases[i]); v == verdict {
			n++
		}
	}
	return n
}

func (r *scalingReport) renderSummary() string {
	lines := []string{
		"### Scaling check",
		"",
		fmt.Sprintf("Revision `%s`, %s=1 vs %s=%d. A case fails when its startup-adjusted time ratio exceeds %.1f. "+
			"Startup (empty `main`, %s) is subtracted from both times before the ratio is taken.",
			sanitizeLabel(r.ref), scaleEnvVar, scaleEnvVar, r.factor, r.maxRatio, formatSeconds(r.startup)),
		"",
		fmt.Sprintf("| Case | %s=1 | %s=%d | Ratio | Verdict |", scaleEnvVar, scaleEnvVar, r.factor),
		"|---|---:|---:|---:|---|",
	}
	for i := range r.cases {
		lines = append(lines, r.caseRow(&r.cases[i]))
	}
	lines = append(lines, "", r.headline())
	if tooFast := r.count(verdictTooFast); tooFast > 0 {
		lines = append(lines, "", fmt.Sprintf("**%d case(s) too fast to measure**: under %.0f ms of work at %s=1, so the ratio is not checked. "+
			"Raise the workload's base size.", tooFast, minScalingWork*1000, scaleEnvVar))
	}
	lines = append(lines, "", r.caveat())
	return strings.Join(lines, "\n") + "\n"
}

func (r *scalingReport) caseRow(sc *scalingCase) string {
	ratio, verdict := r.evaluate(sc)
	ratioText := "n/a"
	if verdict != verdictTooFast {
		ratioText = fmt.Sprintf("%.2f", ratio)
	}
	return fmt.Sprintf("| `%s` | %s | %s | %s | %s |", sanitizeLabel(sc.label),
		formatSeconds(sc.unit), formatSeconds(sc.scaled), ratioText, verdictText(verdict))
}

func (r *scalingReport) headline() string {
	if failed := r.count(verdictFail); failed > 0 {
		return fmt.Sprintf("**%d of %d case(s) scale super-linearly.**", failed, len(r.cases))
	}
	return fmt.Sprintf("No case exceeds ratio %.1f.", r.maxRatio)
}

func (r *scalingReport) caveat() string {
	return fmt.Sprintf("_Linear work gives a ratio near %d and quadratic work near %d. Ratios come from means over few runs on a shared machine, "+
		"so treat values close to the threshold as noise until they reproduce._", r.factor, r.factor*r.factor)
}

func verdictText(verdict scalingVerdict) string {
	switch verdict {
	case verdictFail:
		return "**fail**"
	case verdictTooFast:
		return "too fast to measure"
	default:
		return "pass"
	}
}

func formatSeconds(res benchResult) string {
	return fmt.Sprintf("%.1f ± %.1f ms", res.Mean*1000, res.Stddev*1000)
}
