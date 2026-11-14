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

package types

import (
	"github.com/ballerina-nutcracker/ballerina/ast"
	"github.com/ballerina-nutcracker/ballerina/model"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
)

type typeResolutionContext struct {
	depth                 int
	defaults              []defaultBody
	collectingModuleTypes bool
	recordDefaults        map[model.SymbolRef]*recordDefaultDeclaration
	constantDefaults      map[model.SymbolRef]constantOutcome
	valueStatus           map[model.SymbolRef]resolutionStatus
	mappingDeclarations   []declaredMappingDefaults
}

func newTypeResolutionContext() typeResolutionContext {
	return typeResolutionContext{
		recordDefaults:   make(map[model.SymbolRef]*recordDefaultDeclaration),
		constantDefaults: make(map[model.SymbolRef]constantOutcome),
		valueStatus:      make(map[model.SymbolRef]resolutionStatus),
	}
}

func resolveBTypeNested(t typeResolver, chain *binding, btype ast.BType) (semtypes.SemType, *binding, bool) {
	t.resolutionContext().depth++
	defer func() { t.resolutionContext().depth-- }()
	return resolveBType(t, chain, btype)
}

func resolveFunctionSignatureNested(t typeResolver, chain *binding, fn *ast.BLangFunction) {
	t.resolutionContext().depth++
	defer func() { t.resolutionContext().depth-- }()
	resolveFunctionSignature(t, chain, fn)
}

func resolveResourceMethodSignatureNested(t typeResolver, isClient bool, isService bool, method *ast.BLangResourceMethod) bool {
	t.resolutionContext().depth++
	defer func() { t.resolutionContext().depth-- }()
	return resolveResourceMethodSignature(t, isClient, isService, method)
}

func resolveSimpleVariableInnerNested(t typeResolver, chain *binding, node *ast.BLangVariable) *binding {
	t.resolutionContext().depth++
	defer func() { t.resolutionContext().depth-- }()
	chain, _ = resolveSimpleVariableInner(t, chain, node)
	return chain
}

func resolveStreamCompletionTypeNested(t typeResolver, chain *binding, typeData *ast.TypeData) (semtypes.SemType, *binding, bool) {
	t.resolutionContext().depth++
	defer func() { t.resolutionContext().depth-- }()
	return resolveStreamCompletionType(t, chain, typeData)
}

func resolveTypeDataPairNested(t typeResolver, chain *binding, typeData *ast.TypeData) (semtypes.SemType, *binding, bool) {
	t.resolutionContext().depth++
	defer func() { t.resolutionContext().depth-- }()
	return resolveTypeDataPair(t, chain, typeData)
}
