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

package langinternalruntime

import (
	"ballerina/decimal"
	"ballerina/runtime"
	"ballerina/runtime/extern"
	"ballerina/semtypes"
	"ballerina/values"
	"math"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"unsafe"
)

const (
	orgName    = "ballerina"
	moduleName = "lang.__internal"
)

func initInternalModule(rt *runtime.Runtime) {
	runtime.RegisterExternFunction(rt, orgName, moduleName, "querySort", func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
		sortKeyRows := args[0].(*values.List)
		sortDirections := args[1].(*values.List)
		rowIndices := args[2].(*values.List)
		payloadRows := args[3].(*values.List)

		rowCount := rowIndices.Len()
		keyCount := sortDirections.Len()

		directionFlags := make([]bool, keyCount)
		for keyIndex := 0; keyIndex < keyCount; keyIndex++ {
			directionFlags[keyIndex] = sortDirections.Get(keyIndex).(bool)
		}

		keyRows := make([]*values.List, rowCount)
		for rowIndex := 0; rowIndex < rowCount; rowIndex++ {
			keyRows[rowIndex] = sortKeyRows.Get(rowIndex).(*values.List)
		}

		payloadCount := payloadRows.Len()
		payloadLists := make([]*values.List, payloadCount)
		for payloadIndex := 0; payloadIndex < payloadCount; payloadIndex++ {
			payloadLists[payloadIndex] = payloadRows.Get(payloadIndex).(*values.List)
		}

		order := make([]int, rowCount)
		for rowIndex := 0; rowIndex < rowCount; rowIndex++ {
			order[rowIndex] = rowIndex
		}

		sort.SliceStable(order, func(i, j int) bool {
			leftRow := order[i]
			rightRow := order[j]
			leftKeys := keyRows[leftRow]
			rightKeys := keyRows[rightRow]
			for keyIndex := 0; keyIndex < keyCount; keyIndex++ {
				cmp := compareQuerySortValues(leftKeys.Get(keyIndex), rightKeys.Get(keyIndex), directionFlags[keyIndex])
				switch cmp {
				case values.CmpLT:
					return true
				case values.CmpGT:
					return false
				}
			}
			return false
		})

		reorderListInPlace(ctx, rowIndices, order)
		reorderListInPlace(ctx, sortKeyRows, order)
		for payloadIndex := 0; payloadIndex < payloadCount; payloadIndex++ {
			reorderListInPlace(ctx, payloadLists[payloadIndex], order)
		}
		return nil, nil
	})
	runtime.RegisterExternFunction(rt, orgName, moduleName, "queryGroup", func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
		rows := args[0].(*values.List)
		keyRows := args[1].(*values.List)
		scalarFlags := args[2].(*values.List)
		return queryGroup(ctx, rows, keyRows, scalarFlags)
	})
	runtime.RegisterExternFunction(rt, orgName, moduleName, "queryCollect", func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
		rows := args[0].(*values.List)
		slotCount := args[1].(int64)
		flattenFlags := args[2].(*values.List)
		return queryCollect(ctx, rows, int(slotCount), flattenFlags)
	})
	runtime.RegisterExternFunction(rt, orgName, moduleName, "escapeXMLContent", func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
		return values.EscapeXMLContent(values.String(args[0], nil)), nil
	})
	runtime.RegisterExternFunction(rt, orgName, moduleName, "escapeXMLAttribute", func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
		return values.EscapeXMLAttribute(values.String(args[0], nil)), nil
	})
}

type queryGroupState struct {
	row *values.List
}

type queryGroupIndex map[string]int

func newQueryList(ctx *extern.Context) *values.List {
	return values.NewList(semtypes.LIST, semtypes.ToListAtomicType(ctx.TypeCtx, semtypes.LIST), false, nil, 0, nil)
}

func queryGroup(ctx *extern.Context, rows *values.List, keyRows *values.List, scalarFlags *values.List) (*values.List, error) {
	rowCount := rows.Len()
	slotCount := scalarFlags.Len()
	scalarSlots := make([]bool, slotCount)
	for slot := 0; slot < slotCount; slot++ {
		scalarSlots[slot] = scalarFlags.Get(slot).(bool)
	}

	result := newQueryList(ctx)
	groups := make([]queryGroupState, 0)
	groupIndices := make(queryGroupIndex)
	for rowIndex := 0; rowIndex < rowCount; rowIndex++ {
		sourceRow := rows.Get(rowIndex).(*values.List)
		keyRow := keyRows.Get(rowIndex).(*values.List)
		groupIndex, keySignature, found := findQueryGroup(groupIndices, keyRow)
		if !found {
			groupRow := createQueryGroupRow(ctx, sourceRow, scalarSlots)
			groups = append(groups, queryGroupState{row: groupRow})
			groupIndices[keySignature] = len(groups) - 1
			result.Append(ctx.TypeCtx, groupRow)
			continue
		}
		appendQueryGroupRow(ctx, groups[groupIndex].row, sourceRow, scalarSlots)
	}
	return result, nil
}

func findQueryGroup(groupIndices queryGroupIndex, keyRow *values.List) (int, string, bool) {
	keySignature := queryGroupKeySignature(keyRow)
	groupIndex, ok := groupIndices[keySignature]
	if !ok {
		return 0, keySignature, false
	}
	return groupIndex, keySignature, true
}

func queryGroupKeySignature(keyRow *values.List) string {
	var b strings.Builder
	writeQueryGroupValueSignature(&b, keyRow, make(map[uintptr]bool))
	return b.String()
}

func writeQueryGroupValueSignature(b *strings.Builder, v values.BalValue, visited map[uintptr]bool) {
	switch typedValue := v.(type) {
	case nil:
		b.WriteString("nil;")
	case bool:
		b.WriteString("bool:")
		b.WriteString(strconv.FormatBool(typedValue))
		b.WriteByte(';')
	case int64:
		b.WriteString("int:")
		b.WriteString(strconv.FormatInt(typedValue, 10))
		b.WriteByte(';')
	case float64:
		writeQueryGroupFloatSignature(b, typedValue)
	case string:
		writeQueryGroupStringSignature(b, "string", typedValue)
	case *decimal.Decimal:
		b.WriteString("decimal:")
		b.WriteString(typedValue.String())
		b.WriteByte(';')
	case *values.List:
		writeQueryGroupListSignature(b, typedValue, visited)
	case *values.Map:
		writeQueryGroupMapSignature(b, typedValue, visited)
	default:
		b.WriteString("unknown:")
		b.WriteString(reflect.TypeOf(typedValue).String())
		b.WriteByte(';')
	}
}

func writeQueryGroupFloatSignature(b *strings.Builder, f float64) {
	b.WriteString("float:")
	switch {
	case math.IsNaN(f):
		b.WriteString("NaN")
	case f == 0:
		b.WriteByte('0')
	default:
		b.WriteString(strconv.FormatFloat(f, 'g', -1, 64))
	}
	b.WriteByte(';')
}

func writeQueryGroupStringSignature(b *strings.Builder, tag string, s string) {
	b.WriteString(tag)
	b.WriteByte(':')
	b.WriteString(strconv.Itoa(len(s)))
	b.WriteByte(':')
	b.WriteString(s)
	b.WriteByte(';')
}

func writeQueryGroupListSignature(b *strings.Builder, list *values.List, visited map[uintptr]bool) {
	ptr := uintptr(unsafe.Pointer(list))
	if visited[ptr] {
		b.WriteString("list:cycle;")
		return
	}
	visited[ptr] = true
	defer delete(visited, ptr)

	b.WriteString("list:")
	b.WriteString(strconv.Itoa(list.Len()))
	b.WriteByte('[')
	for i := 0; i < list.Len(); i++ {
		writeQueryGroupValueSignature(b, list.Get(i), visited)
	}
	b.WriteByte(']')
}

func writeQueryGroupMapSignature(b *strings.Builder, mapping *values.Map, visited map[uintptr]bool) {
	ptr := uintptr(unsafe.Pointer(mapping))
	if visited[ptr] {
		b.WriteString("map:cycle;")
		return
	}
	visited[ptr] = true
	defer delete(visited, ptr)

	keys := mapping.Keys()
	sort.Strings(keys)
	b.WriteString("map:")
	b.WriteString(strconv.Itoa(len(keys)))
	b.WriteByte('{')
	for _, key := range keys {
		writeQueryGroupStringSignature(b, "key", key)
		value, _ := mapping.Get(key)
		writeQueryGroupValueSignature(b, value, visited)
	}
	b.WriteByte('}')
}

func createQueryGroupRow(ctx *extern.Context, sourceRow *values.List, scalarSlots []bool) *values.List {
	groupRow := newQueryList(ctx)
	for slot, isScalar := range scalarSlots {
		value := sourceRow.Get(slot)
		if isScalar {
			groupRow.Append(ctx.TypeCtx, value)
			continue
		}
		valuesForGroup := newQueryList(ctx)
		valuesForGroup.Append(ctx.TypeCtx, value)
		groupRow.Append(ctx.TypeCtx, valuesForGroup)
	}
	return groupRow
}

func appendQueryGroupRow(ctx *extern.Context, groupRow *values.List, sourceRow *values.List, scalarSlots []bool) {
	for slot, isScalar := range scalarSlots {
		if isScalar {
			continue
		}
		valuesForGroup := groupRow.Get(slot).(*values.List)
		valuesForGroup.Append(ctx.TypeCtx, sourceRow.Get(slot))
	}
}

func queryCollect(ctx *extern.Context, rows *values.List, slotCount int, flattenFlags *values.List) (*values.List, error) {
	flattenSlots := make([]bool, slotCount)
	for slot := 0; slot < slotCount; slot++ {
		flattenSlots[slot] = flattenFlags.Get(slot).(bool)
	}
	resultRow := newQueryList(ctx)
	for slot := 0; slot < slotCount; slot++ {
		resultRow.Append(ctx.TypeCtx, newQueryList(ctx))
	}
	for rowIndex := 0; rowIndex < rows.Len(); rowIndex++ {
		row := rows.Get(rowIndex).(*values.List)
		for slot := 0; slot < slotCount; slot++ {
			valuesForSlot := resultRow.Get(slot).(*values.List)
			if !flattenSlots[slot] {
				valuesForSlot.Append(ctx.TypeCtx, row.Get(slot))
				continue
			}
			nestedValues := row.Get(slot).(*values.List)
			for valueIndex := 0; valueIndex < nestedValues.Len(); valueIndex++ {
				valuesForSlot.Append(ctx.TypeCtx, nestedValues.Get(valueIndex))
			}
		}
	}
	return resultRow, nil
}

func reorderListInPlace(ctx *extern.Context, list *values.List, order []int) {
	old := make([]values.BalValue, list.Len())
	for i := range list.Len() {
		old[i] = list.Get(i)
	}
	for i, sourceIndex := range order {
		list.FillingSet(ctx.TypeCtx, i, old[sourceIndex])
	}
}

func compareQuerySortValues(left values.BalValue, right values.BalValue, isAscending bool) values.CompareResult {
	return values.CompareK(left, right, isAscending)
}

func init() {
	runtime.RegisterModuleInitializer(initInternalModule)
}
