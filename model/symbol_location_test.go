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

package model

import (
	"testing"

	"ballerina/tools/diagnostics"
)

func TestSymbolConstructorsSetLocation(t *testing.T) {
	location := diagnostics.NewBuiltinLocation()
	value := NewVariableSymbol("value", false, false, false, location)
	typeSymbol := NewTypeSymbol("type", false, location)
	record := NewRecordSymbol("record", false, location)
	object := NewObjectTypeSymbol("object", false, location)

	symbols := []Symbol{
		NewFunctionSymbol("function", FunctionSignature{}, false, location),
		&value,
		&typeSymbol,
		NewXMLNSSymbol("prefix", "uri", location),
		NewClassSymbol("class", false, location),
		NewNetworkClassSymbol("client", false, location),
		NewResourceMethodSymbol("resource", "get", false, location),
		&record,
		&object,
		NewDependentlyTypedFunctionSymbol("dependent", nil, 0, 0, false, location),
	}

	for _, symbol := range symbols {
		if got := symbol.Location(); got != location {
			t.Errorf("%T location = %s, want %s", symbol, got, location)
		}
	}
}
