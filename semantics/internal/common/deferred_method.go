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

package common

import (
	"github.com/ballerina-nutcracker/ballerina/model"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
	"github.com/ballerina-nutcracker/ballerina/tools/diagnostics"
)

// DeferredMethodSymbol is a method placeholder produced before receiver types are known.
type DeferredMethodSymbol struct {
	name  string
	space *model.SymbolSpace
}

var _ model.Symbol = &DeferredMethodSymbol{}

func NewDeferredMethodSymbol(name string, space *model.SymbolSpace) *DeferredMethodSymbol {
	return &DeferredMethodSymbol{name: name, space: space}
}

func (d *DeferredMethodSymbol) MethodName() string              { return d.name }
func (d *DeferredMethodSymbol) SymbolSpace() *model.SymbolSpace { return d.space }

//nolint:forbidigo // Intentional guard: deferred method symbols must be resolved before access.
func (d *DeferredMethodSymbol) Name() string { panic("method symbol has not been resolved yet") }

//nolint:forbidigo // Intentional guard: deferred method symbols must be resolved before access.
func (d *DeferredMethodSymbol) Type() semtypes.SemType {
	panic("method symbol has not been resolved yet")
}

//nolint:forbidigo // Intentional guard: deferred method symbols must be resolved before access.
func (d *DeferredMethodSymbol) Kind() model.SymbolKind {
	panic("method symbol has not been resolved yet")
}

//nolint:forbidigo // Intentional guard: deferred method symbols must be resolved before access.
func (d *DeferredMethodSymbol) SetType(semtypes.SemType) {
	panic("method symbol has not been resolved yet")
}

//nolint:forbidigo // Intentional guard: deferred method symbols must be resolved before access.
func (d *DeferredMethodSymbol) Location() diagnostics.Location {
	panic("method symbol has not been resolved yet")
}

//nolint:forbidigo // Intentional guard: deferred method symbols must be resolved before access.
func (d *DeferredMethodSymbol) SetLocation(diagnostics.Location) {
	panic("method symbol has not been resolved yet")
}

//nolint:forbidigo // Intentional guard: deferred method symbols must be resolved before access.
func (d *DeferredMethodSymbol) IsPublic() bool { panic("method symbol has not been resolved yet") }

//nolint:forbidigo // Intentional guard: deferred method symbols must be resolved before access.
func (d *DeferredMethodSymbol) Copy() model.Symbol { panic("method symbol has not been resolved yet") }
