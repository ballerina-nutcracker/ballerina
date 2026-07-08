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
	"ballerina-lang-go/common"
)

type xmlSubtype struct {
	Primitives int
	Sequence   Bdd
}

const (
	XML_PRIMITIVE_NEVER        = 1
	XML_PRIMITIVE_TEXT         = (1 << 1)
	XML_PRIMITIVE_ELEMENT_RO   = (1 << 2)
	XML_PRIMITIVE_PI_RO        = (1 << 3)
	XML_PRIMITIVE_COMMENT_RO   = (1 << 4)
	XML_PRIMITIVE_ELEMENT_RW   = (1 << 5)
	XML_PRIMITIVE_PI_RW        = (1 << 6)
	XML_PRIMITIVE_COMMENT_RW   = (1 << 7)
	XML_PRIMITIVE_RO_SINGLETON = (((XML_PRIMITIVE_TEXT | XML_PRIMITIVE_ELEMENT_RO) | XML_PRIMITIVE_PI_RO) | XML_PRIMITIVE_COMMENT_RO)
	XML_PRIMITIVE_RO_MASK      = (XML_PRIMITIVE_NEVER | XML_PRIMITIVE_RO_SINGLETON)
	XML_PRIMITIVE_RW_MASK      = ((XML_PRIMITIVE_ELEMENT_RW | XML_PRIMITIVE_PI_RW) | XML_PRIMITIVE_COMMENT_RW)
	XML_PRIMITIVE_SINGLETON    = (XML_PRIMITIVE_RO_SINGLETON | XML_PRIMITIVE_RW_MASK)
	XML_PRIMITIVE_ALL_MASK     = (XML_PRIMITIVE_RO_MASK | XML_PRIMITIVE_RW_MASK)
)

var _ ProperSubtypeData = &xmlSubtype{}

func newXmlSubtypeFromIntBdd(primitives int, sequence Bdd) *xmlSubtype {
	return &xmlSubtype{
		Primitives: primitives,
		Sequence:   sequence,
	}
}

func xmlSubtypeFrom(primitives int, sequence Bdd) *xmlSubtype {
	return newXmlSubtypeFromIntBdd(primitives, sequence)
}

func XMLSingleton(primitives int) SemType {
	return createXmlSemtype(createXmlSubtype(primitives, bddNothing()))
}

func XMLSequence(constituentType SemType) SemType {
	common.Assert(func() bool { return IsSubtypeSimple(constituentType, XML) })
	if IsNever(constituentType) {
		return XMLSequence(XMLSingleton(XML_PRIMITIVE_NEVER))
	}
	if constituentType.some() == 0 {
		return constituentType
	}
	xmlSt := getComplexSubtypeData(constituentType, BTXML)
	if _, ok := xmlSt.(allOrNothingSubtype); ok {
		// xmlSt stays as is
	} else {
		xmlSt = makeXmlSequence(xmlSt.(*xmlSubtype))
	}
	return createXmlSemtype(xmlSt)
}

func XMLItemType(t SemType) SemType {
	if !IsSubtypeSimple(t, XML) {
		return NEVER
	}
	if t.some() == 0 {
		return t
	}
	xmlSt := getComplexSubtypeData(t, BTXML)
	if allOrNothing, ok := xmlSt.(allOrNothingSubtype); ok {
		if allOrNothing.IsAllSubtype() {
			return XML
		}
		return NEVER
	}
	bits := xmlSt.(*xmlSubtype).Primitives &^ XML_PRIMITIVE_NEVER
	var itemTy = NEVER
	if bits&(XML_PRIMITIVE_ELEMENT_RO|XML_PRIMITIVE_ELEMENT_RW) != 0 {
		itemTy = Union(itemTy, XML_ELEMENT)
	}
	if bits&(XML_PRIMITIVE_COMMENT_RO|XML_PRIMITIVE_COMMENT_RW) != 0 {
		itemTy = Union(itemTy, XML_COMMENT)
	}
	if bits&(XML_PRIMITIVE_PI_RO|XML_PRIMITIVE_PI_RW) != 0 {
		itemTy = Union(itemTy, XML_PI)
	}
	if bits&XML_PRIMITIVE_TEXT != 0 {
		itemTy = Union(itemTy, XML_TEXT)
	}
	return itemTy
}

func makeXmlSequence(d *xmlSubtype) SubtypeData {
	primitives := (XML_PRIMITIVE_NEVER | d.Primitives)
	atom := (d.Primitives & XML_PRIMITIVE_SINGLETON)
	sequence := bddUnion(bddAtom(new(createXMLRecAtom(atom))), d.Sequence)
	return createXmlSubtype(primitives, sequence)
}

func createXmlSemtype(xmlSubtype SubtypeData) SemType {
	if allOrNothingSubtype, ok := xmlSubtype.(allOrNothingSubtype); ok {
		if allOrNothingSubtype.IsAllSubtype() {
			return XML
		} else {
			return NEVER
		}
	} else {
		return getBasicSubtype(BTXML, xmlSubtype.(ProperSubtypeData))
	}
}

func createXmlSubtype(primitives int, sequence Bdd) SubtypeData {
	p := (primitives & XML_PRIMITIVE_ALL_MASK)
	if allOrNothing, ok := sequence.(*bddAllOrNothing); ok && allOrNothing.IsAll() && (p == XML_PRIMITIVE_ALL_MASK) {
		return createAll()
	}
	return createXmlSubtypeOrEmpty(p, sequence)
}

func createXmlSubtypeOrEmpty(primitives int, sequence Bdd) SubtypeData {
	if allOrNothing, ok := sequence.(*bddAllOrNothing); ok && allOrNothing.IsNothing() && (primitives == 0) {
		return createNothing()
	}
	return xmlSubtypeFrom(primitives, sequence)
}
