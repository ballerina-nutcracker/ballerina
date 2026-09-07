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

package analysis

import (
	"github.com/ballerina-nutcracker/ballerina/ast"
	"github.com/ballerina-nutcracker/ballerina/context"
	"github.com/ballerina-nutcracker/ballerina/tools/diagnostics"
)

// enclosingClassBody captures the subset of a class or service body that
// semantic analysis (in particular lock validation and isolated-field
// checks) needs when walking method bodies. Classes carry the user-supplied
// name; services carry their deterministic compiler-generated symbol name.
type enclosingClassBody struct {
	// name is the user-supplied class name or compiler-generated service name.
	name     string
	isolated bool
	fields   []*ast.BLangVariable
	initFn   *ast.BLangFunction
	position diagnostics.Location
}

func enclosingFromClass(c *ast.BLangClassDefinition) *enclosingClassBody {
	return &enclosingClassBody{
		name:     c.Name.GetValue(),
		isolated: c.IsIsolated(),
		fields:   c.Fields,
		initFn:   c.InitFunction,
		position: c.GetPosition(),
	}
}

func enclosingFromService(ctx *context.CompilerContext, s *ast.BLangService) *enclosingClassBody {
	return &enclosingClassBody{
		name:     ctx.SymbolName(s.Symbol()),
		isolated: s.IsIsolated(),
		fields:   s.Fields,
		initFn:   s.InitFunction,
		position: s.GetPosition(),
	}
}
