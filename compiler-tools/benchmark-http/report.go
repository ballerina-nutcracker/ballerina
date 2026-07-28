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
	"fmt"
	"math"
	"os"
	"sort"
	"strings"
)

// result is the machine-readable verdict consumed by the CI gate step.
type result struct {
	Regressed       bool    `json:"regressed"`
	Error           bool    `json:"error"`
	Message         string  `json:"message,omitempty"`
	ThroughputBase  float64 `json:"throughputBase,omitempty"`
	ThroughputHead  float64 `json:"throughputHead,omitempty"`
	ThroughputDelta float64 `json:"throughputDelta,omitempty"` // percent, head vs base
	ThresholdPct    float64 `json:"thresholdPct,omitempty"`
}

func writeResultJSON(path string, r result) error {
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(b, '\n'), 0o644)
}

func writeFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0o644)
}

// --- statistics over repeated samples ---

func median(xs []float64) float64 {
	if len(xs) == 0 {
		return 0
	}
	s := append([]float64(nil), xs...)
	sort.Float64s(s)
	n := len(s)
	if n%2 == 1 {
		return s[n/2]
	}
	return (s[n/2-1] + s[n/2]) / 2
}

func stddev(xs []float64) float64 {
	if len(xs) < 2 {
		return 0
	}
	var mean float64
	for _, x := range xs {
		mean += x
	}
	mean /= float64(len(xs))
	var sum float64
	for _, x := range xs {
		d := x - mean
		sum += d * d
	}
	return math.Sqrt(sum / float64(len(xs)-1)) // sample stddev
}

func field(samples []sample, sel func(sample) float64) []float64 {
	out := make([]float64, len(samples))
	for i, s := range samples {
		out[i] = sel(s)
	}
	return out
}

// buildReport renders the markdown comment body and computes the verdict.
func buildReport(baseRef, headRef string, base, head []sample, cfg config) (string, result) {
	thr := func(s sample) float64 { return s.throughput }
	avg := func(s sample) float64 { return s.avgMs }
	p99 := func(s sample) float64 { return s.p99Ms }
	up := func(s sample) float64 { return s.startupSec }
	rss := func(s sample) float64 { return s.rssMB }

	baseThr, headThr := median(field(base, thr)), median(field(head, thr))
	// Fail closed: without samples on both sides and a positive baseline there is
	// no meaningful delta, so report an error rather than a misleading "no
	// regression" from a zero/empty baseline.
	if len(base) == 0 || len(head) == 0 || baseThr <= 0 {
		msg := fmt.Sprintf("invalid throughput samples (base runs=%d, head runs=%d, base median=%.1f req/s)",
			len(base), len(head), baseThr)
		return "HTTP benchmark could not compute a valid comparison:\n\n```text\n" + msg + "\n```\n",
			result{Error: true, Message: msg, ThresholdPct: cfg.threshold}
	}
	delta := (headThr - baseThr) / baseThr * 100
	regressed := delta <= -cfg.threshold

	var b strings.Builder
	fmt.Fprintf(&b, "| Metric | `%s` (base) | `%s` (head) | Δ (head vs base) |\n", baseRef, headRef)
	b.WriteString("|---|---|---|---|\n")

	// Throughput — higher is better; a negative delta is the regression signal.
	thrDelta := signedPct(baseThr, headThr)
	if regressed {
		thrDelta += " ⚠"
	}
	fmt.Fprintf(&b, "| Throughput (req/s) | %s | %s | %s |\n",
		countWithSpread(field(base, thr)), countWithSpread(field(head, thr)), thrDelta)
	baseAvg, headAvg := median(field(base, avg)), median(field(head, avg))
	fmt.Fprintf(&b, "| Avg latency (ms) | %.2f | %.2f | %s |\n", baseAvg, headAvg, signedPct(baseAvg, headAvg))
	baseP99, headP99 := median(field(base, p99)), median(field(head, p99))
	fmt.Fprintf(&b, "| p99 latency (ms) | %.2f | %.2f | %s |\n", baseP99, headP99, signedPct(baseP99, headP99))
	baseUp, headUp := median(field(base, up)), median(field(head, up))
	fmt.Fprintf(&b, "| Startup (s) | %.3f | %.3f | %s |\n", baseUp, headUp, signedPct(baseUp, headUp))
	fmt.Fprintf(&b, "| Peak RSS (MB) | %s | %s | %s |\n",
		rssCell(field(base, rss)), rssCell(field(head, rss)), rssDelta(field(base, rss), field(head, rss)))

	b.WriteString("\n")
	if regressed {
		fmt.Fprintf(&b, "**Result: ⚠ REGRESSION** — throughput %s exceeds the %.0f%% threshold.\n",
			signedPct(baseThr, headThr), cfg.threshold)
	} else {
		fmt.Fprintf(&b, "**Result: ✅ no significant regression** (throughput %s, threshold %.0f%%).\n",
			signedPct(baseThr, headThr), cfg.threshold)
	}
	fmt.Fprintf(&b, "\n_%d repeats, %s each, %d connections. Median shown; ± is sample stddev across repeats._\n",
		cfg.repeats, cfg.duration, cfg.conns)

	return b.String(), result{
		Regressed:       regressed,
		ThroughputBase:  round1(baseThr),
		ThroughputHead:  round1(headThr),
		ThroughputDelta: round1(delta),
		ThresholdPct:    cfg.threshold,
	}
}

// countWithSpread formats a throughput series as "median (±stddev)".
func countWithSpread(xs []float64) string {
	return fmt.Sprintf("%s (±%s)", comma(median(xs)), comma(stddev(xs)))
}

func rssCell(xs []float64) string {
	m := median(xs)
	if m == 0 {
		return "n/a"
	}
	return fmt.Sprintf("%.1f", m)
}

func rssDelta(base, head []float64) string {
	if median(base) == 0 || median(head) == 0 {
		return "n/a"
	}
	return signedPct(median(base), median(head))
}

func signedPct(base, head float64) string {
	if base == 0 {
		return "n/a"
	}
	d := (head - base) / base * 100
	return fmt.Sprintf("%+.1f%%", d)
}

func comma(v float64) string {
	n := int64(v + 0.5)
	s := fmt.Sprintf("%d", n)
	neg := strings.HasPrefix(s, "-")
	s = strings.TrimPrefix(s, "-")
	var out []byte
	for i, c := range []byte(s) {
		if i > 0 && (len(s)-i)%3 == 0 {
			out = append(out, ',')
		}
		out = append(out, c)
	}
	if neg {
		return "-" + string(out)
	}
	return string(out)
}

func round1(v float64) float64 { return math.Round(v*10) / 10 }
