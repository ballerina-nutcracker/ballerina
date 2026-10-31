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

package native

import (
	"github.com/ballerina-nutcracker/ballerina/values"
)

// toJSON is jBallerina's Native.toJson: lists become json[], mappings map<json>, and xml its string form.
// Like jBallerina it keeps a structure in the visited set after converting it and only drops its members,
// so a structure that refers back to an ancestor is reported as cyclic.
func (t *typer) toJSON(v values.BalValue, visited map[values.BalValue]struct{}) (values.BalValue, error) {
	switch v := v.(type) {
	case *values.List:
		if err := enterVisited(visited, v); err != nil {
			return nil, err
		}
		items := make([]values.BalValue, v.Len())
		for i := range items {
			member := v.Get(i)
			converted, err := t.toJSON(member, visited)
			if err != nil {
				return nil, err
			}
			items[i] = converted
			delete(visited, member)
		}
		return values.NewList(t.jsonListTy, t.jsonList, false, nil, len(items), items), nil
	case *values.Map:
		if err := enterVisited(visited, v); err != nil {
			return nil, err
		}
		entries := make([]values.MapEntry, 0, v.Len())
		for _, key := range v.Keys() {
			member, _ := v.Get(key)
			converted, err := t.toJSON(member, visited)
			if err != nil {
				return nil, err
			}
			entries = append(entries, values.MapEntry{Key: key, Value: converted})
			delete(visited, member)
		}
		return values.NewMap(t.jsonMapTy, t.jsonMap, false, entries), nil
	case values.XMLValue:
		return values.String(v, make(map[uintptr]bool)), nil
	default:
		return v, nil
	}
}

func enterVisited(visited map[values.BalValue]struct{}, v values.BalValue) error {
	if _, ok := visited[v]; ok {
		return cyclicReferenceError()
	}
	visited[v] = struct{}{}
	return nil
}
