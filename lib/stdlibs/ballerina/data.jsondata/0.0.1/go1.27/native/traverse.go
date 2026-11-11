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
	"errors"
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
	// fieldNames is the newest entry of jBallerina's field-name deque, which holds the newest nameCount names.
	fieldNames *fieldName
	nameCount  int
	// namesPushed counts the names pushed and not popped; a replayed attempt adds the count it recorded.
	namesPushed    int
	depth          int
	failedAttempts map[attemptKey]failedAttempt
}

// fieldName is an entry of the field-name deque, linked to the entry pushed before it: one name, or a replayed
// run of the newest count names of the deque whose newest entry is run. Entries are never changed.
type fieldName struct {
	name  string
	run   *fieldName
	count int
	prev  *fieldName
}

// attemptKey identifies converting a mapping or list to a union member. The outcome depends only on the value
// and the member, except for the nesting bound, and so do the names it pushes onto the deque.
type attemptKey struct {
	value  any
	member any
	depth  int
}

// failedAttempt is what a failed attempt left on the deque: names is its newest entry, of which the newest
// pushed are the names it pushed and did not pop.
type failedAttempt struct {
	names  *fieldName
	pushed int
}

// maxTraverseDepth bounds the nesting of mappings and lists in a converted value: the traverser recurses per
// level, and Go aborts the process when a goroutine overflows its stack. encoding/json uses the same bound.
const maxTraverseDepth = 10000

func traverse(t *typer, opts projection, json values.BalValue, target semtypes.SemType) (values.BalValue, error) {
	tr := &traverser{typer: t, projection: opts}
	return tr.traverseJSON(json, target)
}

func (tr *traverser) traverseJSON(json values.BalValue, target semtypes.SemType) (values.BalValue, error) {
	return tr.traverseClassified(json, target, tr.classify(target))
}

func (tr *traverser) traverseClassified(json values.BalValue, target semtypes.SemType, ti typeInfo) (values.BalValue, error) {
	if json == nil && tr.isNilable(target) {
		return nil, nil
	}
	switch ti.kind {
	case kindRecord:
		if _, ok := json.(*values.Map); !ok {
			return nil, tr.incompatibleTypeErrorFor(target, json)
		}
		fields, err := tr.recordFields(ti)
		if err != nil {
			return nil, err
		}
		tr.pushFields(fields, ti)
		return tr.traverseMapOrArray(json, tr.newMappingBuilder(ti), ti)
	case kindList:
		if _, ok := json.(*values.List); !ok {
			return nil, tr.incompatibleTypeErrorFor(target, json)
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

func (tr *traverser) pushFieldName(name string) {
	tr.fieldNames = &fieldName{name: name, prev: tr.fieldNames}
	tr.nameCount++
	tr.namesPushed++
}

func (tr *traverser) popFieldName() string {
	name := tr.fieldNames.name
	tr.fieldNames = tr.fieldNames.prev
	tr.nameCount--
	tr.namesPushed--
	return name
}

// traverseUnion replays a member that already failed for the value rather than trying it again: each ancestor
// union tries its next member on the same subtree, so a union of two recursive records is otherwise 2^depth.
func (tr *traverser) traverseUnion(json values.BalValue, ti typeInfo) (values.BalValue, error) {
	for _, member := range tr.orderedMembers(ti.members, json) {
		namesSize := tr.nameCount
		memberInfo := tr.classify(member)
		key, memoizable := tr.attemptKey(json, memberInfo)
		if failed, ok := tr.failedAttempts[key]; memoizable && ok {
			tr.replay(failed)
		} else {
			hierarchySize, restSize, pushed := len(tr.fieldHierarchy), len(tr.restTypes), tr.namesPushed
			result, err := tr.traverseClassified(json, member, memberInfo)
			if err == nil || errors.Is(err, errTooDeep) {
				return result, err
			}
			tr.fieldHierarchy = tr.fieldHierarchy[:min(hierarchySize, len(tr.fieldHierarchy))]
			tr.restTypes = tr.restTypes[:min(restSize, len(tr.restTypes))]
			if memoizable {
				if tr.failedAttempts == nil {
					tr.failedAttempts = make(map[attemptKey]failedAttempt)
				}
				tr.failedAttempts[key] = failedAttempt{names: tr.fieldNames, pushed: tr.namesPushed - pushed}
			}
		}
		// jBallerina restores the deque with pollLast, which drops the oldest names.
		tr.nameCount = min(tr.nameCount, namesSize)
	}
	return nil, tr.incompatibleTypeErrorFor(ti.ty, json)
}

func (tr *traverser) attemptKey(json values.BalValue, member typeInfo) (attemptKey, bool) {
	switch json.(type) {
	case *values.Map, *values.List:
	default:
		return attemptKey{}, false
	}
	key := attemptKey{value: json, depth: tr.depth}
	switch {
	case member.mapping != nil:
		key.member = member.mapping
	case member.list != nil:
		key.member = member.list
	default:
		return attemptKey{}, false
	}
	return key, true
}

// replay leaves the deque as the failed attempt left it: its names pushed after the existing ones. The caller
// then drops as many of the oldest names.
func (tr *traverser) replay(failed failedAttempt) {
	if failed.pushed > 0 {
		tr.fieldNames = &fieldName{run: failed.names, count: failed.pushed, prev: tr.fieldNames}
		tr.namesPushed += failed.pushed
	}
}

// newestNames returns the newest n names of the deque whose newest entry is entry, oldest first.
func newestNames(entry *fieldName, n int) []string {
	type resume struct {
		entry *fieldName
		left  int
	}
	names := make([]string, n)
	var outer []resume
	left := n
	for i := n - 1; i >= 0; {
		switch {
		case left <= 0 || entry == nil:
			if len(outer) == 0 {
				return names[i+1:]
			}
			entry, left = outer[len(outer)-1].entry, outer[len(outer)-1].left
			outer = outer[:len(outer)-1]
		case entry.run != nil:
			outer = append(outer, resume{entry: entry.prev, left: left - entry.count})
			entry, left = entry.run, min(left, entry.count)
		default:
			names[i] = entry.name
			i--
			left--
			entry = entry.prev
		}
	}
	return names
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
	if tr.depth == maxTraverseDepth {
		return nil, errTooDeep
	}
	tr.depth++
	defer func() { tr.depth-- }()
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
	if tr.nameCount == 0 {
		return nil, tr.incompatibleTypeErrorFor(ti.ty, json)
	}
	fieldPath := tr.fieldPath()
	return nil, lazyJSONError(func() string {
		return invalidTypeForFieldError(fieldPath()).message
	})
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

		tr.pushFieldName(field.name)
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
	if tr.nameCount == 0 {
		return nil, lazyJSONError(func() string {
			return incompatibleTypeError(tr.typeName(target), javaValueOf(json)).message
		})
	}
	fieldPath := tr.fieldPath()
	return nil, lazyJSONError(func() string {
		return incompatibleValueForFieldError(javaValueOf(json), tr.typeName(target), fieldPath()).message
	})
}

func (tr *traverser) incompatibleTypeErrorFor(target semtypes.SemType, json values.BalValue) *jsonError {
	return lazyJSONError(func() string {
		return incompatibleTypeError(tr.typeName(target), formatMessageArg(json)).message
	})
}

// fieldPath returns a function that joins the names on the deque now, which the deque's entries keep.
func (tr *traverser) fieldPath() func() string {
	names, count := tr.fieldNames, tr.nameCount
	return func() string {
		return strings.Join(newestNames(names, count), ".")
	}
}
