// Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
//
// WSO2 LLC. licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except
// in compliance with the License.
//
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

package native

import (
	stdruntime "runtime"
	"sync"
	"weak"

	"ballerina/semtypes"
)

type jsonTypePair struct {
	listTy semtypes.SemType
	mapTy  semtypes.SemType
}

// jsonTypesByEnv associates a weak pointer to a semtypes.Env with the canonical
// json[]/map<json> semtypes built for it, self-cleaning once the Env is unreachable.
var jsonTypesByEnv sync.Map // weak.Pointer[env-pointee] -> jsonTypePair

// JSONListAndMapTypes returns the canonical json[]/map<json> semtypes for a context's
// environment, memoized per environment. semtypes.ContextFrom builds a fresh Context
// (with empty memo maps) on every call, so semtypes.CreateJSON's own per-Context memo
// does not stop two independent callers (e.g. two stdlibs) building separate
// ListDefinition/MappingDefinition instances for "the same" json list/map type — each
// registers its own atom into the shared environment, which is otherwise-harmless but
// shifts how unrelated recursive types print in that environment (extra atoms shift
// atom-table numbering). Every caller that needs these types for GoToBalValue must go
// through this shared accessor instead of building its own.
func JSONListAndMapTypes(ctx semtypes.Context) (semtypes.SemType, semtypes.SemType) {
	env := ctx.Env()
	// Boxed as `any` immediately: env's pointee type is unexported in semtypes, so this
	// package can only name weak.Pointer[...] for it via type inference, not explicitly —
	// boxing lets the AddCleanup callback below stay a plain func(any).
	key := any(weak.Make(env))
	if v, ok := jsonTypesByEnv.Load(key); ok {
		p := v.(jsonTypePair)
		return p.listTy, p.mapTy
	}
	jsonTy := semtypes.CreateJSON(ctx)
	listLd := semtypes.NewListDefinition()
	mapMd := semtypes.NewMappingDefinition()
	listTy := listLd.DefineListTypeWrappedWithEnvSemType(env, jsonTy)
	mapTy := mapMd.DefineMappingTypeWrapped(env, nil, jsonTy)
	p := jsonTypePair{listTy, mapTy}
	jsonTypesByEnv.Store(key, p)
	stdruntime.AddCleanup(env, cleanupJSONTypes, key)
	return listTy, mapTy
}

func cleanupJSONTypes(key any) {
	jsonTypesByEnv.Delete(key)
}
