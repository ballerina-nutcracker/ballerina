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

// Port of jBallerina's JsonCreator and of the JsonParser.StateMachine methods that create, link and
// finalize the values the parser builds.

package native

import (
	"errors"
	"strings"

	"github.com/ballerina-nutcracker/ballerina/semtypes"
	"github.com/ballerina-nutcracker/ballerina/values"
)

func (sm *stateMachine) peekExpectedType() *semtypes.SemType {
	if len(sm.expectedTypes) == 0 {
		return nil
	}
	return sm.expectedTypes[len(sm.expectedTypes)-1]
}

func (sm *stateMachine) popExpectedType() *semtypes.SemType {
	top := sm.peekExpectedType()
	if len(sm.expectedTypes) > 0 {
		sm.expectedTypes = sm.expectedTypes[:len(sm.expectedTypes)-1]
	}
	return top
}

func (sm *stateMachine) pushContext(ctx parserContext) {
	sm.parserContexts = append(sm.parserContexts, ctx)
}

func (sm *stateMachine) popContext() {
	sm.parserContexts = sm.parserContexts[:len(sm.parserContexts)-1]
}

func (sm *stateMachine) peekContext() parserContext {
	if len(sm.parserContexts) == 0 {
		return contextArray
	}
	return sm.parserContexts[len(sm.parserContexts)-1]
}

func (sm *stateMachine) pushFieldNames() {
	sm.fieldNameHierarchy = append(sm.fieldNameHierarchy, nil)
}

func (sm *stateMachine) popFieldNames() {
	sm.fieldNameHierarchy = sm.fieldNameHierarchy[:len(sm.fieldNameHierarchy)-1]
}

func (sm *stateMachine) pushFieldName(name string) {
	top := len(sm.fieldNameHierarchy) - 1
	sm.fieldNameHierarchy[top] = append(sm.fieldNameHierarchy[top], name)
}

func (sm *stateMachine) popFieldName() string {
	top := len(sm.fieldNameHierarchy) - 1
	names := sm.fieldNameHierarchy[top]
	name := names[len(names)-1]
	sm.fieldNameHierarchy[top] = names[:len(names)-1]
	return name
}

func (sm *stateMachine) currentFieldPath() string {
	parts := make([]string, len(sm.fieldNameHierarchy))
	for i, names := range sm.fieldNameHierarchy {
		if len(names) > 0 {
			parts[i] = names[len(names)-1]
		}
	}
	return strings.Join(parts, ".")
}

func (sm *stateMachine) updateExpectedType(fields *fieldSet, rest *semtypes.SemType) {
	sm.fieldHierarchy = append(sm.fieldHierarchy, fields)
	sm.visitedFieldHierarchy = append(sm.visitedFieldHierarchy, make(map[string]*recordField))
	sm.restTypes = append(sm.restTypes, rest)
	sm.pushFieldNames()
}

func (sm *stateMachine) updateIndexOfArrayElement() {
	sm.arrayIndexes[len(sm.arrayIndexes)-1]++
}

func (sm *stateMachine) initRootMapValue() (*mappingBuilder, error) {
	expected := *sm.peekExpectedType()
	sm.pushContext(contextMap)
	ti := sm.classify(expected)
	switch ti.kind {
	case kindRecord, kindMap:
		return sm.newMappingBuilder(ti), nil
	case kindJSON:
		return sm.newJSONMappingBuilder(), nil
	case kindAnydata:
		return sm.newAnydataMappingBuilder(), nil
	case kindUnion:
		sm.unionDepth++
		sm.pushFieldNames()
		return sm.newJSONMappingBuilder(), nil
	default:
		return nil, invalidTypeError(sm.typeName(expected), "map type")
	}
}

func (sm *stateMachine) initRootArrayValue() (*listBuilder, error) {
	sm.pushContext(contextArray)
	expected := *sm.peekExpectedType()
	if kind := sm.classify(expected).kind; kind == kindJSON || kind == kindAnydata || kind == kindUnion {
		sm.arrayIndexes = append(sm.arrayIndexes, 0)
	}
	return sm.initArrayValue(expected)
}

func (sm *stateMachine) initArrayValue(expected semtypes.SemType) (*listBuilder, error) {
	ti := sm.classify(expected)
	switch ti.kind {
	case kindList:
		return sm.newListBuilder(ti), nil
	case kindJSON:
		return sm.newJSONListBuilder(), nil
	case kindAnydata:
		return sm.newAnydataListBuilder(), nil
	case kindUnion:
		sm.unionDepth++
		return sm.newJSONListBuilder(), nil
	default:
		return nil, invalidTypeError(sm.typeName(expected), "list type")
	}
}

func (sm *stateMachine) initNewMapValue() (*mappingBuilder, error) {
	parentContext := sm.peekContext()
	sm.pushContext(contextMap)
	expected := sm.peekExpectedType()
	if expected == nil {
		sm.pushFieldNames()
		return nil, nil
	}
	if sm.currentNode != nil {
		sm.nodes = append(sm.nodes, sm.currentNode)
	}
	return sm.checkTypeAndCreateMappingValue(*expected, parentContext)
}

func (sm *stateMachine) checkTypeAndCreateMappingValue(expected semtypes.SemType, parentContext parserContext) (*mappingBuilder, error) {
	ti := sm.classify(expected)
	switch ti.kind {
	case kindRecord, kindMap:
		fields, err := sm.mappingFields(ti, false)
		if err != nil {
			return nil, err
		}
		sm.updateExpectedType(fields, restTypeRef(ti))
		return sm.newMappingBuilder(ti), nil
	case kindJSON:
		sm.updateExpectedType(emptyFieldSet(), &expected)
		return sm.newJSONMappingBuilder(), nil
	case kindAnydata:
		sm.updateExpectedType(emptyFieldSet(), &expected)
		return sm.newAnydataMappingBuilder(), nil
	case kindUnion:
		sm.unionDepth++
		sm.pushFieldNames()
		return sm.newJSONMappingBuilder(), nil
	default:
		if parentContext == contextArray {
			return nil, invalidTypeError(sm.typeName(expected), "map type")
		}
		return nil, invalidTypeForFieldError(sm.currentFieldPath())
	}
}

func (sm *stateMachine) updateNextMapValue() error {
	next, err := sm.initNewMapValue()
	if err != nil {
		return err
	}
	if next != nil {
		sm.currentNode = next
	} else {
		sm.jsonFieldDepth++
	}
	return nil
}

func (sm *stateMachine) initNewArrayValue() (*listBuilder, error) {
	sm.pushContext(contextArray)
	expected := sm.peekExpectedType()
	if expected == nil {
		return nil, nil
	}
	current := sm.currentNode
	next, err := sm.initArrayValue(*expected)
	if err != nil {
		return nil, err
	}
	if current != nil {
		sm.nodes = append(sm.nodes, current)
	}
	return next, nil
}

func (sm *stateMachine) updateNextArrayValue() error {
	sm.arrayIndexes = append(sm.arrayIndexes, 0)
	next, err := sm.initNewArrayValue()
	if err != nil {
		return err
	}
	if next != nil {
		sm.currentNode = next
	}
	return nil
}

// updateMemberExpectedType is JsonCreator.updateExpectedType: it pushes the expected type of the next
// array element.
func (sm *stateMachine) updateMemberExpectedType() error {
	if sm.unionDepth > 0 {
		return nil
	}
	member, err := sm.memberType(sm.peekExpectedType(), sm.arrayIndexes[len(sm.arrayIndexes)-1])
	if err != nil {
		return err
	}
	sm.expectedTypes = append(sm.expectedTypes, member)
	return nil
}

func (sm *stateMachine) updateNextMapValueBasedOnExpType() error {
	if err := sm.updateMemberExpectedType(); err != nil {
		return err
	}
	return sm.updateNextMapValue()
}

func (sm *stateMachine) updateNextArrayValueBasedOnExpType() error {
	if err := sm.updateMemberExpectedType(); err != nil {
		return err
	}
	return sm.updateNextArrayValue()
}

func (sm *stateMachine) memberType(expected *semtypes.SemType, index int) (*semtypes.SemType, error) {
	if expected == nil {
		return nil, nil
	}
	ti := sm.classify(*expected)
	if ti.kind != kindList {
		return expected, nil
	}
	if index < ti.list.FixedLength() || !semtypes.IsNever(ti.list.Rest()) {
		member := ti.list.MemberAtInnerVal(index)
		return &member, nil
	}
	if !sm.allowDataProjection {
		return nil, arraySizeMismatchError()
	}
	return nil, nil
}

func (sm *stateMachine) handleFieldName(jsonFieldName string) error {
	if sm.jsonFieldDepth == 0 && sm.unionDepth == 0 {
		visited := sm.visitedFieldHierarchy[len(sm.visitedFieldHierarchy)-1]
		fields := sm.fieldHierarchy[len(sm.fieldHierarchy)-1]
		sm.currentField = visited[jsonFieldName]
		if sm.currentField == nil {
			sm.currentField = fields.remove(jsonFieldName)
		}
		var fieldType *semtypes.SemType
		if sm.currentField == nil {
			fieldType = sm.restTypes[len(sm.restTypes)-1]
			if fieldType != nil && fields.isRenamed(jsonFieldName) {
				return duplicateFieldError(jsonFieldName)
			}
		} else {
			visited[jsonFieldName] = sm.currentField
			jsonFieldName = sm.currentField.name
			fieldType = &sm.currentField.ty
		}
		sm.expectedTypes = append(sm.expectedTypes, fieldType)
		if !sm.allowDataProjection && fieldType == nil {
			return undefinedFieldError(jsonFieldName)
		}
	} else if sm.peekExpectedType() == nil {
		sm.expectedTypes = append(sm.expectedTypes, nil)
	}
	sm.pushFieldName(jsonFieldName)
	return nil
}

func (sm *stateMachine) processValue(isString bool) error {
	value := sm.value()
	var expected semtypes.SemType
	switch {
	case sm.unionDepth > 0 && isString:
		expected = semtypes.String
	case sm.unionDepth > 0:
		expected = sm.jsonTy
	default:
		popped := sm.popExpectedType()
		if popped == nil {
			return nil
		}
		expected = *popped
	}
	converted, err := sm.convertAndUpdateCurrentNode(value, expected, isString)
	if err != nil {
		return err
	}
	sm.checkNullAndUpdateCurrentNode(converted)
	return nil
}

func (sm *stateMachine) convertAndUpdateCurrentNode(value string, ty semtypes.SemType, isString bool) (any, error) {
	current := sm.currentNode
	if sm.nilAsOptionalField && !sm.isNilable(ty) && value == "null" && sm.currentField != nil && sm.currentField.optional {
		list, ok := current.(*listBuilder)
		if !ok {
			return nil, nil
		}
		// jBallerina leaves the member's filler value in place of a null member of an optional field's list.
		index := sm.arrayIndexes[len(sm.arrayIndexes)-1]
		if filler, ok := values.FillerValue(sm.tc, ty); ok {
			if list.accepts(index) {
				list.set(index, filler)
			}
			return current, nil
		}
	}

	var converted values.BalValue
	var err error
	if isString {
		converted, err = sm.convertStringToken(value, ty)
	} else {
		converted, err = sm.convertNonStringToken(value, ty)
	}
	if err != nil {
		var returned *returnedError
		if !errors.As(err, &returned) {
			return nil, err
		}
		if sm.currentField != nil {
			return nil, incompatibleValueForFieldError(value, sm.typeName(ty), sm.currentFieldPath())
		}
		return nil, incompatibleTypeError(sm.typeName(ty), value)
	}

	switch node := current.(type) {
	case *mappingBuilder:
		node.put(sm.popFieldName(), converted)
	case *listBuilder:
		if index := sm.arrayIndexes[len(sm.arrayIndexes)-1]; node.accepts(index) {
			node.set(index, converted)
		}
	default:
		return converted, nil
	}
	return current, nil
}

func (sm *stateMachine) checkNullAndUpdateCurrentNode(value any) {
	if value != nil {
		sm.currentNode = value
	}
}

// checkUnionAndFinalizeNonArrayObject and checkUnionAndFinalizeArrayObject pop the structure's expected type
// unless it is inside a union-typed structure that is still being read, whose members push no expected type.
func (sm *stateMachine) checkUnionAndFinalizeNonArrayObject() (parserState, error) {
	var state parserState
	var err error
	if sm.unionDepth > 0 {
		sm.popFieldNames()
		state, err = sm.finalizeObject()
	} else {
		state, err = sm.finalizeNonArrayObject()
	}
	if sm.unionDepth == 0 {
		sm.popExpectedType()
	}
	return state, err
}

func (sm *stateMachine) checkUnionAndFinalizeArrayObject() (parserState, error) {
	sm.arrayIndexes = sm.arrayIndexes[:len(sm.arrayIndexes)-1]
	state, err := sm.finalizeObject()
	if sm.unionDepth == 0 {
		sm.popExpectedType()
	}
	return state, err
}

func (sm *stateMachine) finalizeNonArrayObject() (parserState, error) {
	if sm.jsonFieldDepth > 0 {
		sm.jsonFieldDepth--
	}
	if len(sm.expectedTypes) > 0 && sm.peekExpectedType() == nil {
		sm.popContext()
		sm.popFieldNames()
		return sm.skippedValueEndState(), nil
	}
	remaining := sm.fieldHierarchy[len(sm.fieldHierarchy)-1]
	sm.fieldHierarchy = sm.fieldHierarchy[:len(sm.fieldHierarchy)-1]
	sm.visitedFieldHierarchy = sm.visitedFieldHierarchy[:len(sm.visitedFieldHierarchy)-1]
	sm.popFieldNames()
	sm.restTypes = sm.restTypes[:len(sm.restTypes)-1]
	node, _ := sm.currentNode.(*mappingBuilder)
	for _, field := range remaining.remaining() {
		if field.optional {
			continue
		}
		if sm.absentAsNilableType && sm.isNilable(field.ty) {
			if node != nil {
				node.put(field.name, nil)
			}
			continue
		}
		return stateDocEnd, requiredFieldNotPresentError(field.name)
	}
	return sm.finalizeObject()
}

func (sm *stateMachine) skippedValueEndState() parserState {
	if sm.peekContext() == contextMap {
		return stateFieldEnd
	}
	return stateArrayElementEnd
}

func (sm *stateMachine) finalizeObject() (parserState, error) {
	sm.popContext()
	if sm.unionDepth > 0 {
		sm.unionDepth--
		converted, err := sm.verifyAndConvertToUnion(sm.currentNode)
		if err != nil {
			return stateDocEnd, err
		}
		sm.currentNode = converted
	}
	if len(sm.expectedTypes) > 0 && sm.peekExpectedType() == nil {
		return sm.skippedValueEndState(), nil
	}
	if len(sm.nodes) == 0 {
		return stateDocEnd, nil
	}

	parent := sm.nodes[len(sm.nodes)-1]
	sm.nodes = sm.nodes[:len(sm.nodes)-1]
	child, err := sm.materialize(sm.currentNode)
	if err != nil {
		return stateDocEnd, err
	}
	sm.currentNode = parent
	switch parent := parent.(type) {
	case *mappingBuilder:
		parent.put(sm.popFieldName(), child)
		return stateFieldEnd, nil
	case *listBuilder:
		if index := sm.arrayIndexes[len(sm.arrayIndexes)-1]; parent.accepts(index) {
			parent.set(index, child)
		}
	}
	return stateArrayElementEnd, nil
}

// verifyAndConvertToUnion converts a structure read as plain json to the expected union type once the
// outermost structure under that union is complete.
func (sm *stateMachine) verifyAndConvertToUnion(node any) (any, error) {
	if sm.unionDepth > 0 {
		return node, nil
	}
	json, err := sm.materialize(node)
	if err != nil {
		return nil, err
	}
	return traverse(sm.typer, sm.projection, json, *sm.peekExpectedType(), false)
}
