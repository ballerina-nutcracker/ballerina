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

package values

import (
	"reflect"
	"sync"
	"testing"

	"ballerina-lang-go/semtypes"
)

// TestJSONListAndMapTypesConcurrentSameEnv reproduces two independent callers
// (e.g. two stdlibs) racing to build the json[]/map<json> types for the same
// environment. Each goroutine builds its own Context (as semtypes.ContextFrom
// does on every call in real use), mirroring the scenario JSONListAndMapTypes
// exists to guard: without synchronizing the cache-miss path, concurrent
// callers can each register their own atom for "the same" recursive type,
// so the types they get back would carry different atom-table references
// even though they're structurally the same json[]/map<json> type.
func TestJSONListAndMapTypesConcurrentSameEnv(t *testing.T) {
	env := semtypes.CreateTypeEnv()

	const n = 32
	listResults := make([]semtypes.SemType, n)
	mapResults := make([]semtypes.SemType, n)

	var start, done sync.WaitGroup
	start.Add(1)
	done.Add(n)
	for i := range n {
		go func(i int) {
			defer done.Done()
			start.Wait()
			ctx := semtypes.ContextFrom(env)
			listTy, mapTy := JSONListAndMapTypes(ctx)
			listResults[i] = listTy
			mapResults[i] = mapTy
		}(i)
	}
	start.Done()
	done.Wait()

	for i := 1; i < n; i++ {
		if !reflect.DeepEqual(listResults[0], listResults[i]) {
			t.Fatalf("goroutine %d got a different list type than goroutine 0 - concurrent calls built duplicate atoms", i)
		}
		if !reflect.DeepEqual(mapResults[0], mapResults[i]) {
			t.Fatalf("goroutine %d got a different map type than goroutine 0 - concurrent calls built duplicate atoms", i)
		}
	}
}
