// Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
//
// WSO2 LLC. licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except
// in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build debug

package parser_test

import (
	"bytes"
	"strings"
	"sync"
	"testing"

	compilercontext "github.com/ballerina-nutcracker/ballerina/context"
	"github.com/ballerina-nutcracker/ballerina/parser"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
)

func TestConcurrentParsesUseIndependentDebugWriters(t *testing.T) {
	type parseCase struct {
		name, source string
		output       bytes.Buffer
	}
	cases := []*parseCase{
		{name: "alpha.bal", source: "function alphaOnly() {}"},
		{name: "beta.bal", source: "function betaOnly() {}"},
	}
	var wg sync.WaitGroup
	for _, test := range cases {
		wg.Add(1)
		go func() {
			defer wg.Done()
			env := compilercontext.NewCompilerEnvironment(semtypes.CreateTypeEnv(), false)
			cx := compilercontext.NewCompilerContext(env)
			_, err := parser.GetSyntaxTreeWithIdentity(cx, test.name, "identity:"+test.name, test.source,
				parser.DebugOptions{DumpTokens: true, DumpSyntaxTree: true}, &test.output)
			if err != nil {
				t.Errorf("parse %s: %v", test.name, err)
			}
		}()
	}
	wg.Wait()
	for index, test := range cases {
		own := strings.TrimSuffix(test.name, ".bal") + "Only"
		other := strings.TrimSuffix(cases[1-index].name, ".bal") + "Only"
		output := test.output.String()
		if !strings.Contains(output, own) {
			t.Fatalf("debug output for %s lacks %q: %s", test.name, own, output)
		}
		if strings.Contains(output, other) {
			t.Fatalf("debug output for %s contains concurrent parse token %q", test.name, other)
		}
	}
}

func TestZeroDebugOptionsWriteNothing(t *testing.T) {
	env := compilercontext.NewCompilerEnvironment(semtypes.CreateTypeEnv(), false)
	cx := compilercontext.NewCompilerContext(env)
	var output bytes.Buffer
	if _, err := parser.GetSyntaxTreeWithIdentity(cx, "main.bal", "identity:main", "function main() {}",
		parser.DebugOptions{}, &output); err != nil {
		t.Fatal(err)
	}
	if output.Len() != 0 {
		t.Fatalf("zero debug options wrote %q", output.String())
	}
}
