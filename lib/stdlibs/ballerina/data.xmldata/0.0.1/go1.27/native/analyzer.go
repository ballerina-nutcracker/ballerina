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

// The stacks jBallerina's XmlAnalyzerMetaData keeps while converting, shared by the parser and the traversal.

package native

import (
	"slices"

	"github.com/ballerina-nutcracker/ballerina/semtypes"
)

type sourceOptions struct {
	attributePrefix     string
	textFieldName       string
	allowDataProjection bool
	useSemanticEquality bool
}

type analyzer struct {
	*typer
	sourceOptions
	attributeHierarchy    []*qnameMap[*recordField]
	arrayIndexes          []map[string]int
	nodesStack            []node
	fieldHierarchy        []*qnameMap[*recordField]
	visitedFieldHierarchy []*qnameMap[*recordField]
	// restTypes holds nil where jBallerina pushes a null rest type for a closed record.
	restTypes       []*semtypes.SemType
	rootRecord      *typeInfo
	currentField    *recordField
	currentNode     *mapNode
	recordTypeStack []*typeInfo
}

// snapshot is XmlAnalyzerData.copy: the stacks are copied but share their maps, and the current node is not
// part of it.
func (a *analyzer) snapshot() analyzer {
	copied := *a
	copied.attributeHierarchy = slices.Clone(a.attributeHierarchy)
	copied.arrayIndexes = slices.Clone(a.arrayIndexes)
	copied.nodesStack = slices.Clone(a.nodesStack)
	copied.fieldHierarchy = slices.Clone(a.fieldHierarchy)
	copied.visitedFieldHierarchy = slices.Clone(a.visitedFieldHierarchy)
	copied.restTypes = slices.Clone(a.restTypes)
	return copied
}

// resetFrom is XmlAnalyzerData.resetFrom; it leaves the current node as the failed attempt left it.
func (a *analyzer) resetFrom(s analyzer) {
	currentNode := a.currentNode
	recordTypeStack := a.recordTypeStack
	*a = s
	a.attributeHierarchy = slices.Clone(s.attributeHierarchy)
	a.arrayIndexes = slices.Clone(s.arrayIndexes)
	a.nodesStack = slices.Clone(s.nodesStack)
	a.fieldHierarchy = slices.Clone(s.fieldHierarchy)
	a.visitedFieldHierarchy = slices.Clone(s.visitedFieldHierarchy)
	a.restTypes = slices.Clone(s.restTypes)
	a.currentNode = currentNode
	a.recordTypeStack = recordTypeStack
}

func peek[T any](stack []T) T {
	return stack[len(stack)-1]
}

func pop[T any](stack *[]T) T {
	top := (*stack)[len(*stack)-1]
	*stack = (*stack)[:len(*stack)-1]
	return top
}

func (a *analyzer) peekRestType() *semtypes.SemType {
	return peek(a.restTypes)
}

func (a *analyzer) qname(ns, local, prefix string, state attributeState) qname {
	return newQNameWithState(ns, local, prefix, state, a.useSemanticEquality)
}

func (a *analyzer) rootRecordName() string {
	if a.rootRecord == nil {
		return "null"
	}
	return a.typeName(a.rootRecord.ty)
}

// updateExpectedTypeStacks pushes the attributes, fields and rest type of a record.
func (a *analyzer) updateExpectedTypeStacks(ti typeInfo, parserFlavour bool) error {
	attributes := a.attributesOf(ti, a.useSemanticEquality)
	a.attributeHierarchy = append(a.attributeHierarchy, attributes)
	fields, err := a.fieldsOf(ti, attributes, a.useSemanticEquality, parserFlavour)
	if err != nil {
		return err
	}
	a.fieldHierarchy = append(a.fieldHierarchy, fields)
	a.visitedFieldHierarchy = append(a.visitedFieldHierarchy, newQNameMap[*recordField]())
	a.restTypes = append(a.restTypes, restTypeRef(ti))
	return nil
}

func restTypeRef(ti typeInfo) *semtypes.SemType {
	if rest, ok := restTypeOf(ti); ok {
		return &rest
	}
	return nil
}

func (a *analyzer) popMappingTypeStacks() {
	pop(&a.fieldHierarchy)
	pop(&a.visitedFieldHierarchy)
	pop(&a.restTypes)
}

func (a *analyzer) popExpectedTypeStacks() {
	a.popMappingTypeStacks()
	pop(&a.attributeHierarchy)
	pop(&a.arrayIndexes)
}

// validateRequiredFields is DataUtils.validateRequiredFields.
func (a *analyzer) validateRequiredFields(current *mapNode) error {
	for _, entry := range peek(a.fieldHierarchy).entries() {
		field := entry.value
		if field == nil {
			continue
		}
		if ti := a.classify(field.ty); ti.kind == kindArray {
			if ti.size == -1 {
				continue
			}
			switch current.get(field.runtimeName).(type) {
			case *arrayNode:
				continue
			case nil:
			default:
				return arraySizeMismatchError()
			}
		}
		if !field.optional {
			return requiredFieldNotPresentError(field.name)
		}
	}
	return a.validateRequiredAttributes()
}

func (a *analyzer) validateRequiredAttributes() error {
	for _, entry := range peek(a.attributeHierarchy).entries() {
		if entry.value != nil && !entry.value.optional {
			return requiredAttributeNotPresentError(entry.value.name)
		}
	}
	return nil
}

func (a *analyzer) projectionCheck() error {
	if a.allowDataProjection {
		return nil
	}
	return arraySizeMismatchError()
}
