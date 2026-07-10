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

	"ballerina/context"
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
	context.StageDesugaring,
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

	// Analysis and Desugaring grouped phase
	var analysisTotal time.Duration
	type analysisModuleEntry struct {
		name      string
		total     time.Duration
		subStages []context.StageTiming
	}
	var analysisEntries []analysisModuleEntry
	for _, ms := range allStats {
		var moduleTotal time.Duration
		var subs []context.StageTiming
		for _, stage := range analysisStages {
			if d := findStageDuration(ms, stage); d > 0 {
				subs = append(subs, context.StageTiming{Name: stage, Duration: d})
				moduleTotal += d
			}
		}
		if moduleTotal > 0 {
			analysisEntries = append(analysisEntries, analysisModuleEntry{ms.ModuleName, moduleTotal, subs})
			analysisTotal += moduleTotal
		}
	}
	if analysisTotal > 0 {
		grandTotal += analysisTotal
		fmt.Fprintf(&b, "  %-40s %s\n", "Analysis and Desugaring", formatDuration(analysisTotal))
		for _, e := range analysisEntries {
			fmt.Fprintf(&b, "    %-38s %s\n", e.name, formatDuration(e.total))
			for _, sub := range e.subStages {
				fmt.Fprintf(&b, "      %-36s %s\n", string(sub.Name), formatDuration(sub.Duration))
			}
		}
	}

	// BIR Generation
	var birTotal time.Duration
	type moduleEntry struct {
		name     string
		duration time.Duration
	}
	var birEntries []moduleEntry
	for _, ms := range allStats {
		if d := findStageDuration(ms, context.StageBIRGeneration); d > 0 {
			birEntries = append(birEntries, moduleEntry{ms.ModuleName, d})
			birTotal += d
		}
	}
	if birTotal > 0 {
		grandTotal += birTotal
		fmt.Fprintf(&b, "  %-40s %s\n", string(context.StageBIRGeneration), formatDuration(birTotal))
		for _, e := range birEntries {
			fmt.Fprintf(&b, "    %-38s %s\n", e.name, formatDuration(e.duration))
		}
	}

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

	var analysisTotal time.Duration
	for _, ms := range allStats {
		for _, stage := range analysisStages {
			analysisTotal += findStageDuration(ms, stage)
		}
	}
	if analysisTotal > 0 {
		grandTotal += analysisTotal
		fmt.Fprintf(&b, "  %-40s %s\n", "Analysis and Desugaring", formatDuration(analysisTotal))
	}

	var birTotal time.Duration
	for _, ms := range allStats {
		birTotal += findStageDuration(ms, context.StageBIRGeneration)
	}
	if birTotal > 0 {
		grandTotal += birTotal
		fmt.Fprintf(&b, "  %-40s %s\n", string(context.StageBIRGeneration), formatDuration(birTotal))
	}

	fmt.Fprintf(&b, "  %-40s %s\n", "Total", formatDuration(grandTotal))

	return b.String()
}

func findStageDuration(ms *context.ModuleStats, stage context.CompilationStage) time.Duration {
	for _, s := range ms.Stages {
		if s.Name == stage {
			return s.Duration
		}
	}
	return 0
}

func formatDuration(d time.Duration) string {
	return fmt.Sprintf("%8.2fms", float64(d.Nanoseconds())/1e6)
}
