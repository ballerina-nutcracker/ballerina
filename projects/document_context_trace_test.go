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

//go:build debug

package projects

import (
	"encoding/json"
	"strings"
	"sync"
	"testing"

	compilercontext "github.com/ballerina-nutcracker/ballerina/context"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
)

const traceTestSource = "public function main() {}"

func TestCachedParsesRecordOneParseSpan(t *testing.T) {
	env := compilercontext.NewCompilerEnvironment(semtypes.CreateTypeEnv(), true)
	cx := compilercontext.NewCompilerContext(env)
	docContext := newTraceTestDocumentContext(false)

	parseConcurrently(t, docContext, cx, 16)

	if count := parseSpanCount(t, env); count != 1 {
		t.Fatalf("parse spans = %d, want 1 (cache hits must not record spans)", count)
	}
}

func TestCacheDisabledParsesRecordDistinctSpans(t *testing.T) {
	env := compilercontext.NewCompilerEnvironment(semtypes.CreateTypeEnv(), true)
	cx := compilercontext.NewCompilerContext(env)
	docContext := newTraceTestDocumentContext(true)

	parseConcurrently(t, docContext, cx, 8)

	if count := parseSpanCount(t, env); count != 8 {
		t.Fatalf("parse spans = %d, want 8", count)
	}
}

func newTraceTestDocumentContext(disableSyntaxTree bool) *documentContext {
	docID := newDocumentIDFromString("doc", "main.bal", ModuleID{})
	docConfig := NewDocumentConfig(docID, "main.bal", traceTestSource)
	return newDocumentContext(docConfig, disableSyntaxTree, "")
}

func parseConcurrently(t *testing.T, docContext *documentContext, cx *compilercontext.CompilerContext, count int) {
	t.Helper()
	var wg sync.WaitGroup
	for range count {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if syntaxTree := docContext.parse(cx); syntaxTree == nil {
				t.Error("expected syntax tree")
			}
		}()
	}
	wg.Wait()
}

func parseSpanCount(t *testing.T, env *compilercontext.CompilerEnvironment) int {
	t.Helper()
	data, err := env.TraceJSON()
	if err != nil {
		t.Fatalf("TraceJSON: %v", err)
	}
	var document struct {
		TraceEvents []struct {
			Name string `json:"name"`
			Args struct {
				SpanID string `json:"span_id"`
			} `json:"args"`
		} `json:"traceEvents"`
	}
	if err := json.Unmarshal(data, &document); err != nil {
		t.Fatalf("unmarshal trace: %v", err)
	}
	spanIDs := make(map[string]bool)
	count := 0
	for _, event := range document.TraceEvents {
		if !strings.HasPrefix(event.Name, "Parse ") {
			continue
		}
		if spanIDs[event.Args.SpanID] {
			t.Fatalf("duplicate span id %s", event.Args.SpanID)
		}
		spanIDs[event.Args.SpanID] = true
		count++
	}
	return count
}
