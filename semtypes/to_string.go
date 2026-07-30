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

package semtypes

import (
	"fmt"
	"strings"
)

type toStringState struct {
	cx      Context
	visited map[atomKey]bool
}

func newToStringState(cx Context) *toStringState {
	return &toStringState{cx: cx, visited: make(map[atomKey]bool)}
}

func ToString(cx Context, ty SemType) string {
	s := newToStringState(cx)
	return s.semTypeToString(ty)
}

func builtinUnion(cx Context, ty SemType) (string, bool) {
	if IsSameType(cx, ty, Any) {
		return "any", true
	}
	if IsSameType(cx, ty, CreateAnydata(cx)) {
		return "anydata", true
	}
	if IsSameType(cx, ty, CreateJSON(cx)) {
		return "json", true
	}
	return "", false
}

func (s *toStringState) semTypeToString(ty SemType) string {
	if res, ok := builtinUnion(s.cx, ty); ok {
		return res
	}
	if ty.some() == 0 {
		return basicTypeToString(ty.all())
	}
	if name, ok := xmlPredefinedName(s.cx, ty); ok {
		return name
	}
	return s.complexSemtypeToString(ty)
}

func xmlPredefinedName(cx Context, ty SemType) (string, bool) {
	if IsSameType(cx, ty, XMLElement) {
		return "xml:Element", true
	}
	if IsSameType(cx, ty, XMLComment) {
		return "xml:Comment", true
	}
	if IsSameType(cx, ty, XMLText) {
		return "xml:Text", true
	}
	if IsSameType(cx, ty, XMLProcessingInstruction) {
		return "xml:ProcessingInstruction", true
	}
	return "", false
}

func basicTypeToString(ty basicTypeBitSet) string {
	bits := ty & basicTypeMask
	if bits == 0 {
		return "never"
	}
	return basicTypeBitSetToString(bits)
}

func basicTypeBitSetToString(bits basicTypeBitSet) string {
	var parts []string
	for i := 0; i < int(valueTypeCount); i++ {
		if bits&(1<<i) != 0 {
			code := basicTypeCodeFrom(i)
			name := strings.TrimPrefix(code.String(), "BT_")
			parts = append(parts, strings.ToLower(name))
		}
	}
	return strings.Join(parts, "|")
}

func (s *toStringState) complexSemtypeToString(ty SemType) string {
	var parts []string
	allStr := basicTypeBitSetToString(ty.all())
	if allStr != "" {
		parts = append(parts, allStr)
	}
	for _, sub := range unpack(ty) {
		parts = append(parts, s.subtypeToString(sub))
	}
	return strings.Join(parts, "|")
}

func (s *toStringState) subtypeToString(sub basicSubtype) string {
	switch st := sub.subtypeData.(type) {
	case intSubtype:
		return intSubtypeToString(st)
	case booleanSubtype:
		return booleanSubtypeToString(st)
	case floatSubtype:
		return floatSubtypeToString(st)
	case decimalSubtype:
		return decimalSubtypeToString(st)
	case stringSubtype:
		return stringSubtypeToString(st)
	case bdd:
		switch sub.basicTypeCode {
		case btList:
			return s.bddListToString(st)
		case btMapping:
			return s.bddMappingToString(st)
		case btError:
			return s.bddErrorToString(st)
		case btFunction:
			return s.bddFunctionToString(st)
		case btObject:
			return s.bddObjectToString(st)
		case btTypeDesc:
			return s.bddTypedescToString(st)
		default:
			name := strings.TrimPrefix(sub.basicTypeCode.String(), "BT_")
			return strings.ToLower(name)
		}
	case *xmlSubtype:
		return s.xmlSubtypeToString(st)
	default:
		panic(fmt.Sprintf("unimplemented: ToString for %s", sub.basicTypeCode.String()))
	}
}

func bddFormulaToString(cx Context, bdd bdd, atomToString func(atom) string) string {
	var formulas []string
	bddEvery(cx, bdd, conjunctionNil, conjunctionNil, func(cx Context, pos conjunctionHandle, neg conjunctionHandle) bool {
		var posParts []string
		for c := pos; c != conjunctionNil; c = cx.conjunctionNext(c) {
			posParts = append(posParts, atomToString(cx.conjunctionAtom(c)))
		}
		for i, j := 0, len(posParts)-1; i < j; i, j = i+1, j-1 {
			posParts[i], posParts[j] = posParts[j], posParts[i]
		}
		var negParts []string
		for c := neg; c != conjunctionNil; c = cx.conjunctionNext(c) {
			negParts = append(negParts, "¬"+atomToString(cx.conjunctionAtom(c)))
		}
		for i, j := 0, len(negParts)-1; i < j; i, j = i+1, j-1 {
			negParts[i], negParts[j] = negParts[j], negParts[i]
		}
		parts := append(posParts, negParts...)
		formulas = append(formulas, strings.Join(parts, "&"))
		return true
	})
	return strings.Join(formulas, "|")
}

func (s *toStringState) bddListToString(bdd bdd) string {
	return bddFormulaToString(s.cx, bdd, s.listAtomToString)
}

func (s *toStringState) listAtomToString(atom atom) string {
	if recAtom, ok := atom.(*recAtom); ok && recAtom.index() == bddRecAtomReadonly {
		return "readonly"
	}
	key := atom.canonicalKey()
	if s.visited[key] {
		return "..."
	}
	s.visited[key] = true
	defer delete(s.visited, key)
	return s.listAtomicTypeToString(atom)
}

func (s *toStringState) listAtomicTypeToString(atom atom) string {
	atomic := s.cx.ListAtomType(atom)
	var parts []string
	for i := 0; i < atomic.members.FixedLength; i++ {
		member := listMemberAt(atomic.members, atomic.rest, i)
		parts = append(parts, s.semTypeToString(cellInnerVal(member)))
	}
	restStr := s.semTypeToString(cellInnerVal(atomic.rest))
	parts = append(parts, restStr+"...")
	return "[" + strings.Join(parts, ", ") + "]"
}

func (s *toStringState) bddErrorToString(bdd bdd) string {
	return "error<" + bddFormulaToString(s.cx, bdd, s.errorAtomToString) + ">"
}

func (s *toStringState) errorAtomToString(atom atom) string {
	if recAtom, ok := atom.(*recAtom); ok && recAtom.index() < 0 {
		return "error"
	}
	return s.mappingAtomToString(atom)
}

func (s *toStringState) bddFunctionToString(bdd bdd) string {
	return bddFormulaToString(s.cx, bdd, s.functionAtomToString)
}

func (s *toStringState) functionAtomToString(atom atom) string {
	key := atom.canonicalKey()
	if s.visited[key] {
		return "..."
	}
	s.visited[key] = true
	defer delete(s.visited, key)
	return s.functionAtomicTypeToString(atom)
}

func (s *toStringState) functionAtomicTypeToString(atom atom) string {
	atomic := s.cx.FunctionAtomType(atom)
	paramsStr := s.functionParamsToString(atomic.ParamType)
	retStr := s.semTypeToString(atomic.RetType)
	return "function(" + paramsStr + ") returns " + retStr
}

func (s *toStringState) functionParamsToString(paramType SemType) string {
	// ParamType is a list SemType representing the parameter tuple.
	// Try to extract individual parameter types from the list atom.
	if paramType.some() == 0 {
		return s.semTypeToString(paramType)
	}
	for _, sub := range unpack(paramType) {
		if sub.basicTypeCode != btList {
			continue
		}
		bdd, ok := sub.subtypeData.(bdd)
		if !ok {
			continue
		}
		node, ok := bdd.(bddNode)
		if !ok {
			continue
		}
		listAtomic := s.cx.ListAtomType(node.atom())
		var parts []string
		for i := 0; i < listAtomic.members.FixedLength; i++ {
			member := listMemberAt(listAtomic.members, listAtomic.rest, i)
			parts = append(parts, s.semTypeToString(cellInnerVal(member)))
		}
		restInner := cellInnerVal(listAtomic.rest)
		if !IsNever(restInner) {
			parts = append(parts, s.semTypeToString(restInner)+"...")
		}
		return strings.Join(parts, ", ")
	}
	return s.semTypeToString(paramType)
}

func (s *toStringState) bddTypedescToString(bdd bdd) string {
	mappingTy := createBasicSemType(btMapping, bdd)
	constraint := MappingMemberTypeInnerVal(s.cx, mappingTy, String)
	if IsSameType(s.cx, constraint, Val) {
		return "typedesc"
	}
	return "typedesc<" + s.semTypeToString(constraint) + ">"
}

func (s *toStringState) bddMappingToString(bdd bdd) string {
	return bddFormulaToString(s.cx, bdd, s.mappingAtomToString)
}

func (s *toStringState) mappingAtomToString(atom atom) string {
	if recAtom, ok := atom.(*recAtom); ok && recAtom.index() == bddRecAtomReadonly {
		return "readonly"
	}
	key := atom.canonicalKey()
	if s.visited[key] {
		return "..."
	}
	s.visited[key] = true
	defer delete(s.visited, key)
	return s.mappingAtomicTypeToString(atom)
}

func (s *toStringState) mappingAtomicTypeToString(atom atom) string {
	atomic := s.cx.MappingAtomType(atom)
	var parts []string
	for i, name := range atomic.names {
		parts = append(parts, name+": "+s.semTypeToString(cellInnerVal(atomic.types[i])))
	}
	restStr := s.semTypeToString(cellInnerVal(atomic.rest))
	parts = append(parts, restStr+"...")
	return "{| " + strings.Join(parts, ", ") + " |}"
}

func (s *toStringState) bddObjectToString(bdd bdd) string {
	return bddFormulaToString(s.cx, bdd, s.objectAtomToString)
}

func (s *toStringState) objectAtomToString(atom atom) string {
	if recAtom, ok := atom.(*recAtom); ok {
		if recAtom.index() < 0 {
			return "object"
		}
		if recAtom.index() == bddRecAtomObjectReadonly {
			return "readonly"
		}
	}
	key := atom.canonicalKey()
	if s.visited[key] {
		return "..."
	}
	s.visited[key] = true
	defer delete(s.visited, key)
	return s.objectAtomicTypeToString(atom)
}

func (s *toStringState) objectAtomicTypeToString(atom atom) string {
	atomic := s.cx.MappingAtomType(atom)
	var prefix []string
	var members []string
	for i, name := range atomic.names {
		if name == "$qualifiers" {
			qualTy := cellInnerVal(atomic.types[i])
			qualAtomic := ToMappingAtomicType(s.cx, qualTy)
			if qualAtomic != nil {
				isolatedTy := qualAtomic.FieldInnerVal("isolated")
				if IsSameType(s.cx, isolatedTy, BooleanConst(true)) {
					prefix = append(prefix, "isolated")
				}
				networkTy := qualAtomic.FieldInnerVal("network")
				if IsSameType(s.cx, networkTy, StringConst("client")) {
					prefix = append(prefix, "client")
				} else if IsSameType(s.cx, networkTy, StringConst("service")) {
					prefix = append(prefix, "service")
				}
			}
			continue
		}
		memberTy := cellInnerVal(atomic.types[i])
		memberAtomic := ToMappingAtomicType(s.cx, memberTy)
		if memberAtomic == nil {
			members = append(members, name+": "+s.semTypeToString(memberTy))
			continue
		}
		kindTy := memberAtomic.FieldInnerVal("kind")
		valueTy := memberAtomic.FieldInnerVal("value")
		visibilityTy := memberAtomic.FieldInnerVal("visibility")
		visibilityPrefix := visibilityToString(s.cx, visibilityTy)
		if IsSameType(s.cx, kindTy, StringConst("field")) {
			members = append(members, visibilityPrefix+s.semTypeToString(valueTy)+" "+name)
		} else {
			members = append(members, visibilityPrefix+s.objectMethodToString(name, kindTy, valueTy))
		}
	}
	prefix = append(prefix, "object")
	result := strings.Join(prefix, " ")
	if len(members) > 0 {
		result += " { " + strings.Join(members, "; ") + " }"
	}
	return result
}

func visibilityToString(cx Context, visibilityTy SemType) string {
	if IsSameType(cx, visibilityTy, StringConst("public")) {
		return "public "
	}
	if IsSameType(cx, visibilityTy, StringConst("private")) {
		return "private "
	}
	return ""
}

func (s *toStringState) objectMethodToString(name string, kindTy SemType, fnTy SemType) string {
	var methodPrefix string
	if IsSameType(s.cx, kindTy, StringConst("remote-method")) {
		methodPrefix = "remote function "
	} else if IsSameType(s.cx, kindTy, StringConst("resource-method")) {
		methodPrefix = "resource function "
	} else {
		methodPrefix = "function "
	}
	if fnTy.some() == 0 {
		return methodPrefix + name + "()"
	}
	for _, sub := range unpack(fnTy) {
		if sub.basicTypeCode == btFunction {
			bdd, ok := sub.subtypeData.(bdd)
			if !ok {
				continue
			}
			node, ok := bdd.(bddNode)
			if !ok {
				continue
			}
			atomic := s.cx.FunctionAtomType(node.atom())
			paramsStr := s.functionParamsToString(atomic.ParamType)
			retStr := s.semTypeToString(atomic.RetType)
			return methodPrefix + name + "(" + paramsStr + ") returns " + retStr
		}
	}
	return methodPrefix + name + "()"
}

func intSubtypeToString(st intSubtype) string {
	// Check special width types
	type namedWidth struct {
		min, max int64
		name     string
	}
	widths := []namedWidth{
		{0, 255, "int:Unsigned8"},
		{0, 65535, "int:Unsigned16"},
		{0, 4294967295, "int:Unsigned32"},
		{-128, 127, "int:Signed8"},
		{-32768, 32767, "int:Signed16"},
		{-2147483648, 2147483647, "int:Signed32"},
	}
	if len(st.Ranges) == 1 {
		r := st.Ranges[0]
		for _, w := range widths {
			if r.Min == w.min && r.Max == w.max {
				return w.name
			}
		}
	}
	// Individual values or ranges
	var parts []string
	for _, r := range st.Ranges {
		for v := r.Min; v <= r.Max; v++ {
			parts = append(parts, fmt.Sprintf("%d", v))
		}
	}
	return strings.Join(parts, "|")
}

func booleanSubtypeToString(st booleanSubtype) string {
	if st.Value {
		return "true"
	}
	return "false"
}

func floatSubtypeToString(st floatSubtype) string {
	var parts []string
	for _, v := range st.values {
		parts = append(parts, fmt.Sprintf("%g", v.value))
	}
	return strings.Join(parts, "|")
}

func decimalSubtypeToString(st decimalSubtype) string {
	var parts []string
	for _, v := range st.values {
		parts = append(parts, v.value.String())
	}
	return strings.Join(parts, "|")
}

func (s *toStringState) xmlSubtypeToString(st *xmlSubtype) string {
	bits := st.Primitives & ^xmlPrimitiveNever
	bddEvery(s.cx, st.Sequence, conjunctionNil, conjunctionNil, func(cx Context, pos conjunctionHandle, neg conjunctionHandle) bool {
		for c := pos; c != conjunctionNil; c = cx.conjunctionNext(c) {
			if rec, ok := cx.conjunctionAtom(c).(*recAtom); ok {
				bits |= rec.index()
			}
		}
		return true
	})
	return "xml<" + xmlConstituentName(bits) + ">"
}

func xmlConstituentName(bits int) string {
	if bits == 0 {
		return "never"
	}
	var parts []string
	if bits&xmlPrimitiveText != 0 {
		parts = append(parts, "xml:Text")
	}
	if bits&(xmlPrimitiveElementReadonly|xmlPrimitiveElementRw) != 0 {
		parts = append(parts, "xml:Element")
	}
	if bits&(xmlPrimitiveCommentReadonly|xmlPrimitiveCommentRw) != 0 {
		parts = append(parts, "xml:Comment")
	}
	if bits&(xmlPrimitiveProcessingInstructionReadonly|xmlPrimitivePiRw) != 0 {
		parts = append(parts, "xml:ProcessingInstruction")
	}
	return strings.Join(parts, "|")
}

func stringSubtypeToString(st stringSubtype) string {
	// Check for Char type: charData.allowed=false, no char values, nonCharData.allowed=true, no nonChar values
	if !st.charData.allowed && len(st.charData.values) == 0 &&
		st.nonCharData.allowed && len(st.nonCharData.values) == 0 {
		return "string:Char"
	}
	var parts []string
	for _, v := range st.charData.values {
		parts = append(parts, fmt.Sprintf("%q", v.Value()))
	}
	for _, v := range st.nonCharData.values {
		parts = append(parts, fmt.Sprintf("%q", v.Value()))
	}
	return strings.Join(parts, "|")
}
