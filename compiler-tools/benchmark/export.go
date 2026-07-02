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

func (r *report) export(outPath string) error {
	rows := make([]row, 0, len(r.results))
	for _, run := range r.results {
		var base, head *benchResult
		if len(run.export.Results) >= 2 {
			base = &run.export.Results[0]
			head = &run.export.Results[1]
		}
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

func (r *report) formatMetric(value float64) string {
	if r.Mode == memoryMode {
		return fmt.Sprintf("%.3f", value)
	}
	return fmt.Sprintf("%.3f", value*1000.0)
}

func (r *report) title() string {
	if r.Mode == memoryMode {
		return "Ballerina Memory Benchmark"
	}
	return "Ballerina Benchmark"
}

func (r *report) meanLabel() string {
	if r.Mode == memoryMode {
		return "PEAK RSS (MiB)"
	}
	return "MEAN (ms)"
}

func (r *report) stddevLabel() string {
	if r.Mode == memoryMode {
		return "STDDEV (MiB)"
	}
	return "STDDEV (ms)"
}

func (r *report) winnerVerb() string {
	if r.Mode == memoryMode {
		return "uses less memory"
	}
	return "is faster"
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
	ratioStddev := ratio * math.Sqrt(
		math.Pow(result.Stddev/result.Mean, 2)+math.Pow(reference.Stddev/reference.Mean, 2),
	)

	return true, fmt.Sprintf("%.2f", ratio), fmt.Sprintf("%.2f", math.Abs(ratioStddev)), winnerRef
}
