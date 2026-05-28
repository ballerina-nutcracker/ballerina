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

package convert

import (
	"ballerina-lang-go/semtypes"
	"ballerina-lang-go/values"
)

func cloneValue(tc semtypes.Context, value values.BalValue, targetType semtypes.SemType) values.BalValue {
	if value == nil {
		return nil
	}
	switch v := value.(type) {
	case *values.List:
		effectiveTarget := effectiveTargetType(tc, targetType)
		lat := semtypes.ToListAtomicType(tc, effectiveTarget)
		if lat == nil {
			lat = semtypes.ToListAtomicType(tc, v.Type)
			if lat == nil {
				return value
			}
		}
		items := make([]values.BalValue, v.Len())
		for i := 0; i < v.Len(); i++ {
			items[i] = cloneValue(tc, v.Get(i), lat.MemberAtInnerVal(i))
		}
		restFiller, _ := values.FillerFactoryFor(tc, lat.Rest())
		readonly := semtypes.IsSubtype(tc, targetType, semtypes.VAL_READONLY)
		return values.NewList(targetType, lat, readonly, restFiller, v.Len(), items)
	case *values.Map:
		effectiveTarget := effectiveTargetType(tc, targetType)
		atomic := semtypes.ToMappingAtomicType(tc, effectiveTarget)
		mappingTarget := effectiveTarget
		if atomic == nil {
			atomic = semtypes.ToMappingAtomicType(tc, v.Type)
			mappingTarget = v.Type
			if atomic == nil {
				return value
			}
		}
		entries := make([]values.MapEntry, 0, v.Len())
		for _, key := range v.Keys() {
			val, _ := v.Get(key)
			fieldType := mappingFieldType(tc, mappingTarget, atomic, key)
			entries = append(entries, values.MapEntry{Key: key, Value: cloneValue(tc, val, fieldType)})
		}
		readonly := semtypes.IsSubtype(tc, targetType, semtypes.VAL_READONLY)
		return values.NewMap(targetType, atomic, readonly, entries)
	default:
		return value
	}
}
