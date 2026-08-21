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

package corpus

import (
	"bytes"
	"maps"
	"slices"
	"testing"

	"github.com/ballerina-nutcracker/ballerina/bir"
	bircodec "github.com/ballerina-nutcracker/ballerina/bir/codec"
	"github.com/ballerina-nutcracker/ballerina/context"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
	"github.com/ballerina-nutcracker/ballerina/values"
)

// TestRecordFieldAnnotationsSurviveBIRRoundtrip checks that the per-field
// annotations baked into a typedesc constant are preserved by BIR
// serialization. Record field annotations are only reachable from native code,
// so the corpus output-comparison roundtrip cannot observe them.
func TestRecordFieldAnnotationsSurviveBIRRoundtrip(t *testing.T) {
	const fixture = "testdata/record-field-annotations.bal"

	var stdoutBuf, stderrBuf bytes.Buffer
	birPkgs, tyEnv, err := runCompilePhase(fixture, &stdoutBuf, &stderrBuf)
	if err != nil || len(birPkgs) == 0 {
		t.Fatalf("compilation failed for %s: %v (stderr: %s)", fixture, err, stderrBuf.String())
	}

	rootPkg := birPkgs[len(birPkgs)-1]
	before := recordFieldAnnotationKeys(rootPkg)
	if len(before) == 0 {
		t.Fatalf("no typedesc constant with field annotations found in %s", fixture)
	}

	serialized, err := bircodec.Marshal(tyEnv, rootPkg)
	if err != nil {
		t.Fatalf("BIR serialization failed: %v", err)
	}
	freshEnv := context.NewCompilerEnvironment(semtypes.CreateTypeEnv(), false)
	deserialized, err := bircodec.Unmarshal(context.NewCompilerContext(freshEnv), serialized)
	if err != nil {
		t.Fatalf("BIR deserialization failed: %v", err)
	}

	after := recordFieldAnnotationKeys(deserialized)
	if len(before) != len(after) {
		t.Fatalf("typedesc constant count changed: before %d, after %d", len(before), len(after))
	}
	for i, want := range before {
		if got := after[i]; !maps.Equal(want, got) {
			t.Errorf("field annotations for typedesc %d not preserved\nbefore: %v\nafter:  %v", i, want, got)
		}
	}
}

// recordFieldAnnotationKeys collects, for every typedesc constant loaded by the
// package, a field name -> sorted annotation keys mapping. Values are compared
// by key because annotation values are not comparable by ==.
func recordFieldAnnotationKeys(pkg *bir.BIRPackage) []map[string]string {
	var result []map[string]string
	collect := func(fn *bir.BIRFunction) {
		if fn == nil {
			return
		}
		for _, bb := range fn.BasicBlocks {
			for _, instruction := range bb.Instructions {
				load, ok := instruction.(*bir.ConstantLoad)
				if !ok {
					continue
				}
				td, ok := load.Value.(*values.TypeDesc)
				if !ok || len(td.FieldAnnotations) == 0 {
					continue
				}
				entry := make(map[string]string, len(td.FieldAnnotations))
				for _, field := range td.FieldAnnotations.SortedFields() {
					keys := slices.Sorted(maps.Keys(td.FieldAnnotations[field]))
					entry[field] = joinKeys(keys)
				}
				result = append(result, entry)
			}
		}
	}
	for i := range pkg.Functions {
		collect(&pkg.Functions[i])
	}
	collect(pkg.InitFunction)
	return result
}

func joinKeys(keys []string) string {
	out := ""
	for i, key := range keys {
		if i > 0 {
			out += ","
		}
		out += key
	}
	return out
}
