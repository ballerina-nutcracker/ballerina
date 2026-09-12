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

// moduleContentSymbols looks up the ExportedSymbolSpace bound to prefix (an
// import alias, e.g. the `io` in `io:println`) by walking outward from
// scope through its enclosing parents -- the same Main/Prefix/Parent shapes
// scope.go's walkScope already switches over, including walkScope's
// PackageScope fallback (a *model.PackageScope has no Prefix of its own; it
// delegates to its Virtual *model.ModuleScope, same as
// model.PackageScope.GetPrefixedSymbol does). This is the module/QName
// content-lookup helper for the import/module-qualified-reference family
// (today's no-op importHandler, excluded.go): once a per-provider ticket
// builds real candidates there, it can range over
// space.PublicMainSymbols() on the result. Returns ok=false if prefix isn't
// bound to an import anywhere in scope's chain.
//
// Known gap: unlike model.ModuleScope.GetPrefixedSymbol, this does not retry
// prefix through model's unexported mapToLangPrefixIfNeeded table, so
// lang-lib prefixes (e.g. `int:sum`, `string:...`) that rely on that
// built-in aliasing won't resolve here. No consumer needs it yet; revisit if
// model exports an equivalent.
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
