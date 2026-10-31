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

// Port of jBallerina's XmlParser: converts XML text to a record type while reading it event by event. When
// the target type contains a union of structures it parses the text into an XML value and traverses that
// instead, as jBallerina does.

package native

import (
	"strings"

	"github.com/ballerina-nutcracker/ballerina/semtypes"
	"github.com/ballerina-nutcracker/ballerina/values"
)

const contentField = "#content"

type xmlParser struct {
	analyzer
	reader           *staxReader
	restFieldsPoints []*qname
	parents          []*qnameMap[bool]
	siblings         *qnameMap[bool]
}

// parseXML recovers from a panic as XmlParser.parse catches Throwable: the parser keeps jBallerina's stack
// handling, so an input that breaks a stack invariant fails the call instead of the program.
func parseXML(t *typer, opts sourceOptions, input string, target semtypes.SemType) (result values.BalValue, err error) {
	defer recoverAsParseError(&err)
	if t.containsUnionType(target) {
		source, err := values.ParseAsXMLValue(t.tc, input, values.XMLStrictContentMode)
		if err != nil {
			return nil, xmlParseError(err.Error())
		}
		return traverseXML(t, opts, source, target)
	}
	ti := t.classify(target)
	switch ti.kind {
	case kindRecord:
	case kindMap:
		ti.kind = kindRecord
	default:
		return nil, invalidTypeError("record", "")
	}
	p := &xmlParser{
		analyzer: analyzer{typer: t, sourceOptions: opts},
		reader:   newStaxReader(input),
		siblings: newQNameMap[bool](),
	}
	if err := p.parse(ti); err != nil {
		if _, ok := err.(*xmlError); ok {
			return nil, err
		}
		return nil, newXMLError("failed to parse xml: %s", err.Error())
	}
	return t.build(p.currentNode)
}

func (p *xmlParser) parse(root typeInfo) error {
	p.rootRecord = &root
	if err := p.parseRootElement(); err != nil {
		return err
	}
	readNext := false
	for p.reader.hasNext() {
		event := p.reader.eventType()
		if !readNext {
			var err error
			if event, err = p.reader.next(); err != nil {
				return err
			}
		}
		var err error
		if readNext, err = p.parseXMLElements(event); err != nil {
			return err
		}
	}
	return nil
}

func (p *xmlParser) parseXMLElements(event staxEvent) (bool, error) {
	switch event {
	case eventStartElement:
		return false, p.readElement()
	case eventEndElement:
		return false, p.endElement()
	case eventCData:
		return false, p.readText(true)
	case eventCharacters:
		return true, p.readText(false)
	case eventEndDocument:
		return false, p.buildDocument()
	}
	return false, nil
}

func (p *xmlParser) parseRecordRest(startElementName string) error {
	readNext := false
	for p.reader.hasNext() {
		event := p.reader.eventType()
		if !readNext {
			var err error
			if event, err = p.reader.next(); err != nil {
				return err
			}
		}
		if event == eventEndElement && p.reader.name.local == startElementName {
			if err := p.validateRequiredFields(); err != nil {
				return err
			}
			p.popExpectedTypeStacks()
			break
		}
		var err error
		if readNext, err = p.parseXMLElements(event); err != nil {
			return err
		}
	}
	return nil
}

func (p *xmlParser) parseRootElement() error {
	if p.reader.hasNext() {
		event, err := p.reader.next()
		if err != nil {
			return err
		}
		if event == eventComment || event == eventProcessingInstruction {
			return p.parseRootElement()
		}
		if event != eventStartElement {
			return xmlRootMissingError()
		}
	}
	root := *p.rootRecord
	p.currentNode = p.newRecordNode(root)
	elementName := p.elementName()
	if err := p.validateRootName(root, elementName); err != nil {
		return err
	}
	if err := p.validateTypeNamespace(elementName.prefix, elementName.ns, root); err != nil {
		return err
	}
	p.recordTypeStack = append(p.recordTypeStack, &root)
	if err := p.updateExpectedTypeStacks(root, true); err != nil {
		return err
	}
	if err := p.handleAttributes(); err != nil {
		return err
	}
	p.arrayIndexes = append(p.arrayIndexes, map[string]int{})
	return nil
}

func (p *xmlParser) elementName() qname {
	name := p.reader.name
	return p.qname(name.ns, name.local, name.prefix, stateElement)
}

// readCharacters is handleTruncatedCharacters: it joins adjacent character events, noting a comment between
// them, and leaves the reader on the event after them.
func (p *xmlParser) readCharacters() (string, bool, error) {
	var text strings.Builder
	commentInMiddle := false
	for p.reader.eventType() == eventCharacters {
		text.WriteString(p.reader.text)
		event, err := p.reader.next()
		if err != nil {
			return "", false, err
		}
		if event == eventComment {
			commentInMiddle = true
			if _, err := p.reader.next(); err != nil {
				return "", false, err
			}
		}
	}
	return text.String(), commentInMiddle, nil
}

func (p *xmlParser) readTextValue(isCData bool) (string, bool, error) {
	if isCData {
		return p.reader.text, false, nil
	}
	return p.readCharacters()
}

func (p *xmlParser) readText(isCData bool) error {
	field := p.currentField
	text, commentInMiddle, err := p.readTextValue(isCData)
	if err != nil {
		return err
	}
	if strings.TrimSpace(text) == "" {
		return nil
	}
	if field == nil {
		contentName := p.qname("", p.textFieldName, "", stateNotDefined)
		if !peek(p.fieldHierarchy).contains(contentName) {
			switch {
			case peek(p.visitedFieldHierarchy).contains(contentName):
				field, _ = peek(p.visitedFieldHierarchy).get(contentName)
			case !p.allowDataProjection:
				return undefinedFieldError(p.textFieldName, p.rootRecordName())
			default:
				return nil
			}
		} else {
			field, _ = peek(p.fieldHierarchy).remove(contentName)
		}
		if field == nil {
			return nil
		}
	}

	key := field.runtimeName
	ti := p.classify(field.ty)
	if commentInMiddle && !ti.isStringValueAssignable() {
		return invalidTypeError(p.typeName(field.ty), "string")
	}
	if array, ok := p.currentNode.get(key).(*arrayNode); ok && !ti.isAnydataOrJSON() {
		if field.name == p.textFieldName {
			return p.putRestValue(p.currentNode, key, text, field.ty)
		}
		index := peek(p.arrayIndexes)[key]
		if ti.size != -1 && ti.size <= index {
			return p.projectionCheck()
		}
		converted, err := p.convertStringToRestExpType(text, field.ty)
		if err != nil {
			return err
		}
		array.add(index, converted)
		return nil
	}
	switch ti.kind {
	case kindRecord:
		return p.handleContentFieldInRecordType(ti, text)
	case kindArray:
		return p.addTextToCurrentNodeIfExpTypeIsArray(ti, key, text)
	case kindAnydata, kindJSON:
		if len(p.nodesStack) == 0 {
			return newXMLError("failed to parse xml")
		}
		parent := pop(&p.nodesStack).(*mapNode)
		return p.convertTextAndUpdateCurrentNode(p.currentNode, parent, key, text, field.ty)
	}
	return p.putRestValue(p.currentNode, key, text, field.ty)
}

func (p *xmlParser) putRestValue(target *mapNode, key, text string, ty semtypes.SemType) error {
	converted, err := p.convertStringToRestExpType(text, ty)
	if err != nil {
		return err
	}
	target.put(key, converted)
	return nil
}

func (p *xmlParser) convertTextAndUpdateCurrentNode(current, parent *mapNode, key, text string, restType semtypes.SemType) error {
	existing := current.get(key)
	converted, err := p.convertStringToRestExpType(text, restType)
	if err != nil {
		return err
	}
	if existing == nil && !current.isEmpty() {
		current.put(contentField, converted)
	} else if array, ok := parent.get(key).(*arrayNode); ok {
		array.add(array.length()-1, converted)
	} else {
		parent.put(key, converted)
	}
	p.currentNode = parent
	p.popExpectedTypeStacks()
	p.updateSiblingAndRootRecord()
	return nil
}

func (p *xmlParser) addTextToCurrentNodeIfExpTypeIsArray(arrayInfo typeInfo, key, text string) error {
	elementInfo := p.classify(arrayInfo.element)
	switch {
	case elementInfo.kind == kindRecord:
		return p.handleContentFieldInRecordType(elementInfo, text)
	case elementInfo.isAnydataOrJSON():
		parent := peek(p.nodesStack).(*mapNode)
		array, _ := parent.get(key).(*arrayNode)
		index := p.arrayIndexes[len(p.arrayIndexes)-2][key]
		if arrayInfo.size != -1 && index >= arrayInfo.size {
			return p.projectionCheck()
		}
		converted, err := p.convertStringToRestExpType(text, arrayInfo.ty)
		if err != nil {
			return err
		}
		array.add(index, converted)
	}
	return nil
}

func (p *xmlParser) handleContentFieldInRecordType(recordInfo typeInfo, text string) error {
	p.popExpectedTypeStacks()
	p.updateSiblingAndRootRecord()
	for _, field := range p.recordFields(recordInfo) {
		if strings.Contains(field.name, p.textFieldName) {
			converted, err := p.parserConvertStringToExpType(text, field.ty)
			if err != nil {
				return err
			}
			p.currentNode.put(field.runtimeName, converted)
			p.currentNode = pop(&p.nodesStack).(*mapNode)
			return nil
		}
	}
	rest, ok := restTypeOf(recordInfo)
	if !ok {
		return nil
	}
	if err := p.putRestValue(p.currentNode, p.textFieldName, text, rest); err != nil {
		return err
	}
	p.currentNode = pop(&p.nodesStack).(*mapNode)
	return nil
}

func (p *xmlParser) buildDocument() error {
	if len(p.fieldHierarchy) == 0 {
		return nil
	}
	return p.validateRequiredFields()
}

func (p *xmlParser) endElement() error {
	field := p.currentField
	p.currentField = nil
	elementName := p.elementName()
	if seen, ok := p.siblings.get(elementName); p.siblings.contains(elementName) && ok && !seen {
		p.siblings.put(elementName, true)
	}
	if field != nil && p.classify(field.ty).isNonNilablePrimitive() && !field.optional &&
		p.currentNode.get(field.runtimeName) == nil {
		return fieldCannotConvertError(field.name, p.typeName(field.ty))
	}
	if len(p.parents) == 0 || !peek(p.parents).contains(elementName) {
		return nil
	}
	if err := p.validateRequiredFields(); err != nil {
		return err
	}
	p.currentNode = pop(&p.nodesStack).(*mapNode)
	p.popExpectedTypeStacks()
	p.updateSiblingAndRootRecord()
	return nil
}

// validateRequiredFields is XmlParser.validateRequiredFields, which unlike the traversal's version does not
// check array sizes.
func (p *xmlParser) validateRequiredFields() error {
	for _, entry := range peek(p.fieldHierarchy).entries() {
		if entry.value != nil && !entry.value.optional {
			return requiredFieldNotPresentError(entry.value.name)
		}
	}
	return p.validateRequiredAttributes()
}

func (p *xmlParser) readElement() error {
	elementName := p.elementName()
	fields := peek(p.fieldHierarchy)
	var field *recordField
	if peek(p.visitedFieldHierarchy).contains(elementName) {
		field, _ = peek(p.visitedFieldHierarchy).get(elementName)
	} else if fields.contains(elementName) {
		elementName, _ = fields.matchedName(elementName)
		field, _ = fields.remove(elementName)
	}
	p.currentField = field
	if field == nil {
		local := elementName.local
		switch {
		case p.peekRestType() != nil:
			if fields.containsLocal(local) {
				return undefinedFieldError(local, p.rootRecordName())
			}
			next, err := p.handleRestField(elementName)
			if err != nil {
				return err
			}
			p.currentNode = next
		case !p.allowDataProjection:
			return undefinedFieldError(local, p.rootRecordName())
		}
		return nil
	}

	peek(p.visitedFieldHierarchy).put(elementName, field)
	key := field.runtimeName
	existing := p.currentNode.get(key)
	ti := p.classify(field.ty)
	if !p.siblings.contains(elementName) {
		p.siblings.put(elementName, false)
	} else if _, isArray := existing.(*arrayNode); ti.isAnydataOrJSON() && !isArray {
		array := p.arrayValueFor(field.ty)
		array.append(existing)
		p.currentNode.put(key, array)
	} else if ti.kind != kindArray {
		return foundArrayForNonArrayTypeError(p.typeName(field.ty), field.name)
	}
	return p.initializeNextValueBasedOnExpectedType(key, field.ty, existing)
}

func (p *xmlParser) initializeNextValueBasedOnExpectedType(key string, fieldType semtypes.SemType, existing node) error {
	ti := p.classify(fieldType)
	switch ti.kind {
	case kindRecord:
		return p.updateNextRecord(key, fieldType, ti)
	case kindArray:
		if existing == nil {
			peek(p.arrayIndexes)[key] = 0
			p.currentNode.put(key, p.arrayValueFor(fieldType))
		} else {
			peek(p.arrayIndexes)[key]++
		}
		return p.updateNextArrayMember(key, fieldType, ti.element)
	case kindMap, kindAnydata, kindJSON:
		return p.initializeAttributesForNextMappingValue(key, fieldType)
	}
	return nil
}

func (p *xmlParser) updateNextArrayMember(key string, fieldType, elementType semtypes.SemType) error {
	ti := p.classify(elementType)
	switch ti.kind {
	case kindRecord:
		return p.updateNextRecord(key, fieldType, ti)
	case kindMap, kindAnydata, kindJSON:
		return p.initializeAttributesForNextMappingValue(key, fieldType)
	}
	return nil
}

func (p *xmlParser) initializeAttributesForNextMappingValue(key string, fieldType semtypes.SemType) error {
	p.parents = append(p.parents, p.siblings)
	p.siblings = newQNameMap[bool]()
	next, err := p.updateNextMappingValue(key, fieldType)
	if err != nil {
		return err
	}
	p.handleAttributesRest(fieldType, next)
	p.currentNode = next
	return nil
}

func (p *xmlParser) updateNextMappingValue(key string, fieldType semtypes.SemType) (*mapNode, error) {
	ty := fieldType
	fieldInfo := p.classify(fieldType)
	if fieldInfo.kind == kindArray {
		ty = fieldInfo.element
	}
	var next *mapNode
	if ti := p.classify(ty); ti.kind == kindMap {
		next = newMapNode(ty)
		rest := ti.mapping.Rest()
		p.restTypes = append(p.restTypes, &rest)
	} else {
		next = newMapNode(p.mapType(ty))
		p.restTypes = append(p.restTypes, &ty)
	}
	p.attributeHierarchy = append(p.attributeHierarchy, newQNameMap[*recordField]())
	p.fieldHierarchy = append(p.fieldHierarchy, newQNameMap[*recordField]())
	p.visitedFieldHierarchy = append(p.visitedFieldHierarchy, newQNameMap[*recordField]())
	p.recordTypeStack = append(p.recordTypeStack, nil)
	if array, ok := p.currentNode.get(key).(*arrayNode); ok {
		if fieldInfo.isAnydataOrJSON() {
			array.append(next)
		} else {
			size := fieldInfo.size
			if fieldInfo.kind != kindArray {
				size = array.ty.size
			}
			index := peek(p.arrayIndexes)[key]
			if index < size || size == -1 {
				array.add(index, next)
			} else if err := p.projectionCheck(); err != nil {
				return nil, err
			}
		}
	} else {
		p.currentNode.put(key, next)
	}
	p.nodesStack = append(p.nodesStack, p.currentNode)
	p.arrayIndexes = append(p.arrayIndexes, map[string]int{})
	return next, nil
}

func (p *xmlParser) updateNextRecord(key string, fieldType semtypes.SemType, recordInfo typeInfo) error {
	p.parents = append(p.parents, p.siblings)
	p.siblings = newQNameMap[bool]()
	p.recordTypeStack = append(p.recordTypeStack, p.rootRecord)
	p.rootRecord = &recordInfo
	next, err := p.updateNextValue(recordInfo, key, fieldType)
	if err != nil {
		return err
	}
	p.currentNode = next
	name := p.reader.name
	if err := p.validateTypeNamespace(name.prefix, name.ns, recordInfo); err != nil {
		return err
	}
	return p.handleAttributes()
}

func (p *xmlParser) updateNextValue(recordInfo typeInfo, key string, fieldType semtypes.SemType) (*mapNode, error) {
	next := p.newRecordNode(recordInfo)
	if err := p.updateExpectedTypeStacks(recordInfo, true); err != nil {
		return nil, err
	}
	if array, ok := p.currentNode.get(key).(*arrayNode); ok {
		arrayInfo := p.classify(fieldType)
		index := peek(p.arrayIndexes)[key]
		if arrayInfo.size == -1 || index < arrayInfo.size {
			array.add(index, next)
		} else if err := p.projectionCheck(); err != nil {
			return nil, err
		}
	} else {
		p.currentNode.put(key, next)
	}
	p.nodesStack = append(p.nodesStack, p.currentNode)
	p.arrayIndexes = append(p.arrayIndexes, map[string]int{})
	return next, nil
}

func (p *xmlParser) updateSiblingAndRootRecord() {
	p.siblings = pop(&p.parents)
	p.rootRecord = pop(&p.recordTypeStack)
}

func (p *xmlParser) handleRestField(elementName qname) (*mapNode, error) {
	p.nodesStack = append(p.nodesStack, p.currentNode)
	p.restFieldsPoints = append(p.restFieldsPoints, &elementName)
	p.parents = append(p.parents, p.siblings)
	return p.parseRestField()
}

func (p *xmlParser) parseRestField() (*mapNode, error) {
	event := p.reader.eventType()
	if restNode, ok, err := p.handleRecordRestType(); err != nil || ok {
		return restNode, err
	}
	var currentFieldName string
	readNext := false
	for len(p.restFieldsPoints) > 0 {
		exit := false
		var err error
		switch event {
		case eventStartElement:
			currentFieldName, err = p.readElementRest()
		case eventEndElement:
			exit, err = p.endElementRest()
		case eventCData:
			err = p.readTextRest(currentFieldName, true)
		case eventCharacters:
			err = p.readTextRest(currentFieldName, false)
			readNext = true
		}
		if err != nil {
			return nil, err
		}
		if exit {
			break
		}
		if !p.reader.hasNext() || len(p.restFieldsPoints) == 0 {
			break
		}
		if readNext {
			readNext = false
			event = p.reader.eventType()
		} else if event, err = p.reader.next(); err != nil {
			return nil, err
		}
	}
	return p.currentNode, nil
}

func (p *xmlParser) updateExpectedTypeStacksOfRestType(restType semtypes.SemType) {
	ti := p.classify(restType)
	switch {
	case ti.kind == kindArray:
		p.updateExpectedTypeStacksOfRestType(ti.element)
	case ti.isAnydataOrJSON():
		p.fieldHierarchy = append(p.fieldHierarchy, newQNameMap[*recordField]())
		p.visitedFieldHierarchy = append(p.visitedFieldHierarchy, newQNameMap[*recordField]())
		p.restTypes = append(p.restTypes, &restType)
		p.arrayIndexes = append(p.arrayIndexes, map[string]int{})
	}
}

func (p *xmlParser) readElementRest() (string, error) {
	elementName := p.elementName()
	name := elementName.local
	restType := *p.peekRestType()
	last, hasLast := p.siblings.lastName()
	if !p.siblings.isEmpty() && hasLast && !p.siblings.getOrDefault(last, true) {
		p.parents = append(p.parents, p.siblings)
		p.siblings = newQNameMap[bool]()
		p.restFieldsPoints = append(p.restFieldsPoints, nil)
		p.updateExpectedTypeStacksOfRestType(restType)
		p.siblings.put(elementName, false)
		next := newMapNode(p.mapTypeFromConstraint(restType))
		p.handleAttributesRest(restType, next)
		target := p.currentNode
		switch existing := p.currentNode.get(last.local).(type) {
		case nil:
			p.currentNode.put(name, next)
		case *arrayNode:
			if m, ok := existing.last().(*mapNode); ok {
				target = m
			}
			target.put(name, next)
		case *mapNode:
			target = existing
			target.put(name, next)
		}
		p.nodesStack = append(p.nodesStack, p.currentNode)
		p.currentNode = target
		return name, nil
	}
	if !p.siblings.contains(elementName) {
		p.siblings.put(elementName, false)
		if restInfo := p.classify(restType); restInfo.kind == kindArray {
			array := p.arrayValueFor(restType)
			p.updateNextArrayMemberForRestType(array, restInfo.element)
			p.currentNode.put(name, array)
		} else {
			next := newMapNode(p.mapTypeFromConstraint(restType))
			p.currentNode.put(name, next)
			p.handleAttributesRest(restType, next)
		}
		return name, nil
	}
	existing := p.currentNode.get(name)
	if array, ok := existing.(*arrayNode); ok {
		p.updateNextArrayMemberForRestType(array, restType)
		p.siblings.put(elementName, false)
		return name, nil
	}
	if !p.classify(restType).isArrayValueAssignable() {
		return "", foundArrayForNonArrayTypeError(p.typeName(restType), name)
	}
	array := p.arrayValueFor(restType)
	array.append(existing)
	p.updateNextArrayMemberForRestType(array, restType)
	p.currentNode.put(name, array)
	p.siblings.put(elementName, false)
	return name, nil
}

func (p *xmlParser) updateNextArrayMemberForRestType(array *arrayNode, restType semtypes.SemType) {
	if p.isSimpleType(restType) {
		return
	}
	next := newMapNode(p.mapTypeFromConstraint(restType))
	array.append(next)
	p.handleAttributesRest(restType, next)
}

func (p *xmlParser) endElementRest() (bool, error) {
	elementName := p.elementName()
	if seen, ok := p.siblings.get(elementName); p.siblings.contains(elementName) && ok && !seen {
		p.siblings.put(elementName, true)
	}
	if isEqualQName(peek(p.restFieldsPoints), &elementName) {
		p.currentNode = pop(&p.nodesStack).(*mapNode)
		p.siblings = pop(&p.parents)
		pop(&p.restFieldsPoints)
		p.siblings.put(elementName, true)
		return true, nil
	}
	if len(p.parents) == 0 || !peek(p.parents).contains(elementName) {
		return false, nil
	}
	p.currentNode = pop(&p.nodesStack).(*mapNode)
	p.siblings = pop(&p.parents)
	if p.siblings.contains(elementName) {
		p.popMappingTypeStacks()
		pop(&p.arrayIndexes)
	}
	pop(&p.restFieldsPoints)
	p.siblings.put(elementName, true)
	return false, nil
}

func (p *xmlParser) readTextRest(name string, isCData bool) error {
	text, commentInMiddle, err := p.readTextValue(isCData)
	if err != nil {
		return err
	}
	if strings.TrimSpace(text) == "" {
		return nil
	}
	restType := *p.peekRestType()
	if commentInMiddle && !p.classify(restType).isStringValueAssignable() {
		return invalidTypeError(p.typeName(restType), "string")
	}
	converted, err := p.convertStringToRestExpType(text, restType)
	if err != nil {
		return err
	}
	switch existing := p.currentNode.get(name).(type) {
	case *arrayNode:
		if p.isSimpleType(restType) {
			existing.append(converted)
		} else {
			existing.add(existing.length()-1, converted)
		}
	case *mapNode:
		if !existing.isEmpty() {
			existing.put(p.textFieldName, converted)
			return nil
		}
		p.currentNode.put(name, converted)
	default:
		p.currentNode.put(name, converted)
	}
	return nil
}

func (p *xmlParser) handleAttributes() error {
	for _, attribute := range p.reader.attributes {
		name := p.qname(attribute.name.ns, p.attributePrefix+attribute.name.local, attribute.name.prefix, stateAttribute)
		field, _ := peek(p.attributeHierarchy).remove(name)
		if field == nil {
			name.state = stateNotDefined
			found, err := p.fieldFromFieldHierarchy(name)
			if err != nil {
				return err
			}
			if found == nil {
				continue
			}
			field = found
		}
		if converted, err := p.parserConvertStringToExpType(attribute.value, field.ty); err == nil {
			p.currentNode.put(field.runtimeName, converted)
		}
	}
	return nil
}

func (p *xmlParser) fieldFromFieldHierarchy(name qname) (*recordField, error) {
	var field *recordField
	if peek(p.visitedFieldHierarchy).contains(name) {
		field, _ = peek(p.visitedFieldHierarchy).get(name)
	} else {
		field, _ = peek(p.fieldHierarchy).remove(name)
		peek(p.visitedFieldHierarchy).put(name, field)
	}
	if field != nil || p.allowDataProjection {
		return field, nil
	}
	return nil, undefinedFieldError(name.local, p.rootRecordName())
}

func (p *xmlParser) handleAttributesRest(restType semtypes.SemType, target *mapNode) {
	for _, attribute := range p.reader.attributes {
		if converted, err := p.convertStringToRestExpType(attribute.value, restType); err == nil {
			target.put(attribute.name.local, converted)
		}
	}
}

func (p *xmlParser) handleRecordRestType() (*mapNode, bool, error) {
	p.currentField = nil
	restType := *p.peekRestType()
	elementName := p.elementName()
	name := elementName.local
	ti := p.classify(restType)
	switch ti.kind {
	case kindRecord:
		p.updateStacksWhenRecordAsRestType(elementName)
		next, err := p.updateNextValue(ti, name, restType)
		if err != nil {
			return nil, false, err
		}
		return p.finishRecordRest(next, name)
	case kindArray:
		indexes := peek(p.arrayIndexes)
		if _, ok := indexes[name]; ok {
			indexes[name]++
		} else {
			indexes[name] = 0
		}
		elementInfo := p.classify(ti.element)
		if elementInfo.kind != kindRecord {
			return nil, false, nil
		}
		p.updateStacksWhenRecordAsRestType(elementName)
		if !p.currentNode.containsKey(name) {
			p.currentNode.put(name, p.arrayValueFor(restType))
		}
		next, err := p.updateNextValue(elementInfo, name, restType)
		if err != nil {
			return nil, false, err
		}
		return p.finishRecordRest(next, name)
	}
	return nil, false, nil
}

func (p *xmlParser) finishRecordRest(next *mapNode, name string) (*mapNode, bool, error) {
	p.currentNode = next
	if err := p.handleAttributes(); err != nil {
		return nil, false, err
	}
	if err := p.parseRecordRest(name); err != nil {
		return nil, false, err
	}
	p.siblings.clear()
	return pop(&p.nodesStack).(*mapNode), true, nil
}

func (p *xmlParser) updateStacksWhenRecordAsRestType(elementName qname) {
	if !p.siblings.contains(elementName) {
		p.siblings.put(elementName, false)
	}
	p.parents = append(p.parents, p.siblings)
	p.siblings = newQNameMap[bool]()
}
