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

package projects

import (
	"fmt"
	"strings"
	"time"

	"ballerina-lang-go/context"
)

var standaloneStages = []context.CompilationStage{
	context.StageParse,
	context.StageASTBuild,
	context.StageImportResolution,
	context.StageSymbolResolution,
	context.StageTopLevelTypeResolution,
}

var analysisStages = []context.CompilationStage{
	context.StageLocalNodeResolution,
	context.StageSemanticAnalysis,
	context.StageCFGCreation,
	context.StageCFGAnalysis,
}

func formatStatsReport(allStats []*context.ModuleStats) string {
	if len(allStats) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString("Compilation Stats:\n")

	var grandTotal time.Duration

	for _, stage := range standaloneStages {
		var stageTotal time.Duration
		type moduleEntry struct {
			name     string
			duration time.Duration
		}
		var entries []moduleEntry
		for _, ms := range allStats {
			if d := findStageDuration(ms, stage); d > 0 {
				entries = append(entries, moduleEntry{ms.ModuleName, d})
				stageTotal += d
			}
		}
		if stageTotal == 0 {
			continue
		}
		grandTotal += stageTotal
		fmt.Fprintf(&b, "  %-40s %s\n", string(stage), formatDuration(stageTotal))
		for _, e := range entries {
			fmt.Fprintf(&b, "    %-38s %s\n", e.name, formatDuration(e.duration))
		}
	}

	analysisTotal := writeGroupedStageStats(&b, allStats, "Analysis", analysisStages)
	grandTotal += analysisTotal

	desugaringTotal := writeSingleStageStats(&b, allStats, context.StageDesugaring)
	grandTotal += desugaringTotal

	birTotal := writeSingleStageStats(&b, allStats, context.StageBIRGeneration)
	grandTotal += birTotal

	fmt.Fprintf(&b, "  %-40s %s\n", "Total", formatDuration(grandTotal))

	return b.String()
}

func formatStatsReportOneline(allStats []*context.ModuleStats) string {
	if len(allStats) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString("Compilation Stats:\n")

	var grandTotal time.Duration

	for _, stage := range standaloneStages {
		var stageTotal time.Duration
		for _, ms := range allStats {
			stageTotal += findStageDuration(ms, stage)
		}
		if stageTotal == 0 {
			continue
		}
		grandTotal += stageTotal
		fmt.Fprintf(&b, "  %-40s %s\n", string(stage), formatDuration(stageTotal))
	}

	for _, stage := range analysisStages {
		stageTotal := singleStageTotal(allStats, stage)
		if stageTotal == 0 {
			continue
		}
		grandTotal += stageTotal
		fmt.Fprintf(&b, "  %-40s %s\n", string(stage), formatDuration(stageTotal))
	}

	desugaringTotal := singleStageTotal(allStats, context.StageDesugaring)
	if desugaringTotal > 0 {
		grandTotal += desugaringTotal
		fmt.Fprintf(&b, "  %-40s %s\n", string(context.StageDesugaring), formatDuration(desugaringTotal))
	}

	birTotal := singleStageTotal(allStats, context.StageBIRGeneration)
	if birTotal > 0 {
		grandTotal += birTotal
		fmt.Fprintf(&b, "  %-40s %s\n", string(context.StageBIRGeneration), formatDuration(birTotal))
	}

	fmt.Fprintf(&b, "  %-40s %s\n", "Total", formatDuration(grandTotal))

	return b.String()
}

func writeGroupedStageStats(b *strings.Builder, allStats []*context.ModuleStats, name string, stages []context.CompilationStage) time.Duration {
	type moduleEntry struct {
		name      string
		total     time.Duration
		subStages []context.StageTiming
	}

	var total time.Duration
	var entries []moduleEntry
	for _, ms := range allStats {
		var moduleTotal time.Duration
		var subs []context.StageTiming
		for _, stage := range stages {
			if d := findStageDuration(ms, stage); d > 0 {
				subs = append(subs, context.StageTiming{Name: stage, Duration: d})
				moduleTotal += d
			}
		}
		if moduleTotal > 0 {
			entries = append(entries, moduleEntry{ms.ModuleName, moduleTotal, subs})
			total += moduleTotal
		}
	}
	if total == 0 {
		return 0
	}

	fmt.Fprintf(b, "  %-40s %s\n", name, formatDuration(total))
	for _, e := range entries {
		fmt.Fprintf(b, "    %-38s %s\n", e.name, formatDuration(e.total))
		for _, sub := range e.subStages {
			fmt.Fprintf(b, "      %-36s %s\n", string(sub.Name), formatDuration(sub.Duration))
		}
	}
	return total
}

func writeSingleStageStats(b *strings.Builder, allStats []*context.ModuleStats, stage context.CompilationStage) time.Duration {
	type moduleEntry struct {
		name     string
		duration time.Duration
	}

	var total time.Duration
	var entries []moduleEntry
	for _, ms := range allStats {
		if d := findStageDuration(ms, stage); d > 0 {
			entries = append(entries, moduleEntry{ms.ModuleName, d})
			total += d
		}
	}
	if total == 0 {
		return 0
	}

	fmt.Fprintf(b, "  %-40s %s\n", string(stage), formatDuration(total))
	for _, e := range entries {
		fmt.Fprintf(b, "    %-38s %s\n", e.name, formatDuration(e.duration))
	}
	return total
}

func singleStageTotal(allStats []*context.ModuleStats, stage context.CompilationStage) time.Duration {
	var total time.Duration
	for _, ms := range allStats {
		total += findStageDuration(ms, stage)
	}
	return total
}

func findStageDuration(ms *context.ModuleStats, stage context.CompilationStage) time.Duration {
	var total time.Duration
	for _, s := range ms.Stages {
		if s.Name == stage {
			total += s.Duration
		}
	}
	return total
}

func formatDuration(d time.Duration) string {
	return fmt.Sprintf("%8.2fms", float64(d.Nanoseconds())/1e6)
}
