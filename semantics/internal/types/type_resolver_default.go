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

import "github.com/ballerina-nutcracker/ballerina/model"

// defaultTypeResolver resolves one default expression. A default is resolved after the declaration that owns it,
// so the resolver is created where the default is found and keeps the scope and resolver context flags in effect
// there. Everything else, including the naming counters and implicit imports of the enclosing scope, delegates to
// the enclosing resolver. Defaults met while resolving this one are queued here and drained when it ends.
type defaultTypeResolver struct {
	typeResolver
	scope           model.Scope
	context         resolverContext
	pendingDefaults []func()
}

func newDefaultTypeResolver(t typeResolver) *defaultTypeResolver {
	return &defaultTypeResolver{typeResolver: t, scope: t.currentScope(), context: resolverContextFlags(t)}
}

func (d *defaultTypeResolver) parent() typeResolver          { return d.typeResolver }
func (d *defaultTypeResolver) currentScope() model.Scope     { return d.scope }
func (d *defaultTypeResolver) setCurrentScope(s model.Scope) { d.scope = s }

func (d *defaultTypeResolver) deferDefault(resolve func()) {
	d.pendingDefaults = append(d.pendingDefaults, resolve)
}

// resolve runs resolve on d, then the defaults it queued.
func (d *defaultTypeResolver) resolve(resolve func(t typeResolver)) {
	resolve(d)
	drainDefaults(&d.pendingDefaults)
}
