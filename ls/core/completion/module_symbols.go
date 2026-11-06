// Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
//
// WSO2 LLC licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except in compliance
// with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
// License for the specific language governing permissions and limitations
// under the License.

package completion

import "github.com/ballerina-nutcracker/ballerina/model"

// Prefixes are matched verbatim, without ModuleScope.GetPrefixedSymbol's langlib mapping.
func moduleContentSymbols(scope model.Scope, prefix string) (model.ExportedSymbolSpace, bool) {
	for scope != nil {
		switch s := scope.(type) {
		case *model.BlockScope:
			if space, ok := s.Prefix[prefix]; ok {
				return space, true
			}
			scope = s.Parent
		case *model.FunctionScope:
			if space, ok := s.Prefix[prefix]; ok {
				return space, true
			}
			scope = s.Parent
		case *model.ModuleScope:
			space, ok := s.Prefix[prefix]
			return space, ok
		case *model.PackageScope:
			if s.Virtual == nil {
				return model.ExportedSymbolSpace{}, false
			}
			space, ok := s.Virtual.Prefix[prefix]
			return space, ok
		default:
			return model.ExportedSymbolSpace{}, false
		}
	}
	return model.ExportedSymbolSpace{}, false
}
