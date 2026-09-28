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
	"math"
	"os"
	"sort"
	"strings"
)

const (
	significanceSigmas      = 1.0
	significanceMinDeltaPct = 1.0
	timeNoiseThreshold      = 0.05
	memoryNoiseThreshold    = 0.03

	maxSummaryEntriesPerSection = 10
)

type entryKind int

const (
	kindUnavailable entryKind = iota
	kindRegression
	kindImprovement
	kindNeutral
)

type summaryEntry struct {
	label    string
	base     *benchResult
	head     *benchResult
	deltaPct float64
	sigmas   float64
	maxCV    float64
	kind     entryKind
	noisy    bool
}

func combinedSigma(base, head *benchResult) float64 {
	return math.Sqrt(base.Stddev*base.Stddev + head.Stddev*head.Stddev)
}

func coefficientOfVariation(res *benchResult) float64 {
	if res == nil || res.Mean <= 0 {
		return 0
	}
	return res.Stddev / res.Mean
}

func sanitizeLabel(label string) string {
	return strings.Map(func(r rune) rune {
		switch r {
		case '<', '>':
			return '?'
		default:
			return r
		}
	}, label)
}

func (r *report) classify(run *runResult) summaryEntry {
	entry := summaryEntry{label: sanitizeLabel(run.label), kind: kindUnavailable}
	base, head := resultPair(run)
	if base == nil || head == nil || base.Mean <= 0 || head.Mean <= 0 {
		return entry
	}
	entry.base = base
	entry.head = head

	delta := head.Mean - base.Mean
	entry.deltaPct = delta / base.Mean * 100
	entry.sigmas = sigmaMultiple(delta, combinedSigma(base, head))
	entry.kind = classifyDelta(delta, entry.deltaPct, entry.sigmas)
	entry.maxCV = math.Max(coefficientOfVariation(base), coefficientOfVariation(head))
	entry.noisy = entry.maxCV >= r.info().noiseThreshold
	return entry
}

// sigmaMultiple is never NaN: a zero delta scores zero however small the
// combined sigma is, which keeps 0/0 out of the comparators used for ordering.
func sigmaMultiple(delta, sigma float64) float64 {
	switch {
	case delta == 0:
		return 0
	case sigma == 0:
		return math.Inf(1)
	default:
		return math.Abs(delta) / sigma
	}
}

func classifyDelta(delta, deltaPct, sigmas float64) entryKind {
	significant := sigmas >= significanceSigmas && math.Abs(deltaPct) >= significanceMinDeltaPct
	switch {
	case significant && delta > 0:
		return kindRegression
	case significant && delta < 0:
		return kindImprovement
	default:
		return kindNeutral
	}
}

func (r *report) summarize() []summaryEntry {
	entries := make([]summaryEntry, 0, len(r.results))
	for i := range r.results {
		entries = append(entries, r.classify(&r.results[i]))
	}
	return entries
}

func (r *report) renderSummary() string {
	var regressions, improvements, unreliable, notMeasured []summaryEntry
	for _, entry := range r.summarize() {
		switch {
		case entry.kind == kindRegression:
			regressions = append(regressions, entry)
		case entry.kind == kindImprovement:
			improvements = append(improvements, entry)
		case entry.kind == kindUnavailable:
			notMeasured = append(notMeasured, entry)
		case entry.noisy:
			unreliable = append(unreliable, entry)
		}
	}
	sortBySignificance(regressions)
	sortBySignificance(improvements)
	sortByNoise(unreliable)
	sortByLabel(notMeasured)

	var lines []string
	lines = append(lines, r.renderSection("Regressions", regressions, r.deltaBullet)...)
	lines = append(lines, r.renderSection("Improvements", improvements, r.deltaBullet)...)
	lines = append(lines, r.renderSection("Unreliable measurements", unreliable, r.noiseBullet)...)
	lines = append(lines, r.renderSection("Not measured", notMeasured, unavailableBullet)...)
	if len(lines) == 0 {
		lines = append(lines, fmt.Sprintf("No case beyond %.0f sigma and %.0f%%.",
			significanceSigmas, significanceMinDeltaPct), "")
	}
	lines = append(lines, r.renderCaveat())
	return strings.Join(lines, "\n") + "\n"
}

func (r *report) renderSection(title string, entries []summaryEntry, bullet func(summaryEntry) string) []string {
	if len(entries) == 0 {
		return nil
	}
	lines := []string{"### " + title, ""}
	shown := entries
	if len(shown) > maxSummaryEntriesPerSection {
		shown = shown[:maxSummaryEntriesPerSection]
	}
	for _, entry := range shown {
		lines = append(lines, bullet(entry))
	}
	if remaining := len(entries) - len(shown); remaining > 0 {
		lines = append(lines, fmt.Sprintf("- …and %d more (see the table below)", remaining))
	}
	return append(lines, "")
}

func (r *report) deltaBullet(entry summaryEntry) string {
	bullet := fmt.Sprintf("- `%s` %+.2f%% (%s → %s, %s)", entry.label, entry.deltaPct,
		r.formatWithUnit(entry.base.Mean), r.formatWithUnit(entry.head.Mean), sigmaText(entry.sigmas))
	if entry.noisy {
		bullet += fmt.Sprintf(" — unreliable: %s", cvText(entry.maxCV))
	}
	return bullet
}

func (r *report) noiseBullet(entry summaryEntry) string {
	return fmt.Sprintf("- `%s` %s (%s → %s, %+.2f%%)", entry.label, cvText(entry.maxCV),
		r.formatWithUnit(entry.base.Mean), r.formatWithUnit(entry.head.Mean), entry.deltaPct)
}

func unavailableBullet(entry summaryEntry) string {
	return fmt.Sprintf("- `%s` — no base and head measurement pair", entry.label)
}

func (r *report) renderCaveat() string {
	return fmt.Sprintf("_A case is reported when it moves at least %.0f combined standard deviation and at least %.0f%%; "+
		"a measurement is unreliable once its coefficient of variation reaches %.0f%%. With 3 runs the standard deviation is a weak "+
		"estimate. The sigma multiple above and the ± column in the table are different quantities._",
		significanceSigmas, significanceMinDeltaPct, r.info().noiseThreshold*100)
}

func (r *report) formatWithUnit(value float64) string {
	return r.formatMetric(value) + " " + r.info().unit
}

func sigmaText(sigmas float64) string {
	if math.IsInf(sigmas, 1) {
		return "stddev 0"
	}
	return fmt.Sprintf("%.2f sigma", sigmas)
}

func cvText(cv float64) string {
	return fmt.Sprintf("worst CV %.1f%%", cv*100)
}

func sortBySignificance(entries []summaryEntry) {
	sort.SliceStable(entries, func(i, j int) bool {
		a, b := entries[i], entries[j]
		if a.sigmas != b.sigmas {
			return a.sigmas > b.sigmas
		}
		if math.Abs(a.deltaPct) != math.Abs(b.deltaPct) {
			return math.Abs(a.deltaPct) > math.Abs(b.deltaPct)
		}
		return a.label < b.label
	})
}

func sortByNoise(entries []summaryEntry) {
	sort.SliceStable(entries, func(i, j int) bool {
		a, b := entries[i], entries[j]
		if a.maxCV != b.maxCV {
			return a.maxCV > b.maxCV
		}
		return a.label < b.label
	})
}

func sortByLabel(entries []summaryEntry) {
	sort.SliceStable(entries, func(i, j int) bool {
		return entries[i].label < entries[j].label
	})
}

func (r *report) exportSummary(outPath string) error {
	if err := os.WriteFile(outPath, []byte(r.renderSummary()), 0o644); err != nil {
		return fmt.Errorf("failed to write summary report %q: %w", outPath, err)
	}
	return nil
}
