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

package semantics

import "ballerina/ast"

// enclosingClassBody captures the subset of a class or service body that
// semantic analysis (in particular lock validation and isolated-field
// checks) needs when walking method bodies. Classes carry a className
// derived from the user-supplied name; services have no name and instead
// carry a per-service tag used to disambiguate field lock keys.
type enclosingClassBody struct {
	// name is the user-supplied class name for class bodies, empty for
	// service bodies (services have no name).
	name     string
	isolated bool
	fields   []ast.SimpleVariableNode
	initFn   *ast.BLangFunction
}

func enclosingFromClass(c *ast.BLangClassDefinition) *enclosingClassBody {
	return &enclosingClassBody{
		name:     c.Name.GetValue(),
		isolated: c.IsIsolated(),
		fields:   c.Fields,
		initFn:   c.InitFunction,
	}
}

func enclosingFromService(s *ast.BLangService) *enclosingClassBody {
	return &enclosingClassBody{
		isolated: s.IsIsolated(),
		fields:   s.Fields,
		initFn:   s.InitFunction,
	}
}
