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

package modules

import (
	"sync"

	"github.com/ballerina-nutcracker/ballerina/bir"
	"github.com/ballerina-nutcracker/ballerina/runtime/extern"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
	"github.com/ballerina-nutcracker/ballerina/values"
)

type BIRModule struct {
	Pkg     *bir.BIRPackage
	globals sync.Map // string -> values.BalValue
}

type ExternFunction struct {
	Name string
	Impl extern.NativeFunc
}

// NewBIRModule creates a module whose globals start out at the filler value of
// their declared type, then overlays globals. pkg may be nil for a module that
// only ever carries externally registered globals.
func NewBIRModule(typeCtx semtypes.Context, pkg *bir.BIRPackage, globals map[string]values.BalValue) *BIRModule {
	module := &BIRModule{Pkg: pkg}
	if pkg != nil {
		for key, gv := range pkg.GlobalVars {
			v, ok := values.FillerValue(typeCtx, gv.GetType())
			if ok {
				module.SetGlobal(key, v)
			}
		}
	}
	module.SetGlobals(globals)
	return module
}

// GetGlobal returns the value of the module-level variable stored under key.
// Safe for concurrent use; it does not imply the logical atomicity that a
// lock statement over the variable provides.
func (m *BIRModule) GetGlobal(key string) (values.BalValue, bool) {
	return m.globals.Load(key)
}

// SetGlobal stores value as the module-level variable under key. Safe for
// concurrent use; see GetGlobal on what it does not guarantee.
func (m *BIRModule) SetGlobal(key string, value values.BalValue) {
	m.globals.Store(key, value)
}

// SetGlobals stores every entry of globals, leaving keys absent from it
// untouched.
func (m *BIRModule) SetGlobals(globals map[string]values.BalValue) {
	for key, value := range globals {
		m.SetGlobal(key, value)
	}
}
