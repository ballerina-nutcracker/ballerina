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

// Record field and annotation handling shared by the parser and the traversal, ported from jBallerina's
// DataUtils and the matching XmlParser helpers.

package native

import (
	"fmt"
	"slices"
	"strings"

	"github.com/ballerina-nutcracker/ballerina/semtypes"
	"github.com/ballerina-nutcracker/ballerina/values"
)

const moduleKeyPrefix = "ballerina/data.xmldata"

// typeAnnotations are the annotations of the target record type and of its own fields, keyed by the field
// name as the runtime spells it. Annotations of any other type are not visible to native code.
type typeAnnotations struct {
	typeLevel values.AnnotationValues
	fields    map[string]values.AnnotationValues
}

func isModuleAnnotationKey(key, name string) bool {
	return strings.HasPrefix(key, moduleKeyPrefix) && strings.HasSuffix(key, name)
}

func findAnnotation(annotations values.AnnotationValues, name string) (values.BalValue, bool) {
	keys := make([]string, 0, len(annotations))
	for key := range annotations {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	for _, key := range keys {
		if isModuleAnnotationKey(key, name) {
			return annotations[key], true
		}
	}
	return nil, false
}

func annotationString(config values.BalValue, field string) (string, bool) {
	m, ok := config.(*values.Map)
	if !ok {
		return "", false
	}
	value, ok := m.Get(field)
	if !ok {
		return "", false
	}
	return fmt.Sprint(value), true
}

// recordField is a jBallerina Field: name is the field name with identifier escapes removed, runtimeName
// the key the runtime stores the field under.
type recordField struct {
	name        string
	runtimeName string
	ty          semtypes.SemType
	optional    bool
}

func unescapeIdentifier(name string) string {
	return strings.ReplaceAll(name, "\\", "")
}

// annotationsOf returns the annotations visible for a record type: only the target type's own.
func (t *typer) annotationsOf(ti typeInfo) typeAnnotations {
	if ti.mapping != nil && semtypes.IsSameType(t.tc, ti.ty, t.root) {
		return t.rootAnnots
	}
	return typeAnnotations{}
}

func (t *typer) recordFields(ti typeInfo) []recordField {
	names := ti.mapping.FieldNames()
	fields := make([]recordField, 0, len(names))
	for _, name := range names {
		fields = append(fields, recordField{
			name:        unescapeIdentifier(name),
			runtimeName: name,
			ty:          ti.mapping.FieldInnerVal(name),
			optional:    ti.mapping.IsOptional(t.tc, name),
		})
	}
	return fields
}

func (t *typer) recordField(ti typeInfo, name string) (recordField, bool) {
	for _, field := range t.recordFields(ti) {
		if field.name == name {
			return field, true
		}
	}
	return recordField{}, false
}

func restTypeOf(ti typeInfo) (semtypes.SemType, bool) {
	rest := ti.mapping.Rest()
	return rest, !semtypes.IsNever(rest)
}

// sortedFieldAnnotations lists field annotations in field-name order, so the iteration is deterministic.
func sortedFieldAnnotations(annots typeAnnotations) []string {
	names := make([]string, 0, len(annots.fields))
	for name := range annots.fields {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}

func fieldQNameFromAnnotations(fieldAnnotations values.AnnotationValues, fieldName string, semantic bool) qname {
	if config, ok := findAnnotation(fieldAnnotations, "Namespace"); ok {
		uri, _ := annotationString(config, "uri")
		prefix, _ := annotationString(config, "prefix")
		return newQName(uri, fieldName, prefix, semantic)
	}
	return newQName(nsNotDefined, fieldName, "", semantic)
}

func modifiedName(fieldAnnotations values.AnnotationValues, fieldName string) string {
	if config, ok := findAnnotation(fieldAnnotations, "Name"); ok {
		if name, ok := annotationString(config, "value"); ok {
			return name
		}
	}
	return fieldName
}

// attributesOf is getAllAttributesInRecordType: the fields annotated with @Attribute, keyed by their XML name.
func (t *typer) attributesOf(ti typeInfo, semantic bool) *qnameMap[*recordField] {
	attributes := newQNameMap[*recordField]()
	annots := t.annotationsOf(ti)
	for _, runtimeName := range sortedFieldAnnotations(annots) {
		fieldAnnotations := annots.fields[runtimeName]
		if _, ok := findAnnotation(fieldAnnotations, "Attribute"); !ok {
			continue
		}
		name := unescapeIdentifier(runtimeName)
		field, ok := t.recordField(ti, name)
		if !ok {
			continue
		}
		q := fieldQNameFromAnnotations(fieldAnnotations, name, semantic)
		q.state = stateAttribute
		q.local = modifiedName(fieldAnnotations, name)
		attributes.put(q, &field)
	}
	return attributes
}

// fieldsOf is getAllFieldsInRecordType: the element fields keyed by their XML name. The parser's version
// looks annotations up by the escaped field name, so an annotated field whose name needs escaping keeps its
// plain name there.
func (t *typer) fieldsOf(ti typeInfo, attributes *qnameMap[*recordField], semantic, escapedAnnotationKeys bool) (*qnameMap[*recordField], error) {
	annots := t.annotationsOf(ti)
	modified := map[string]qname{}
	for _, runtimeName := range sortedFieldAnnotations(annots) {
		fieldAnnotations := annots.fields[runtimeName]
		name := unescapeIdentifier(runtimeName)
		key := name
		if escapedAnnotationKeys {
			key = runtimeName
		}
		q := fieldQNameFromAnnotations(fieldAnnotations, name, semantic)
		q.local = modifiedName(fieldAnnotations, name)
		modified[key] = q
	}

	fieldMap := newQNameMap[*recordField]()
	fieldNames := map[string][]qname{}
	for _, recordField := range t.recordFields(ti) {
		field := &recordField
		q, ok := modified[field.name]
		if !ok {
			q = newQName(nsNotDefined, field.name, "", semantic)
		}
		switch {
		case attributes.contains(q) && q.state == stateNotDefined:
			if attribute, _ := attributes.get(q); attribute == nil || field.name != attribute.name {
				q.state = stateElement
				fieldMap.put(q, field)
				fieldNames[q.local] = []qname{q}
			}
		case fieldMap.hasExact(q):
			return nil, duplicateFieldError(q.local)
		case len(fieldNames[q.local]) > 0:
			for _, existing := range fieldNames[q.local] {
				if isSameAttributeFlag(existing.state, q.state) && isSameNamespace(existing, q) {
					return nil, duplicateFieldError(q.local)
				}
			}
			fieldMap.put(q, field)
			fieldNames[q.local] = append(fieldNames[q.local], q)
		case !attributes.contains(q):
			fieldMap.put(q, field)
			fieldNames[q.local] = []qname{q}
		}
	}
	return fieldMap, nil
}

func (m *qnameMap[V]) hasExact(name qname) bool {
	_, ok := m.members[name.key()]
	return ok
}

// validateRootName is validateAndGetXmlNameFromRecordAnnotation: a @Name on the record type must match the
// element's local name.
func (t *typer) validateRootName(ti typeInfo, element qname) error {
	if config, ok := findAnnotation(t.annotationsOf(ti).typeLevel, "Name"); ok {
		name, _ := annotationString(config, "value")
		if name != element.local {
			return typeNameMismatchError(name, element.local)
		}
	}
	return nil
}

func namespaceOf(config values.BalValue) []string {
	var namespace []string
	if prefix, ok := annotationString(config, "prefix"); ok {
		namespace = append(namespace, prefix)
	}
	uri, _ := annotationString(config, "uri")
	return append(namespace, uri)
}

// validateTypeNamespace checks the element's namespace against a @Namespace on the record type.
func (t *typer) validateTypeNamespace(prefix, uri string, ti typeInfo) error {
	config, ok := findAnnotation(t.annotationsOf(ti).typeLevel, "Namespace")
	if !ok {
		return nil
	}
	return t.checkNamespace(namespaceOf(config), prefix, uri, ti)
}

func (t *typer) checkNamespace(namespace []string, prefix, uri string, ti typeInfo) error {
	switch {
	case len(namespace) == 1 && uri == namespace[0]:
		return nil
	case len(namespace) == 2 && prefix == namespace[0] && uri == namespace[1]:
		return nil
	}
	return namespaceMismatchError(t.typeName(ti.ty))
}

// validateTypeNamespaceForNextRecord checks a nested record element against the @Namespace of the parent's
// field, or failing that of the parent type.
func (t *typer) validateTypeNamespaceForNextRecord(prefix, uri string, ti typeInfo, parent *typeInfo, fieldName string) error {
	if parent == nil {
		return nil
	}
	annots := t.annotationsOf(*parent)
	for _, runtimeName := range sortedFieldAnnotations(annots) {
		if unescapeIdentifier(runtimeName) != fieldName {
			continue
		}
		if config, ok := findAnnotation(annots.fields[runtimeName], "Namespace"); ok {
			return t.checkNamespace(namespaceOf(config), prefix, uri, ti)
		}
	}
	if config, ok := findAnnotation(annots.typeLevel, "Namespace"); ok {
		namespace := namespaceOf(config)
		if len(namespace) == 2 || len(namespace) == 1 && uri == namespace[0] {
			return nil
		}
		return namespaceMismatchError(t.typeName(ti.ty))
	}
	return nil
}
