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
	"sync"
	"testing"

	compilercontext "ballerina/context"
	"ballerina/semtypes"
)

func TestSyntaxTreeRecordsCachedParseDurationOnce(t *testing.T) {
	env := compilercontext.NewCompilerEnvironment(semtypes.CreateTypeEnv(), true)
	cx := compilercontext.NewCompilerContext(env)
	cx.InitModuleStats("test/module")

	docID := newDocumentIDFromString("doc", "main.bal", ModuleID{})
	docConfig := NewDocumentConfig(docID, "main.bal", "public function main() {}")
	docContext := newDocumentContext(docConfig, false, "")
	document := newDocument(docContext, &Module{moduleCtx: &moduleContext{compilerCtx: cx}})

	if syntaxTree := document.SyntaxTree(); syntaxTree == nil {
		t.Fatal("expected syntax tree")
	}

	var wg sync.WaitGroup
	for range 16 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if syntaxTree := docContext.parseWithStats(cx); syntaxTree == nil {
				t.Error("expected cached syntax tree")
			}
		}()
	}
	wg.Wait()

	stats := cx.GetModuleStats()
	if stats == nil {
		t.Fatal("expected module stats")
	}

	parseStageCount := 0
	for _, stage := range stats.Stages {
		if stage.Name == compilercontext.StageParse {
			parseStageCount++
		}
	}
	if parseStageCount != 1 {
		t.Fatalf("expected exactly one parse stage, got %d", parseStageCount)
	}
}
