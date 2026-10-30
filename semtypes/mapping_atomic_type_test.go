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

func TestMappingAtomicTypeRest(t *testing.T) {
	env := CreateTypeEnv()
	cx := ContextFrom(env)
	fields := []Field{{name: "a", typeOf: Int}}
	tests := []struct {
		name string
		rest SemType
	}{
		{"closed", Never},
		{"typed rest", String},
		{"open", CreateAnydata(cx)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			md := NewMappingDefinition()
			atomic := ToMappingAtomicType(cx, md.Define(env, fields, tt.rest))
			if atomic == nil {
				t.Fatal("expected a mapping atomic type")
			}
			if !IsSameType(cx, atomic.Rest(), tt.rest) {
				t.Errorf("got rest %s expected %s", ToString(cx, atomic.Rest()), ToString(cx, tt.rest))
			}
		})
	}
}
