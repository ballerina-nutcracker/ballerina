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
	"regexp"
	"strings"
	"testing"
	"time"

	"ballerina-lang-go/context"
)

func allStagesStats() []*context.ModuleStats {
	return []*context.ModuleStats{
		{
			ModuleName: "myorg/mymod",
			Stages: []context.StageTiming{
				{Name: context.StageParse, Duration: 5 * time.Millisecond},
				{Name: context.StageASTBuild, Duration: 3 * time.Millisecond},
				{Name: context.StageImportResolution, Duration: 1 * time.Millisecond},
				{Name: context.StageSymbolResolution, Duration: 4 * time.Millisecond},
				{Name: context.StageTopLevelTypeResolution, Duration: 2 * time.Millisecond},
				{Name: context.StageLocalNodeResolution, Duration: 6 * time.Millisecond},
				{Name: context.StageSemanticAnalysis, Duration: 7 * time.Millisecond},
				{Name: context.StageCFGCreation, Duration: 2 * time.Millisecond},
				{Name: context.StageCFGAnalysis, Duration: 1 * time.Millisecond},
				{Name: context.StageDesugaring, Duration: 3 * time.Millisecond},
				{Name: context.StageBIRGeneration, Duration: 8 * time.Millisecond},
			},
		},
	}
}

var (
	durationPattern         = regexp.MustCompile(`\d+\.\d+ms`)
	topLevelAnalysisPattern = regexp.MustCompile(`(?m)^  Analysis\s+\d+\.\d+ms$`)
)

func TestFormatStatsReport(t *testing.T) {
	report := formatStatsReport(allStagesStats())

	assertContains(t, report, "Compilation Stats:")
	assertContains(t, report, "Parse")
	assertContains(t, report, "AST Build")
	assertContains(t, report, "Import Resolution")
	assertContains(t, report, "Symbol Resolution")
	assertContains(t, report, "Top-Level Type Resolution")
	if !topLevelAnalysisPattern.MatchString(report) {
		t.Errorf("expected output to contain the top-level Analysis row, got:\n%s", report)
	}
	assertContains(t, report, "Local Type Resolution")
	assertContains(t, report, "Semantic Analysis")
	assertContains(t, report, "CFG Creation")
	assertContains(t, report, "CFG Analysis")
	assertContains(t, report, "Desugaring")
	assertContains(t, report, "BIR Generation")
	assertContains(t, report, "Total")
	assertContains(t, report, "myorg/mymod")

	matches := durationPattern.FindAllString(report, -1)
	if len(matches) == 0 {
		t.Error("expected duration values in format N.NNms, found none")
	}
}

func TestFormatStatsReportOneline(t *testing.T) {
	report := formatStatsReportOneline(allStagesStats())

	assertContains(t, report, "Compilation Stats:")
	assertContains(t, report, "Parse")
	assertContains(t, report, "Local Type Resolution")
	assertContains(t, report, "Semantic Analysis")
	assertContains(t, report, "CFG Creation")
	assertContains(t, report, "CFG Analysis")
	assertContains(t, report, "Desugaring")
	assertContains(t, report, "BIR Generation")
	assertContains(t, report, "Total")

	if strings.Contains(report, "myorg/mymod") {
		t.Error("oneline report should not contain per-module breakdown")
	}

	matches := durationPattern.FindAllString(report, -1)
	if len(matches) == 0 {
		t.Error("expected duration values in format N.NNms, found none")
	}
}

func TestFormatStatsReportEmpty(t *testing.T) {
	if got := formatStatsReport(nil); got != "" {
		t.Errorf("expected empty string for nil input, got %q", got)
	}
	if got := formatStatsReportOneline(nil); got != "" {
		t.Errorf("expected empty string for nil input, got %q", got)
	}
	if got := formatStatsReport([]*context.ModuleStats{}); got != "" {
		t.Errorf("expected empty string for empty input, got %q", got)
	}
}

func TestFindStageDurationSumsRepeatedStageEntries(t *testing.T) {
	stats := &context.ModuleStats{
		ModuleName: "multi-file/mod",
		Stages: []context.StageTiming{
			{Name: context.StageParse, Duration: 2 * time.Millisecond},
			{Name: context.StageASTBuild, Duration: 10 * time.Millisecond},
			{Name: context.StageParse, Duration: 3 * time.Millisecond},
		},
	}

	if got := findStageDuration(stats, context.StageParse); got != 5*time.Millisecond {
		t.Fatalf("expected repeated parse entries to be summed, got %s", got)
	}
}

func TestFormatStatsReportPartialStages(t *testing.T) {
	stats := []*context.ModuleStats{
		{
			ModuleName: "partial/mod",
			Stages: []context.StageTiming{
				{Name: context.StageParse, Duration: 10 * time.Millisecond},
				{Name: context.StageSemanticAnalysis, Duration: 20 * time.Millisecond},
			},
		},
	}

	report := formatStatsReport(stats)
	assertContains(t, report, "Parse")
	assertContains(t, report, "Semantic Analysis")
	assertContains(t, report, "Total")

	if strings.Contains(report, "BIR Generation") {
		t.Error("report should omit BIR Generation when no BIR stage data exists")
	}
	if strings.Contains(report, "Import Resolution") {
		t.Error("report should omit Import Resolution when no data exists for it")
	}
}

func assertContains(t *testing.T, s, substr string) {
	t.Helper()
	if !strings.Contains(s, substr) {
		t.Errorf("expected output to contain %q, got:\n%s", substr, s)
	}
}
