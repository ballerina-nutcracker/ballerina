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

// Port of jBallerina's XmlTraversal: converts an XML value to a record or map type. It keeps jBallerina's
// stacks, including the places where a nested map shares its parent's field and rest-type stacks.

package native

import (
	"strings"

	"github.com/ballerina-nutcracker/ballerina/semtypes"
	"github.com/ballerina-nutcracker/ballerina/values"
)

// xmlSeq is a BXmlSequence: an element's children, or the source value when it is a sequence. The runtime
// represents empty content as an empty text item, which a Ballerina XML sequence never contains.
type xmlSeq []values.XMLValue

func flattenXML(v values.XMLValue) xmlSeq {
	items := []values.XMLValue{v}
	if seq, ok := v.(*values.XMLSequence); ok {
		items = seq.Children
	}
	var result xmlSeq
	for _, item := range items {
		if text, ok := item.(*values.XMLText); ok && text.Body == "" {
			continue
		}
		result = append(result, item)
	}
	return result
}

func childrenOf(e *values.XMLElement) xmlSeq {
	if e.Children == nil {
		return nil
	}
	return flattenXML(e.Children)
}

// xmlText is BXml.toString, which renders a text item with its markup characters escaped.
func xmlText(v values.XMLValue) string {
	return v.XMLString()
}

func isCommentOrPI(v values.XMLValue) bool {
	switch v.(type) {
	case *values.XMLComment, *values.XMLProcessingInstruction:
		return true
	}
	return false
}

func traverseXML(t *typer, opts sourceOptions, source values.XMLValue, target semtypes.SemType) (result values.BalValue, err error) {
	defer recoverAsParseError(&err)
	a := &analyzer{typer: t, sourceOptions: opts}
	if err := a.traverseToType(source, target); err != nil {
		return nil, err
	}
	return t.build(a.currentNode)
}

func (a *analyzer) traverseToType(source values.XMLValue, target semtypes.SemType) error {
	ti := a.classify(target)
	switch ti.kind {
	case kindRecord:
		return a.traverseWithRecord(source, ti)
	case kindMap:
		return a.traverseWithRecord(source, typeInfo{kind: kindRecord, ty: ti.ty, mapping: ti.mapping})
	case kindUnion:
		return a.traverseToUnion(source, ti)
	default:
		return invalidTypeError("record or map", a.typeName(target))
	}
}

func (a *analyzer) traverseToUnion(source values.XMLValue, ti typeInfo) error {
	saved := a.snapshot()
	for _, member := range ti.members {
		if semtypes.IsSubtypeSimple(member, semtypes.Error) {
			continue
		}
		if err := a.traverseToType(source, member); err == nil {
			return nil
		}
		a.resetFrom(saved)
	}
	return cannotConvertSourceError(a.typeName(ti.ty))
}

func (a *analyzer) traverseWithRecord(source values.XMLValue, ti typeInfo) error {
	a.currentNode = a.newRecordNode(ti)
	children, err := a.validateRootElement(source, ti)
	if err != nil {
		return err
	}
	if err := a.convertSequence(children, ti.ty); err != nil {
		return err
	}
	return a.validateRequiredFields(a.currentNode)
}

func (a *analyzer) validateRootElement(source values.XMLValue, ti typeInfo) (xmlSeq, error) {
	element, ok := source.(*values.XMLElement)
	if !ok {
		if seq, isSeq := source.(*values.XMLSequence); isSeq {
			filtered := filterEmptyOrCommentOrPI(flattenXML(seq))
			if len(filtered) == 1 {
				return a.validateRootElement(filtered[0], ti)
			}
		}
		return nil, xmlRootMissingError()
	}
	a.rootRecord = &ti
	elementName := a.qname(element.NamespaceURI, element.LocalName, element.Prefix, stateElement)
	if err := a.validateRootName(ti, elementName); err != nil {
		return nil, err
	}
	if err := a.validateTypeNamespace(element.Prefix, element.NamespaceURI, ti); err != nil {
		return nil, err
	}
	if err := a.updateExpectedTypeStacks(ti, false); err != nil {
		return nil, err
	}
	if err := a.handleAttributes(element, a.currentNode); err != nil {
		return nil, err
	}
	a.arrayIndexes = append(a.arrayIndexes, map[string]int{})
	return childrenOf(element), nil
}

func (a *analyzer) traverseItem(item values.XMLValue, ty semtypes.SemType) error {
	switch item := item.(type) {
	case *values.XMLElement:
		ti := a.classify(ty)
		if ti.isXML() || ti.kind == kindArray && a.classify(ti.element).isXML() {
			return a.convertXML(item, false)
		}
		return a.convertElement(item)
	case *values.XMLSequence:
		return a.convertSequence(flattenXML(item), ty)
	case *values.XMLText:
		return a.convertXML(item, true)
	}
	return nil
}

func (a *analyzer) convertXML(item values.XMLValue, isText bool) error {
	field := a.currentField
	mapValue := a.currentNode
	if field == nil {
		contentName := a.qname("", a.textFieldName, "", stateNotDefined)
		switch {
		case peek(a.fieldHierarchy).contains(contentName):
			field, _ = peek(a.fieldHierarchy).remove(contentName)
		case peek(a.visitedFieldHierarchy).contains(contentName):
			field, _ = peek(a.visitedFieldHierarchy).get(contentName)
		case a.peekRestType() != nil:
			field = &recordField{name: a.textFieldName, runtimeName: a.textFieldName, ty: *a.peekRestType()}
		case !a.allowDataProjection:
			return undefinedFieldError(a.textFieldName, a.rootRecordName())
		default:
			return nil
		}
	}
	if field == nil {
		return nil
	}

	fieldType := field.ty
	key := field.runtimeName
	existing := mapValue.get(key)
	_, existingIsArray := existing.(*arrayNode)
	var converted node
	found := false
	if ti := a.classify(fieldType); ti.kind == kindUnion {
		saved := a.snapshot()
		for _, member := range ti.members {
			if !existingIsArray && a.classify(member).kind == kindArray {
				continue
			}
			value, err := a.convertedXMLValue(item, isText, member)
			if err != nil {
				a.resetFrom(saved)
				continue
			}
			converted = value
			found = value != nil
			fieldType = member
			break
		}
		if !found {
			return fieldCannotConvertError(field.name, a.typeName(fieldType))
		}
	} else {
		value, err := a.convertedXMLValue(item, isText, fieldType)
		if err != nil {
			return err
		}
		converted = value
	}

	fieldInfo := a.classify(fieldType)
	if array, ok := existing.(*arrayNode); ok {
		if field.name == a.textFieldName {
			mapValue.put(key, converted)
			return nil
		}
		if fieldInfo.isAnydataOrJSON() {
			array.append(converted)
			return nil
		}
		index := peek(a.arrayIndexes)[key]
		if fieldInfo.size != -1 && fieldInfo.size <= index {
			return a.projectionCheck()
		}
		if a.classify(fieldInfo.element).kind == kindXMLText {
			if _, isElement := item.(*values.XMLElement); isElement {
				return cannotConvertError("xml<lang.xml:Text>", xmlText(item), a.typeName(fieldType))
			}
		}
		array.add(index, converted)
		return nil
	}
	if fieldInfo.kind == kindArray {
		return cannotConvertError("string", xmlText(item), a.typeName(fieldType))
	}
	mapValue.put(key, converted)
	return nil
}

// convertedXMLValue converts text content to the field type, or copies an XML item.
func (a *analyzer) convertedXMLValue(item values.XMLValue, isText bool, ty semtypes.SemType) (node, error) {
	if !isText {
		return copyXML(a.tc, item), nil
	}
	return a.convertStringToExpType(xmlText(item), ty)
}

func (a *analyzer) convertElement(element *values.XMLElement) error {
	elementName := a.qname(element.NamespaceURI, element.LocalName, element.Prefix, stateElement)
	fields := peek(a.fieldHierarchy)
	var field *recordField
	if peek(a.visitedFieldHierarchy).contains(elementName) {
		field, _ = peek(a.visitedFieldHierarchy).get(elementName)
		if field != nil && !a.isArrayValueAssignable(field.ty) {
			return foundArrayForNonArrayTypeError(a.typeName(field.ty), field.name)
		}
	} else if f, _ := fields.get(elementName); f != nil {
		field = f
		if a.classify(f.ty).kind != kindArray {
			fields.remove(elementName)
			if elementName.local != f.name {
				fields.remove(a.qname(nsNotDefined, f.name, "", stateNotDefined))
				for _, entry := range fields.entries() {
					if entry.value.name == f.name {
						fields.remove(entry.name)
					}
				}
			}
		}
	}
	a.currentField = field
	if field == nil {
		elementLocal := elementName.local
		rest := a.peekRestType()
		switch {
		case rest != nil:
			if fields.containsLocal(elementLocal) {
				return undefinedFieldError(elementLocal, a.rootRecordName())
			}
			return a.convertToRestType(element, *rest)
		case !a.allowDataProjection:
			return undefinedFieldError(elementLocal, a.rootRecordName())
		}
		return nil
	}
	peek(a.visitedFieldHierarchy).put(elementName, field)
	if !a.isSupportedType(field.ty) {
		return unsupportedTypeError()
	}
	return a.convertToFieldType(element, field, field.ty, a.currentNode)
}

func (a *analyzer) isArrayValueAssignable(ty semtypes.SemType) bool {
	ti := a.classify(ty)
	if ti.kind == kindUnion {
		for _, member := range ti.members {
			if a.classify(member).isArrayValueAssignable() {
				return true
			}
		}
	}
	return ti.isArrayValueAssignable()
}

func (a *analyzer) convertToFieldType(element *values.XMLElement, field *recordField, ty semtypes.SemType, mapValue *mapNode) error {
	ti := a.classify(ty)
	switch ti.kind {
	case kindRecord:
		return a.convertToRecordType(element, ty, field.runtimeName, ti, mapValue)
	case kindArray:
		return a.convertToArrayType(element, field.runtimeName, ti, mapValue)
	case kindMap:
		return a.convertToMapType(element, ty, ty, field.runtimeName, mapValue)
	case kindJSON, kindAnydata:
		a.updateNextMap(ty)
		a.arrayIndexes = append(a.arrayIndexes, map[string]int{})
		if err := a.convertToRestType(element, ty); err != nil {
			return err
		}
		a.popExpectedTypeStacks()
		return nil
	case kindUnion:
		return a.convertFieldTypeToUnion(element, field, ti, mapValue)
	}
	if ti.isXML() {
		return a.traverseItem(element, ty)
	}
	return a.convertSequence(childrenOf(element), ty)
}

func (a *analyzer) convertFieldTypeToUnion(element *values.XMLElement, field *recordField, ti typeInfo, mapValue *mapNode) error {
	saved := a.snapshot()
	for _, member := range ti.members {
		if semtypes.IsSubtypeSimple(member, semtypes.Error) {
			continue
		}
		if err := a.convertToFieldType(element, field, member, mapValue); err == nil {
			return nil
		}
		a.resetFrom(saved)
		mapValue.put(field.runtimeName, nil)
	}
	return fieldCannotConvertError(field.name, a.typeName(ti.ty))
}

func (a *analyzer) convertToArrayType(element *values.XMLElement, key string, arrayInfo typeInfo, mapValue *mapNode) error {
	if a.classify(arrayInfo.element).isAnydataOrJSON() {
		return a.convertToRestType(element, arrayInfo.ty)
	}
	if mapValue.get(key) == nil {
		peek(a.arrayIndexes)[key] = 0
		mapValue.put(key, a.arrayValueFor(arrayInfo.ty))
	} else {
		peek(a.arrayIndexes)[key]++
	}
	return a.convertToArrayMemberType(element, key, arrayInfo, arrayInfo.element, mapValue)
}

func (a *analyzer) convertToArrayMemberType(element *values.XMLElement, key string, arrayInfo typeInfo, elementType semtypes.SemType, mapValue *mapNode) error {
	ti := a.classify(elementType)
	switch ti.kind {
	case kindRecord:
		return a.convertToRecordType(element, arrayInfo.ty, key, ti, mapValue)
	case kindMap:
		return a.convertToMapType(element, arrayInfo.ty, elementType, key, mapValue)
	case kindUnion:
		saved := a.snapshot()
		for _, member := range ti.members {
			if semtypes.IsSubtypeSimple(member, semtypes.Error) {
				continue
			}
			if err := a.convertToArrayMemberType(element, key, arrayInfo, member, mapValue); err == nil {
				return nil
			}
			a.resetFrom(saved)
		}
		return fieldCannotConvertError(key, a.typeName(arrayInfo.ty))
	case kindXML, kindXMLElement, kindXMLText:
		return a.traverseItem(element, arrayInfo.ty)
	}
	return a.convertSequence(childrenOf(element), arrayInfo.ty)
}

func (a *analyzer) convertToRecordType(element *values.XMLElement, fieldType semtypes.SemType, key string, recordInfo typeInfo, mapValue *mapNode) error {
	next, err := a.updateNextRecord(element, recordInfo, key, fieldType, mapValue)
	if err != nil {
		return err
	}
	a.currentNode = next
	previous := a.rootRecord
	a.rootRecord = &recordInfo
	if err := a.convertSequence(childrenOf(element), fieldType); err != nil {
		return err
	}
	if err := a.validateRequiredFields(a.currentNode); err != nil {
		return err
	}
	a.popExpectedTypeStacks()
	a.rootRecord = previous
	a.currentNode = pop(&a.nodesStack).(*mapNode)
	return nil
}

func (a *analyzer) convertToMapType(element *values.XMLElement, fieldType, elementType semtypes.SemType, key string, mapValue *mapNode) error {
	a.updateNextMap(elementType)
	next, err := a.updateNextMappingValue(elementType, key, fieldType, mapValue)
	if err != nil {
		return err
	}
	a.currentNode = next
	if err := a.convertSequence(childrenOf(element), fieldType); err != nil {
		return err
	}
	if err := a.validateRequiredFields(a.currentNode); err != nil {
		return err
	}
	a.popExpectedTypeStacks()
	a.currentNode = pop(&a.nodesStack).(*mapNode)
	return nil
}

func (a *analyzer) updateNextMap(ty semtypes.SemType) {
	rest := ty
	if ti := a.classify(ty); ti.kind == kindMap {
		rest = ti.mapping.Rest()
	}
	a.restTypes = append(a.restTypes, &rest)
	a.fieldHierarchy = append(a.fieldHierarchy, newQNameMap[*recordField]())
	a.visitedFieldHierarchy = append(a.visitedFieldHierarchy, newQNameMap[*recordField]())
	a.attributeHierarchy = append(a.attributeHierarchy, newQNameMap[*recordField]())
}

func (a *analyzer) updateNextRecord(element *values.XMLElement, recordInfo typeInfo, key string, fieldType semtypes.SemType, mapValue *mapNode) (*mapNode, error) {
	if err := a.updateExpectedTypeStacks(recordInfo, false); err != nil {
		return nil, err
	}
	next, err := a.updateNextMappingValue(recordInfo.ty, key, fieldType, mapValue)
	if err != nil {
		return nil, err
	}
	if err := a.validateTypeNamespaceForNextRecord(element.Prefix, element.NamespaceURI, recordInfo, a.rootRecord, key); err != nil {
		return nil, err
	}
	if err := a.handleAttributes(element, next); err != nil {
		return nil, err
	}
	return next, nil
}

func (a *analyzer) updateNextMappingValue(ty semtypes.SemType, key string, fieldType semtypes.SemType, mapValue *mapNode) (*mapNode, error) {
	a.currentField = nil
	var next *mapNode
	switch ti := a.classify(ty); ti.kind {
	case kindRecord:
		next = a.newRecordNode(ti)
	case kindMap:
		next = newMapNode(ty)
	case kindAnydata, kindJSON:
		next = newMapNode(a.mapType(ty))
	default:
		return nil, unsupportedTypeError()
	}
	if array, ok := mapValue.get(key).(*arrayNode); ok {
		arrayInfo := a.classify(fieldType)
		index := peek(a.arrayIndexes)[key]
		if arrayInfo.size == -1 || index < arrayInfo.size {
			array.add(index, next)
		} else if err := a.projectionCheck(); err != nil {
			return nil, err
		}
	} else {
		mapValue.put(key, next)
	}
	a.nodesStack = append(a.nodesStack, mapValue)
	a.arrayIndexes = append(a.arrayIndexes, map[string]int{})
	return next, nil
}

func (a *analyzer) convertToRestType(element *values.XMLElement, restType semtypes.SemType) error {
	name := element.LocalName
	a.currentField = &recordField{name: name, runtimeName: name, ty: restType, optional: true}
	return a.checkRestTypeAndConvert(element, name, restType, restType, a.currentNode)
}

func (a *analyzer) checkRestTypeAndConvert(element *values.XMLElement, name string, restType, elementType semtypes.SemType, mapValue *mapNode) error {
	ti := a.classify(elementType)
	switch ti.kind {
	case kindRecord:
		return a.convertToRecordType(element, restType, name, ti, mapValue)
	case kindArray:
		indexes := peek(a.arrayIndexes)
		if _, ok := indexes[name]; !ok {
			indexes[name] = 0
			mapValue.put(name, a.arrayValueFor(restType))
		} else {
			indexes[name]++
		}
		return a.checkRestTypeAndConvert(element, name, restType, a.classify(restType).element, mapValue)
	case kindUnion:
		return a.checkRestTypeAndConvertForUnion(element, name, restType, ti, mapValue)
	}
	if mapValue.containsKey(name) && mapValue.get(name) != nil {
		return a.handleArrayValueForRestType(element, name, restType, mapValue)
	}
	if isNextElementContent(element) {
		return a.convertContentToRestType(element, name, restType, mapValue)
	}
	if !a.classify(restType).isAnydataOrJSON() {
		return expectedAnydataOrJSONError()
	}
	return a.convertToJSONOrAnydataAsRestType(element, name, restType, mapValue)
}

func (a *analyzer) checkRestTypeAndConvertForUnion(element *values.XMLElement, name string, restType semtypes.SemType, ti typeInfo, mapValue *mapNode) error {
	restIsUnion := a.classify(restType).kind == kindUnion
	restIsArray := a.classify(restType).kind == kindArray
	saved := a.snapshot()
	for _, member := range ti.members {
		if semtypes.IsSubtypeSimple(member, semtypes.Error) {
			continue
		}
		memberRest := restType
		if restIsUnion {
			memberRest = member
		}
		if err := a.checkRestTypeAndConvert(element, name, memberRest, member, mapValue); err == nil {
			return nil
		}
		a.resetFrom(saved)
		if !restIsArray {
			mapValue.put(name, nil)
		}
	}
	return fieldCannotConvertError(name, a.typeName(ti.ty))
}

func (a *analyzer) handleArrayValueForRestType(element *values.XMLElement, name string, restType semtypes.SemType, mapValue *mapNode) error {
	current := mapValue.get(name)
	array, isArray := current.(*arrayNode)
	if !isArray {
		if !a.isArrayValueAssignable(restType) {
			return foundArrayForNonArrayTypeError(a.typeName(restType), name)
		}
		array = a.arrayValueFor(restType)
		array.append(current)
		mapValue.put(name, array)
	}
	if isNextElementContent(element) {
		if len(elementAttributes(element)) > 0 {
			next := newMapNode(a.mapTypeFromConstraint(restType))
			a.handleAttributesRest(element, next, restType)
			if !next.isEmpty() {
				array.append(next)
				a.currentField = &recordField{name: a.textFieldName, runtimeName: a.textFieldName, ty: restType}
				a.nodesStack = append(a.nodesStack, a.currentNode)
				a.currentNode = next
				if err := a.convertSequence(childrenOf(element), restType); err != nil {
					return err
				}
				a.currentNode = pop(&a.nodesStack).(*mapNode)
				return nil
			}
		}
		return a.convertSequence(childrenOf(element), restType)
	}

	next := newMapNode(a.mapTypeFromConstraint(restType))
	restInfo := a.classify(restType)
	if restInfo.isAnydataOrJSON() {
		array.append(next)
	} else {
		index := peek(a.arrayIndexes)[name]
		if restInfo.size != -1 && restInfo.size <= index {
			if err := a.projectionCheck(); err != nil {
				return err
			}
		} else {
			array.add(index, next)
		}
	}
	a.nodesStack = append(a.nodesStack, a.currentNode)
	a.currentNode = next
	a.handleAttributesRest(element, next, restType)
	a.fieldHierarchy = append(a.fieldHierarchy, newQNameMap[*recordField]())
	a.visitedFieldHierarchy = append(a.visitedFieldHierarchy, newQNameMap[*recordField]())
	a.arrayIndexes = append(a.arrayIndexes, map[string]int{})
	memberType := restType
	if restInfo.kind == kindArray {
		memberType = restInfo.element
	}
	a.restTypes = append(a.restTypes, &memberType)
	if err := a.convertSequence(childrenOf(element), memberType); err != nil {
		return err
	}
	pop(&a.fieldHierarchy)
	pop(&a.visitedFieldHierarchy)
	pop(&a.restTypes)
	pop(&a.arrayIndexes)
	a.currentNode = pop(&a.nodesStack).(*mapNode)
	return nil
}

func (a *analyzer) convertContentToRestType(element *values.XMLElement, name string, restType semtypes.SemType, mapValue *mapNode) error {
	if len(elementAttributes(element)) > 0 {
		next := newMapNode(a.mapTypeFromConstraint(restType))
		a.handleAttributesRest(element, next, restType)
		if !next.isEmpty() {
			mapValue.put(name, next)
			a.currentField = &recordField{name: a.textFieldName, runtimeName: a.textFieldName, ty: restType}
			a.nodesStack = append(a.nodesStack, a.currentNode)
			a.currentNode = next
			if err := a.convertSequence(childrenOf(element), restType); err != nil {
				return err
			}
			a.currentNode = pop(&a.nodesStack).(*mapNode)
			return nil
		}
	}
	return a.convertSequence(childrenOf(element), restType)
}

func (a *analyzer) convertToJSONOrAnydataAsRestType(element *values.XMLElement, name string, restType semtypes.SemType, mapValue *mapNode) error {
	next := newMapNode(a.mapTypeFromConstraint(restType))
	mapValue.put(name, next)
	a.nodesStack = append(a.nodesStack, a.currentNode)
	a.currentNode = next
	a.handleAttributesRest(element, next, restType)
	if err := a.convertSequence(childrenOf(element), restType); err != nil {
		return err
	}
	a.currentNode = pop(&a.nodesStack).(*mapNode)
	return nil
}

func isNextElementContent(element *values.XMLElement) bool {
	for _, child := range childrenOf(element) {
		if isCommentOrPI(child) || strings.TrimSpace(xmlText(child)) == "" {
			continue
		}
		if _, ok := child.(*values.XMLText); ok {
			return true
		}
	}
	return false
}

func filterEmptyOrCommentOrPI(seq xmlSeq) xmlSeq {
	var filtered xmlSeq
	for _, item := range seq {
		if isCommentOrPI(item) {
			continue
		}
		if text := xmlText(item); text == "" || strings.TrimSpace(text) != "" {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func (a *analyzer) convertSequence(seq xmlSeq, ty semtypes.SemType) error {
	filtered := filterEmptyOrCommentOrPI(seq)
	switch len(filtered) {
	case 0:
		if ti := a.classify(ty); ti.kind != kindRecord {
			if field := a.currentField; field != nil && ti.isNonNilablePrimitive() && !field.optional {
				return fieldCannotConvertError(field.name, a.typeName(field.ty))
			}
		}
		return nil
	case 1:
		return a.traverseItem(filtered[0], ty)
	}
	if allText(filtered) {
		if !a.classify(ty).isStringValueAssignable() {
			return invalidTypeError(a.typeName(ty), "string")
		}
		var text strings.Builder
		for _, item := range filtered {
			text.WriteString(item.(*values.XMLText).Body)
		}
		return a.convertXML(values.NewXMLText(text.String()), true)
	}
	for _, item := range filtered {
		if err := a.traverseItem(item, ty); err != nil {
			return err
		}
	}
	return nil
}

func allText(seq xmlSeq) bool {
	for _, item := range seq {
		if _, ok := item.(*values.XMLText); !ok {
			return false
		}
	}
	return true
}

// xmlAttribute is one entry of an element's attribute map, keyed by expanded name.
type xmlAttribute struct {
	key   string
	value string
}

func elementAttributes(element *values.XMLElement) []xmlAttribute {
	if element.Attributes == nil {
		return nil
	}
	var attributes []xmlAttribute
	for _, key := range element.Attributes.Keys() {
		value, _ := element.Attributes.Get(key)
		text, _ := value.(string)
		attributes = append(attributes, xmlAttribute{key: key, value: text})
	}
	return attributes
}

const xmlnsKeyPrefix = "{" + values.XMLNSNamespaceURI + "}"

func namespacePrefixes(attributes []xmlAttribute) map[string]string {
	prefixes := map[string]string{}
	for _, attribute := range attributes {
		if prefix, ok := strings.CutPrefix(attribute.key, xmlnsKeyPrefix); ok {
			prefixes[attribute.value] = prefix
		}
	}
	return prefixes
}

func (a *analyzer) attributeQName(prefixes map[string]string, key, attributePrefix string) qname {
	if end := strings.LastIndex(key, "}"); end > 0 {
		ns := key[1:end]
		return a.qname(ns, attributePrefix+key[end+1:], prefixes[ns], stateNotDefined)
	}
	return a.qname("", attributePrefix+key, "", stateNotDefined)
}

func innerElementNames(element *values.XMLElement) map[string]bool {
	names := map[string]bool{}
	for _, child := range childrenOf(element) {
		if e, ok := child.(*values.XMLElement); ok {
			names[e.QualifiedName()] = true
		}
	}
	return names
}

func (a *analyzer) handleAttributes(element *values.XMLElement, current *mapNode) error {
	inner := innerElementNames(element)
	attributes := elementAttributes(element)
	prefixes := namespacePrefixes(attributes)
	for _, attribute := range attributes {
		if strings.HasPrefix(attribute.key, xmlnsKeyPrefix) {
			continue
		}
		name := a.attributeQName(prefixes, attribute.key, a.attributePrefix)
		field, _ := peek(a.attributeHierarchy).remove(name)
		if field == nil {
			if inner[name.local] {
				continue
			}
			if peek(a.visitedFieldHierarchy).contains(name) {
				field, _ = peek(a.visitedFieldHierarchy).get(name)
			} else {
				field, _ = peek(a.fieldHierarchy).remove(name)
				peek(a.visitedFieldHierarchy).put(name, field)
			}
		}
		if field == nil {
			if a.allowDataProjection {
				continue
			}
			return undefinedFieldError(name.local, a.rootRecordName())
		}
		if !a.isSupportedTypeForAttributes(field.ty) {
			return cannotConvertAttributeToArrayError(field.name, a.typeName(field.ty))
		}
		if converted, err := a.convertStringToExpType(attribute.value, field.ty); err == nil {
			current.put(field.runtimeName, converted)
		}
	}
	return nil
}

func (a *analyzer) handleAttributesRest(element *values.XMLElement, current *mapNode, restType semtypes.SemType) {
	inner := innerElementNames(element)
	attributes := elementAttributes(element)
	prefixes := namespacePrefixes(attributes)
	for _, attribute := range attributes {
		if strings.HasPrefix(attribute.key, xmlnsKeyPrefix) {
			continue
		}
		name := a.attributeQName(prefixes, attribute.key, "")
		if inner[name.local] {
			continue
		}
		if converted, err := a.convertStringToExpType(attribute.value, restType); err == nil {
			current.put(name.local, converted)
		}
	}
}

// copyXML is BXml.copy: a deep copy of the item.
func copyXML(tc semtypes.Context, item values.XMLValue) values.XMLValue {
	switch item := item.(type) {
	case *values.XMLElement:
		var entries []values.MapEntry
		for _, attribute := range elementAttributes(item) {
			entries = append(entries, values.MapEntry{Key: attribute.key, Value: attribute.value})
		}
		var children []values.XMLValue
		for _, child := range childrenOf(item) {
			children = append(children, copyXML(tc, child))
		}
		attributes := values.NewXMLStringMap(tc, false, entries)
		return values.NewXMLElement(tc, item.Prefix, item.LocalName, item.NamespaceURI, attributes, nil,
			values.NewNormalizedXMLSequence(children), false)
	case *values.XMLSequence:
		var children []values.XMLValue
		for _, child := range item.Children {
			children = append(children, copyXML(tc, child))
		}
		return values.NewNormalizedXMLSequence(children)
	case *values.XMLComment:
		return values.NewXMLComment(item.Body, false)
	case *values.XMLProcessingInstruction:
		return values.NewXMLProcessingInstruction(item.Target, item.Data, false)
	default:
		return item
	}
}
