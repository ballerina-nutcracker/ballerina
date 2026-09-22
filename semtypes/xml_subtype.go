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
	"github.com/ballerina-nutcracker/ballerina/common"
)

type xmlSubtype struct {
	Primitives int
	Sequence   bdd
}

const (
	xmlPrimitiveNever                         = 1
	xmlPrimitiveText                          = (1 << 1)
	xmlPrimitiveElementReadonly               = (1 << 2)
	xmlPrimitiveProcessingInstructionReadonly = (1 << 3)
	xmlPrimitiveCommentReadonly               = (1 << 4)
	xmlPrimitiveElementRw                     = (1 << 5)
	xmlPrimitivePiRw                          = (1 << 6)
	xmlPrimitiveCommentRw                     = (1 << 7)
	xmlPrimitiveRoSingleton                   = (((xmlPrimitiveText | xmlPrimitiveElementReadonly) | xmlPrimitiveProcessingInstructionReadonly) | xmlPrimitiveCommentReadonly)
	xmlPrimitiveRoMask                        = (xmlPrimitiveNever | xmlPrimitiveRoSingleton)
	xmlPrimitiveRwMask                        = ((xmlPrimitiveElementRw | xmlPrimitivePiRw) | xmlPrimitiveCommentRw)
	xmlPrimitiveSingleton                     = (xmlPrimitiveRoSingleton | xmlPrimitiveRwMask)
	xmlPrimitiveAllMask                       = (xmlPrimitiveRoMask | xmlPrimitiveRwMask)
)

var _ properSubtypeData = &xmlSubtype{}

func newXmlSubtypeFromIntBdd(primitives int, sequence bdd) *xmlSubtype {
	return &xmlSubtype{
		Primitives: primitives,
		Sequence:   sequence,
	}
}

func xmlSubtypeFrom(primitives int, sequence bdd) *xmlSubtype {
	return newXmlSubtypeFromIntBdd(primitives, sequence)
}

func XMLSingleton(primitives int) SemType {
	return createXmlSemtype(createXmlSubtype(primitives, bddNothing()))
}

func XMLSequence(constituentType SemType) SemType {
	common.Assert(func() bool { return IsSubtypeSimple(constituentType, XML) })
	if IsNever(constituentType) {
		return XMLSequence(XMLSingleton(xmlPrimitiveNever))
	}
	if constituentType.some() == 0 {
		return constituentType
	}
	xmlSt := getComplexSubtypeData(constituentType, btXML)
	if _, ok := xmlSt.(allOrNothingSubtype); ok {
		// xmlSt stays as is
	} else {
		xmlSt = makeXmlSequence(xmlSt.(*xmlSubtype))
	}
	return createXmlSemtype(xmlSt)
}

func XMLItemType(t SemType) SemType {
	if !IsSubtypeSimple(t, XML) {
		return Never
	}
	if t.some() == 0 {
		return t
	}
	xmlSt := getComplexSubtypeData(t, btXML)
	if allOrNothing, ok := xmlSt.(allOrNothingSubtype); ok {
		if allOrNothing.IsAllSubtype() {
			return XML
		}
		return Never
	}
	bits := xmlSt.(*xmlSubtype).Primitives &^ xmlPrimitiveNever
	var itemTy = Never
	if bits&(xmlPrimitiveElementReadonly|xmlPrimitiveElementRw) != 0 {
		itemTy = Union(itemTy, XMLElement)
	}
	if bits&(xmlPrimitiveCommentReadonly|xmlPrimitiveCommentRw) != 0 {
		itemTy = Union(itemTy, XMLComment)
	}
	if bits&(xmlPrimitiveProcessingInstructionReadonly|xmlPrimitivePiRw) != 0 {
		itemTy = Union(itemTy, XMLProcessingInstruction)
	}
	if bits&xmlPrimitiveText != 0 {
		itemTy = Union(itemTy, XMLText)
	}
	return itemTy
}

func makeXmlSequence(d *xmlSubtype) subtypeData {
	primitives := (xmlPrimitiveNever | d.Primitives)
	atom := (d.Primitives & xmlPrimitiveSingleton)
	sequence := bddUnion(bddAtom(new(createXMLRecAtom(atom))), d.Sequence)
	return createXmlSubtype(primitives, sequence)
}

func createXmlSemtype(xmlSubtype subtypeData) SemType {
	if allOrNothingSubtype, ok := xmlSubtype.(allOrNothingSubtype); ok {
		if allOrNothingSubtype.IsAllSubtype() {
			return XML
		} else {
			return Never
		}
	} else {
		return getBasicSubtype(btXML, xmlSubtype.(properSubtypeData))
	}
}

func createXmlSubtype(primitives int, sequence bdd) subtypeData {
	p := (primitives & xmlPrimitiveAllMask)
	if allOrNothing, ok := sequence.(*bddAllOrNothing); ok && allOrNothing.IsAll() && (p == xmlPrimitiveAllMask) {
		return createAll()
	}
	return createXmlSubtypeOrEmpty(p, sequence)
}

func createXmlSubtypeOrEmpty(primitives int, sequence bdd) subtypeData {
	if allOrNothing, ok := sequence.(*bddAllOrNothing); ok && allOrNothing.IsNothing() && (primitives == 0) {
		return createNothing()
	}
	return xmlSubtypeFrom(primitives, sequence)
}
