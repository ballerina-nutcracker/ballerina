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

// Port of jBallerina's JsonTraverse: converts an in-memory json value to a target type. The field-name,
// field-hierarchy and rest-type stacks are kept exactly as jBallerina keeps them, including where a
// nested field name is left on the stack, because the stacks decide the field paths in error messages.

package native

import (
	"strings"

	"github.com/ballerina-nutcracker/ballerina/semtypes"
	"github.com/ballerina-nutcracker/ballerina/values"
)

type projection struct {
	allowDataProjection bool
	nilAsOptionalField  bool
	absentAsNilableType bool
}

type restEntry struct {
	ty semtypes.SemType
	ok bool
}

type traverser struct {
	*typer
	projection
	fieldHierarchy []*fieldSet
	restTypes      []restEntry
	// fieldNames is jBallerina's field-name deque, oldest entry first.
	fieldNames []string
	// renameRoot is set when target is the root value, so the outermost record gets the `@jsondata:Name` overrides.
	renameRoot bool
}

func traverse(t *typer, opts projection, json values.BalValue, target semtypes.SemType, renameRoot bool) (values.BalValue, error) {
	tr := &traverser{typer: t, projection: opts, renameRoot: renameRoot}
	return tr.traverseJSON(json, target)
}

func (tr *traverser) traverseJSON(json values.BalValue, target semtypes.SemType) (values.BalValue, error) {
	ti := tr.classify(target)
	if json == nil && tr.isNilable(target) {
		return nil, nil
	}
	switch ti.kind {
	case kindRecord:
		if _, ok := json.(*values.Map); !ok {
			return nil, incompatibleTypeError(tr.typeName(target), formatMessageArg(json))
		}
		fields, err := tr.recordFields(ti, tr.renameRoot && len(tr.fieldHierarchy) == 0)
		if err != nil {
			return nil, err
		}
		tr.pushFields(fields, ti)
		return tr.traverseMapOrArray(json, tr.newMappingBuilder(ti), ti)
	case kindList:
		if _, ok := json.(*values.List); !ok {
			return nil, incompatibleTypeError(tr.typeName(target), formatMessageArg(json))
		}
		return tr.traverseMapOrArray(json, tr.newListBuilder(ti), ti)
	case kindUnion:
		return tr.traverseUnion(json, ti)
	case kindJSON, kindAnydata:
		return json, nil
	case kindMap:
		tr.pushFields(emptyFieldSet(), ti)
		return tr.traverseMapOrArray(json, tr.newMappingBuilder(ti), ti)
	case kindUnsupported:
		return nil, invalidTypeError(tr.typeName(target), "anydata")
	default:
		return tr.convertToBasicType(json, target)
	}
}

func (tr *traverser) pushFields(fields *fieldSet, ti typeInfo) {
	rest, ok := restType(ti)
	tr.fieldHierarchy = append(tr.fieldHierarchy, fields)
	tr.restTypes = append(tr.restTypes, restEntry{ty: rest, ok: ok})
}

func (tr *traverser) popFields() *fieldSet {
	top := tr.fieldHierarchy[len(tr.fieldHierarchy)-1]
	tr.fieldHierarchy = tr.fieldHierarchy[:len(tr.fieldHierarchy)-1]
	return top
}

func (tr *traverser) popRestType() {
	tr.restTypes = tr.restTypes[:len(tr.restTypes)-1]
}

func (tr *traverser) popFieldName() string {
	name := tr.fieldNames[len(tr.fieldNames)-1]
	tr.fieldNames = tr.fieldNames[:len(tr.fieldNames)-1]
	return name
}

func (tr *traverser) traverseUnion(json values.BalValue, ti typeInfo) (values.BalValue, error) {
	for _, member := range tr.orderedMembers(ti.members, json) {
		hierarchySize, restSize, namesSize := len(tr.fieldHierarchy), len(tr.restTypes), len(tr.fieldNames)
		result, err := tr.traverseJSON(json, member)
		if err == nil {
			return result, nil
		}
		tr.fieldHierarchy = tr.fieldHierarchy[:min(hierarchySize, len(tr.fieldHierarchy))]
		tr.restTypes = tr.restTypes[:min(restSize, len(tr.restTypes))]
		// jBallerina restores the deque with pollLast, which drops the oldest names.
		if extra := len(tr.fieldNames) - namesSize; extra > 0 {
			tr.fieldNames = tr.fieldNames[extra:]
		}
	}
	return nil, incompatibleTypeError(tr.typeName(ti.ty), formatMessageArg(json))
}

// orderedMembers tries a member the value already belongs to before one that needs a numeric conversion.
// jBallerina tries members in declaration order, which runtime types do not keep.
func (tr *traverser) orderedMembers(members []semtypes.SemType, json values.BalValue) []semtypes.SemType {
	var exact, rest []semtypes.SemType
	for _, member := range members {
		if json != nil && tr.contains(member, json) && tr.classify(member).isScalar() {
			exact = append(exact, member)
		} else {
			rest = append(rest, member)
		}
	}
	return append(exact, rest...)
}

func (tr *traverser) traverseMapOrArray(json values.BalValue, node any, ti typeInfo) (values.BalValue, error) {
	switch json := json.(type) {
	case *values.Map:
		if builder, ok := node.(*mappingBuilder); ok {
			return tr.traverseMapValue(json, builder)
		}
	case *values.List:
		if builder, ok := node.(*listBuilder); ok {
			return tr.traverseArrayValue(json, builder)
		}
	}
	if ti.kind == kindRecord {
		tr.popFields()
		tr.popRestType()
	}
	if len(tr.fieldNames) == 0 {
		return nil, incompatibleTypeError(tr.typeName(ti.ty), formatMessageArg(json))
	}
	return nil, invalidTypeForFieldError(tr.currentFieldPath())
}

func (tr *traverser) traverseMapValue(json *values.Map, node *mappingBuilder) (values.BalValue, error) {
	for _, key := range json.Keys() {
		value, _ := json.Get(key)
		fields := tr.fieldHierarchy[len(tr.fieldHierarchy)-1]
		field := fields.remove(key)
		if field == nil {
			if rest := tr.restTypes[len(tr.restTypes)-1]; rest.ok {
				if fields.isRenamed(key) {
					return nil, duplicateFieldError(key)
				}
				if err := tr.addRestField(rest.ty, key, value, node); err != nil {
					return nil, err
				}
				continue
			}
			if tr.allowDataProjection {
				continue
			}
			tr.popFields()
			tr.popRestType()
			return nil, undefinedFieldError(key)
		}

		tr.fieldNames = append(tr.fieldNames, field.name)
		if tr.nilAsOptionalField && !tr.isNilable(field.ty) && value == nil && field.optional {
			continue
		}
		if tr.classify(field.ty).isExactBasic() {
			converted, err := tr.convertToBasicType(value, field.ty)
			if err != nil {
				return nil, err
			}
			node.put(tr.popFieldName(), converted)
			continue
		}
		converted, err := tr.traverseJSON(value, field.ty)
		if err != nil {
			return nil, err
		}
		node.put(field.name, converted)
	}
	remaining := tr.popFields()
	if err := tr.checkRequiredFields(remaining, node); err != nil {
		return nil, err
	}
	tr.popRestType()
	return node.build(tr.typer), nil
}

func (tr *traverser) checkRequiredFields(remaining *fieldSet, node *mappingBuilder) error {
	for _, field := range remaining.remaining() {
		if field.optional {
			continue
		}
		if tr.absentAsNilableType && tr.isNilable(field.ty) {
			node.put(field.name, nil)
			continue
		}
		return requiredFieldNotPresentError(field.name)
	}
	return nil
}

func (tr *traverser) traverseArrayValue(json *values.List, node *listBuilder) (values.BalValue, error) {
	atomic := node.atomic
	fixedLength := atomic.FixedLength()
	hasRest := !semtypes.IsNever(atomic.Rest())
	if !hasRest && !tr.allowDataProjection && json.Len() > fixedLength {
		return nil, arraySizeMismatchError()
	}
	for i := 0; i < json.Len(); i++ {
		if !hasRest && i >= fixedLength {
			break
		}
		converted, err := tr.traverseJSON(json.Get(i), atomic.MemberAtInnerVal(i))
		if err != nil {
			return nil, err
		}
		node.set(i, converted)
	}
	return node.build(tr.typer)
}

func (tr *traverser) addRestField(rest semtypes.SemType, key string, value values.BalValue, node *mappingBuilder) error {
	ti := tr.classify(rest)
	var converted values.BalValue
	var err error
	switch {
	case ti.kind == kindJSON || ti.kind == kindAnydata:
		converted = value
	case ti.isExactBasic() && ti.kind != kindNil:
		converted, err = tr.convertToBasicType(value, rest)
	default:
		converted, err = tr.traverseJSON(value, rest)
	}
	if err != nil {
		return err
	}
	node.put(key, converted)
	return nil
}

func (tr *traverser) convertToBasicType(json values.BalValue, target semtypes.SemType) (values.BalValue, error) {
	converted, err := values.CloneWithType(tr.tc, json, target)
	if err == nil {
		return converted, nil
	}
	if len(tr.fieldNames) == 0 {
		return nil, incompatibleTypeError(tr.typeName(target), javaValueOf(json))
	}
	return nil, incompatibleValueForFieldError(javaValueOf(json), tr.typeName(target), tr.currentFieldPath())
}

func (tr *traverser) currentFieldPath() string {
	return strings.Join(tr.fieldNames, ".")
}
