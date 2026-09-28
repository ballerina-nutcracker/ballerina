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
	"html/template"
	"math"
	"os"
	"time"
)

type (
	benchExport struct {
		Results []benchResult `json:"results"`
	}
	benchResult struct {
		Command string  `json:"command"`
		Mean    float64 `json:"mean"`
		Stddev  float64 `json:"stddev"`
		Median  float64 `json:"median"`
	}
)

func parseHyperfineExport(path string) (*benchExport, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read hyperfine export file: %w", err)
	}

	var export benchExport
	if err := json.Unmarshal(b, &export); err != nil {
		return nil, fmt.Errorf("failed to parse hyperfine export JSON: %w", err)
	}

	return &export, nil
}

type (
	modeInfo struct {
		title          string
		meanLabel      string
		stddevLabel    string
		unit           string
		winnerVerb     string
		scale          float64
		noiseThreshold float64
	}
	report struct {
		BaseRef   string
		HeadRef   string
		Mode      benchmarkMode
		Generated time.Time
		results   []runResult
	}
	row struct {
		Label          string
		Base           *benchResult
		Head           *benchResult
		BaseMean       string
		BaseStddev     string
		HeadMean       string
		HeadStddev     string
		DeltaAvailable bool
		DeltaRatio     string
		DeltaStddev    string
		DeltaWinnerRef string
	}
)

func resultPair(run *runResult) (base, head *benchResult) {
	if len(run.export.Results) < 2 {
		return nil, nil
	}
	return &run.export.Results[0], &run.export.Results[1]
}

func (r *report) export(outPath string) error {
	rows := make([]row, 0, len(r.results))
	for i := range r.results {
		run := &r.results[i]
		base, head := resultPair(run)
		tblRow := row{
			Label: run.label,
			Base:  base,
			Head:  head,
		}
		if base != nil {
			tblRow.BaseMean = r.formatMetric(base.Mean)
			tblRow.BaseStddev = r.formatMetric(base.Stddev)
		}
		if head != nil {
			tblRow.HeadMean = r.formatMetric(head.Mean)
			tblRow.HeadStddev = r.formatMetric(head.Stddev)
		}
		tblRow.DeltaAvailable, tblRow.DeltaRatio, tblRow.DeltaStddev, tblRow.DeltaWinnerRef = computeDelta(base, head, r.BaseRef, r.HeadRef)
		rows = append(rows, tblRow)
	}

	tpl := template.Must(template.New("report").Parse(htmlTemplate))
	f, err := os.Create(outPath)
	if err != nil {
		return fmt.Errorf("failed to generate html report %q: %w", outPath, err)
	}
	defer func() { _ = f.Close() }()

	data := struct {
		Report      report
		Rows        []row
		Title       string
		MeanLabel   string
		StddevLabel string
		WinnerVerb  string
	}{
		Report:      *r,
		Rows:        rows,
		Title:       r.title(),
		MeanLabel:   r.meanLabel(),
		StddevLabel: r.stddevLabel(),
		WinnerVerb:  r.winnerVerb(),
	}

	if err := tpl.Execute(f, data); err != nil {
		return fmt.Errorf("failed to render html report %q: %w", outPath, err)
	}
	return nil
}

func infoForMode(mode benchmarkMode) modeInfo {
	if mode == memoryMode {
		return modeInfo{
			title:          "Ballerina Memory Benchmark",
			meanLabel:      "PEAK RSS (MiB)",
			stddevLabel:    "STDDEV (MiB)",
			unit:           "MiB",
			winnerVerb:     "uses less memory",
			scale:          1.0,
			noiseThreshold: memoryNoiseThreshold,
		}
	}
	return modeInfo{
		title:          "Ballerina Benchmark",
		meanLabel:      "MEAN (ms)",
		stddevLabel:    "STDDEV (ms)",
		unit:           "ms",
		winnerVerb:     "is faster",
		scale:          1000.0,
		noiseThreshold: timeNoiseThreshold,
	}
}

func (r *report) info() modeInfo {
	return infoForMode(r.Mode)
}

func (r *report) formatMetric(value float64) string {
	return fmt.Sprintf("%.3f", value*r.info().scale)
}

func (r *report) title() string {
	return r.info().title
}

func (r *report) meanLabel() string {
	return r.info().meanLabel
}

func (r *report) stddevLabel() string {
	return r.info().stddevLabel
}

func (r *report) winnerVerb() string {
	return r.info().winnerVerb
}

func computeDelta(base, head *benchResult, baseRef, headRef string) (bool, string, string, string) {
	if base == nil || head == nil || base.Mean <= 0 || head.Mean <= 0 {
		return false, "", "", ""
	}

	winnerRef := headRef
	result := base
	reference := head

	switch {
	case base.Mean < head.Mean:
		winnerRef = baseRef
		result = head
		reference = base
	case base.Mean == head.Mean:
		winnerRef = "tie"
		result = base
		reference = head
	}

	ratio := result.Mean / reference.Mean
	if base.Mean == head.Mean {
		ratio = 1.0
	}

	// Uses the same uncertainty propagation formula as hyperfine:
	// https://github.com/sharkdp/hyperfine/blob/327d5f4d9107141929f67f062bf9ef59f98b7399/src/benchmark/relative_speed.rs#L56-L64
	resultRelStddev := result.Stddev / result.Mean
	referenceRelStddev := reference.Stddev / reference.Mean
	ratioStddev := ratio * math.Sqrt(resultRelStddev*resultRelStddev+referenceRelStddev*referenceRelStddev)

	return true, fmt.Sprintf("%.2f", ratio), fmt.Sprintf("%.2f", math.Abs(ratioStddev)), winnerRef
}
