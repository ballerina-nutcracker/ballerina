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

package scope

import (
	"github.com/ballerina-nutcracker/ballerina/ast"
	"github.com/ballerina-nutcracker/ballerina/model"
)

// outsideLockVarRef: an isolated module-level variable may only be accessed
// inside a lock.
func outsideLockVarRef(cx Context, ref *ast.BLangVarRef) {
	if md, ok := cx.ModuleVar(ref.Symbol()); ok && md.IsIsolated() {
		cx.SemanticErr("access of an isolated variable must be inside a lock statement", ref.GetPosition())
	}
}

// outsideLockFieldAccess: a mutable field of an isolated object may only be
// accessed through self inside a lock, except in the class's init method.
func outsideLockFieldAccess(cx Context, access *ast.BLangFieldBaseAccess, receiver model.SymbolRef) {
	fn, ok := EnclosingFunction(cx)
	if !ok || !fn.isSelf(receiver) || !fn.class.IsIsolated() {
		return
	}
	field, ok := fn.class.field(access.Field.GetValue())
	if !ok || field.immutable || fn.isClassInit() {
		return
	}
	cx.SemanticErr("mutable field access within isolated object must be inside a lock statement", access.GetPosition())
}
