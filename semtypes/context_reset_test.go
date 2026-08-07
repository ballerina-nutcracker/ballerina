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

package semtypes

import "testing"

// TestContextResetClearsMemoCaches verifies that Reset() actually empties the
// context's memo caches (guarding against unbounded growth of a pooled
// context reused across many unrelated invocations) while leaving the
// context behaviorally equivalent to a freshly built one for the same Env.
func TestContextResetClearsMemoCaches(t *testing.T) {
	env := CreateTypeEnv()
	ctx := ContextFrom(env)

	ld := &ListDefinition{}
	listTy := ld.DefineListTypeWrapped(env, []SemType{INT}, 1, NEVER, CellMutability_CELL_MUT_NONE)

	jsonBefore := CreateJSON(ctx)
	subtypeBefore := IsSubtype(ctx, listTy, jsonBefore)

	assertTrue(t, len(ctx._listMemo) > 0 || len(ctx._mappingMemo) > 0,
		"list/mapping memo should be populated before Reset")

	ctx.Reset()

	assertEqual(t, len(ctx._listMemo), 0)
	assertEqual(t, len(ctx._mappingMemo), 0)
	assertEqual(t, len(ctx._functionMemo), 0)
	assertEqual(t, len(ctx._comparableMemo), 0)
	assertEqual(t, len(ctx._fillerMemo), 0)
	assertEqual(t, len(ctx._streamImplementorMemo), 0)
	assertEqual(t, len(ctx._listenerMemo), 0)

	jsonAfter := CreateJSON(ctx)
	subtypeAfter := IsSubtype(ctx, listTy, jsonAfter)

	assertTrue(t, IsSubtype(ctx, jsonBefore, jsonAfter), "json type should be equivalent after Reset")
	assertTrue(t, IsSubtype(ctx, jsonAfter, jsonBefore), "json type should be equivalent after Reset")
	assertEqual(t, subtypeBefore, subtypeAfter)
}
