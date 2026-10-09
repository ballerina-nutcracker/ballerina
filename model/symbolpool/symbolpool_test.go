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

package symbolpool_test

import (
	"testing"

	"github.com/ballerina-nutcracker/ballerina/context"
	"github.com/ballerina-nutcracker/ballerina/model"
	"github.com/ballerina-nutcracker/ballerina/model/symbolpool"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
	"github.com/ballerina-nutcracker/ballerina/tools/diagnostics"
)

// TestValueSymbolFlagsRoundtrip is a unit test because a project test cannot
// observe a lost flag: TestProjectSerializationRoundtrip only runs -v projects,
// and dropping TopLevel or Final only makes an imported module variable
// narrowable, which can surface only as a missing diagnostic.
func TestValueSymbolFlagsRoundtrip(t *testing.T) {
	t.Parallel()
	env := context.NewCompilerEnvironment(semtypes.CreateTypeEnv(), false)
	pkgID := env.NewPackageID(model.Name("testorg"), []model.Name{model.Name("flags")}, model.Name("0.1.0"))
	space := env.NewSymbolSpace(*pkgID)
	location := diagnostics.NewBuiltinLocation()

	mutable := model.NewVariableSymbol("mutableVar", true, false, false, location)
	mutable.SetTopLevel()
	mutable.SetType(semtypes.Int)
	space.AddSymbol("mutableVar", &mutable)

	final := model.NewVariableSymbol("finalVar", true, false, false, location)
	final.SetTopLevel()
	final.SetFinal()
	final.SetType(semtypes.Int)
	space.AddSymbol("finalVar", &final)

	constant := model.NewConstantValueSymbol("CONST", true, location)
	constant.SetTopLevel()
	constant.SetType(semtypes.Int)
	constant.SetConstantValue(int64(3))
	space.AddSymbol("CONST", constant)

	data, err := symbolpool.Marshal(model.NewExportedSymbolSpaces([]*model.SymbolSpace{space}, nil), env)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	freshEnv := context.NewCompilerEnvironment(semtypes.CreateTypeEnv(), false)
	exported, err := symbolpool.Unmarshal(freshEnv, data)
	if err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	cases := []struct {
		name  string
		final bool
		cnst  bool
	}{
		{name: "mutableVar"},
		{name: "finalVar", final: true},
		{name: "CONST", cnst: true},
	}
	for _, tc := range cases {
		ref, ok := exported.GetSymbol(tc.name)
		if !ok {
			t.Fatalf("%s: missing after roundtrip", tc.name)
		}
		metadata, ok := freshEnv.ValueSymbolMetadata(ref)
		if !ok {
			t.Fatalf("%s: not a value symbol after roundtrip", tc.name)
		}
		if !metadata.TopLevel {
			t.Errorf("%s: TopLevel lost in roundtrip", tc.name)
		}
		if metadata.Final != tc.final {
			t.Errorf("%s: Final = %v, want %v", tc.name, metadata.Final, tc.final)
		}
		if metadata.Const != tc.cnst {
			t.Errorf("%s: Const = %v, want %v", tc.name, metadata.Const, tc.cnst)
		}
	}
}
