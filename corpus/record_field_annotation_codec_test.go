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
	"fmt"
	"maps"
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/ballerina-nutcracker/ballerina/bir"
	bircodec "github.com/ballerina-nutcracker/ballerina/bir/codec"
	"github.com/ballerina-nutcracker/ballerina/context"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
	"github.com/ballerina-nutcracker/ballerina/values"
)

// TestRecordFieldAnnotationsSurviveBIRRoundtrip checks that the per-field
// annotations baked into a typedesc constant are preserved by BIR
// serialization, values included. Record field annotations are only reachable
// from native code, so the corpus output-comparison roundtrip cannot observe
// them.
func TestRecordFieldAnnotationsSurviveBIRRoundtrip(t *testing.T) {
	const fixture = "testdata/record-field-annotations.bal"

	var stdoutBuf, stderrBuf bytes.Buffer
	birPkgs, tyEnv, err := runCompilePhase(fixture, &stdoutBuf, &stderrBuf)
	if err != nil || len(birPkgs) == 0 {
		t.Fatalf("compilation failed for %s: %v (stderr: %s)", fixture, err, stderrBuf.String())
	}

	rootPkg := birPkgs[len(birPkgs)-1]
	before := recordFieldAnnotationDigest(rootPkg)
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

	after := recordFieldAnnotationDigest(deserialized)
	if !slices.Equal(before, after) {
		t.Errorf("field annotations not preserved across the BIR roundtrip\nbefore:\n%s\nafter:\n%s",
			strings.Join(before, "\n"), strings.Join(after, "\n"))
	}
}

// recordFieldAnnotationDigest renders every typedesc constant loaded by the
// package as deterministic "field key=value" lines, so that a roundtrip
// comparison catches a dropped or corrupted annotation value and not just a
// missing key.
func recordFieldAnnotationDigest(pkg *bir.BIRPackage) []string {
	var digest []string
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
				for _, field := range slices.Sorted(maps.Keys(td.FieldAnnotations)) {
					annotations := td.FieldAnnotations[field]
					for _, key := range slices.Sorted(maps.Keys(annotations)) {
						digest = append(digest, fmt.Sprintf("%s %s=%s", field, key,
							renderAnnotationValue(annotations[key])))
					}
				}
			}
		}
	}
	for i := range pkg.Functions {
		collect(&pkg.Functions[i])
	}
	collect(pkg.InitFunction)
	sort.Strings(digest)
	return digest
}

// renderAnnotationValue renders an annotation value structurally. Annotation
// values are not comparable by ==, and a mapping's own String() is not stable
// across a roundtrip for our purposes, so mappings and lists are rendered
// field by field.
func renderAnnotationValue(value values.AnnotationValue) string {
	switch value := value.(type) {
	case *values.Map:
		keys := value.Keys()
		sort.Strings(keys)
		parts := make([]string, 0, len(keys))
		for _, key := range keys {
			entry, _ := value.Get(key)
			parts = append(parts, fmt.Sprintf("%s:%s", key, renderAnnotationValue(entry)))
		}
		return "{" + strings.Join(parts, ",") + "}"
	case *values.List:
		parts := make([]string, 0, value.Len())
		for i := 0; i < value.Len(); i++ {
			parts = append(parts, renderAnnotationValue(value.Get(i)))
		}
		return "[" + strings.Join(parts, ",") + "]"
	case *values.RuntimeAnnotationValueRef:
		// A non-constant annotation value is a reference to a module global; the
		// reference itself is what the codec has to preserve.
		return "ref(" + value.GlobalLookupKey() + ")"
	default:
		return fmt.Sprintf("%v", value)
	}
}
