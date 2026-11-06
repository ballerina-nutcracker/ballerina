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

import "testing"

//go:noinline
func deferStartDefinitionSpan(t typeResolver, name string) {
	defer startDefinitionSpan(t, "Constant", name).end()
}

//go:noinline
func deferEnterDefinitionSpan(t typeResolver, name string) {
	defer enterDefinitionSpan(t, t.traceSpan().StartChildFunc("Service", func() string { return name })).end()
}

func TestDefinitionSpanDoesNotAllocate(t *testing.T) {
	// Not parallel: testing.AllocsPerRun reads process-wide malloc stats.
	var r typeResolver = &packageTypeResolver{}
	name := "c"
	tests := []struct {
		name    string
		bracket func()
	}{
		{"startDefinitionSpan", func() { deferStartDefinitionSpan(r, name) }},
		{"enterDefinitionSpan", func() { deferEnterDefinitionSpan(r, name) }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if allocs := testing.AllocsPerRun(1000, test.bracket); allocs != 0 {
				t.Fatalf("expected no allocations, got %v allocs/op", allocs)
			}
		})
	}
}
