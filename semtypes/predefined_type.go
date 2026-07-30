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

const (
	neverBits          = basicTypeBitSet(0)
	nilBits            = basicTypeBitSet(1 << int(btNil))
	booleanBits        = basicTypeBitSet(1 << int(btBoolean))
	intBits            = basicTypeBitSet(1 << int(btInt))
	floatBits          = basicTypeBitSet(1 << int(btFloat))
	decimalBits        = basicTypeBitSet(1 << int(btDecimal))
	stringBits         = basicTypeBitSet(1 << int(btString))
	errorBits          = basicTypeBitSet(1 << int(btError))
	listBits           = basicTypeBitSet(1 << int(btList))
	mappingBits        = basicTypeBitSet(1 << int(btMapping))
	tableBits          = basicTypeBitSet(1 << int(btTable))
	cellBits           = basicTypeBitSet(1 << int(btCell))
	undefBits          = basicTypeBitSet(1 << int(btUndef))
	regexpBits         = basicTypeBitSet(1 << int(btRegexp))
	functionBits       = basicTypeBitSet(1 << int(btFunction))
	typedescBits       = basicTypeBitSet(1 << int(btTypeDesc))
	handleBits         = basicTypeBitSet(1 << int(btHandle))
	xmlBits            = basicTypeBitSet(1 << int(btXML))
	objectBits         = basicTypeBitSet(1 << int(btObject))
	streamBits         = basicTypeBitSet(1 << int(btStream))
	futureBits         = basicTypeBitSet(1 << int(btFuture))
	valBits            = basicTypeBitSet(valueTypeMask)
	innerBits          = basicTypeBitSet(valueTypeMask) | undefBits
	anyBits            = basicTypeBitSet(valueTypeMask & ^(1 << int(btError)))
	simpleOrStringBits = basicTypeBitSet((1 << int(btNil)) | (1 << int(btBoolean)) | (1 << int(btInt)) | (1 << int(btFloat)) | (1 << int(btDecimal)) | (1 << int(btString)))
	numberBits         = basicTypeBitSet((1 << int(btInt)) | (1 << int(btFloat)) | (1 << int(btDecimal)))
	simpleBasicBits    = nilBits | booleanBits | intBits | floatBits | decimalBits
)

var (
	Never          = neverBits.semType()
	Nil            = nilBits.semType()
	Boolean        = booleanBits.semType()
	Int            = intBits.semType()
	Float          = floatBits.semType()
	Decimal        = decimalBits.semType()
	String         = stringBits.semType()
	Error          = errorBits.semType()
	List           = listBits.semType()
	Mapping        = mappingBits.semType()
	Table          = tableBits.semType()
	Cell           = cellBits.semType()
	Undef          = undefBits.semType()
	Regexp         = regexpBits.semType()
	Function       = functionBits.semType()
	Typedesc       = typedescBits.semType()
	Handle         = handleBits.semType()
	XML            = xmlBits.semType()
	Object         = objectBits.semType()
	Stream         = streamBits.semType()
	Future         = futureBits.semType()
	Val            = valBits.semType()
	Inner          = innerBits.semType()
	Any            = anyBits.semType()
	SimpleOrString = simpleOrStringBits.semType()
	Number         = numberBits.semType()
	SimpleBasic    = simpleBasicBits.semType()

	predefTypeEnv                     = predefinedTypeEnvGetInstance()
	Byte                              = intWidthUnsigned(8)
	Char                              = makeStringChar()
	ReadonlyXMLElement                = XMLSingleton(xmlPrimitiveElementReadonly)
	XMLElement                        = XMLSingleton((xmlPrimitiveElementReadonly | xmlPrimitiveElementRw))
	ReadonlyXMLComment                = XMLSingleton(xmlPrimitiveCommentReadonly)
	XMLComment                        = XMLSingleton((xmlPrimitiveCommentReadonly | xmlPrimitiveCommentRw))
	XMLText                           = XMLSequence(XMLSingleton(xmlPrimitiveText))
	ReadonlyXMLProcessingInstruction  = XMLSingleton(xmlPrimitiveProcessingInstructionReadonly)
	XMLProcessingInstruction          = XMLSingleton((xmlPrimitiveProcessingInstructionReadonly | xmlPrimitivePiRw))
	bddRecAtomReadonly                = 0
	bddSubtypeRo                      = bddAtom(new(createRecAtom(bddRecAtomReadonly)))
	mappingRo                         = getBasicSubtype(btMapping, bddSubtypeRo)
	cellAtomicVal                     = predefTypeEnv.cellAtomicVal()
	atomCellVal                       = predefTypeEnv.atomCellVal()
	cellAtomicNever                   = predefTypeEnv.cellAtomicNever()
	atomCellNever                     = predefTypeEnv.atomCellNever()
	cellAtomicInner                   = predefTypeEnv.cellAtomicInner()
	atomCellInner                     = predefTypeEnv.atomCellInner()
	cellAtomicUndef                   = predefTypeEnv.cellAtomicUndef()
	atomCellUndef                     = predefTypeEnv.atomCellUndef()
	cellSemtypeInner                  = getBasicSubtype(btCell, bddAtom(atomCellInner))
	MappingAtomicInner                = mappingAtomicTypeFrom(nil, nil, cellSemtypeInner)
	ListAtomicInner                   = listAtomicTypeFrom(fixedLengthArrayEmpty(), cellSemtypeInner)
	cellAtomicInnerMapping            = predefTypeEnv.cellAtomicInnerMapping()
	atomCellInnerMapping              = predefTypeEnv.atomCellInnerMapping()
	cellSemtypeInnerMapping           = getBasicSubtype(btCell, bddAtom(atomCellInnerMapping))
	listAtomicMapping                 = predefTypeEnv.listAtomicMapping()
	atomListMapping                   = predefTypeEnv.atomListMapping()
	listSubtypeMapping                = bddAtom(atomListMapping)
	cellAtomicInnerMappingRo          = predefTypeEnv.cellAtomicInnerMappingRO()
	atomCellInnerMappingRo            = predefTypeEnv.atomCellInnerMappingRO()
	cellSemtypeInnerMappingRo         = getBasicSubtype(btCell, bddAtom(atomCellInnerMappingRo))
	listAtomicMappingRo               = predefTypeEnv.listAtomicMappingRO()
	atomListMappingRo                 = predefTypeEnv.atomListMappingRO()
	listSubtypeMappingRo              = bddAtom(atomListMappingRo)
	cellSemtypeVal                    = getBasicSubtype(btCell, bddAtom(atomCellVal))
	cellSemtypeUndef                  = getBasicSubtype(btCell, bddAtom(atomCellUndef))
	atomCellObjectMemberKind          = predefTypeEnv.atomCellObjectMemberKind()
	cellSemtypeObjectMemberKind       = getBasicSubtype(btCell, bddAtom(atomCellObjectMemberKind))
	atomCellObjectMemberVisibility    = predefTypeEnv.atomCellObjectMemberVisibility()
	cellSemtypeObjectMemberVisibility = getBasicSubtype(btCell, bddAtom(atomCellObjectMemberVisibility))
	atomMappingObjectMember           = predefTypeEnv.atomMappingObjectMember()
	mappingSemtypeObjectMember        = getBasicSubtype(btMapping, bddAtom(atomMappingObjectMember))
	atomCellObjectMember              = predefTypeEnv.atomCellObjectMember()
	cellSemtypeObjectMember           = getBasicSubtype(btCell, bddAtom(atomCellObjectMember))
	cellSemtypeObjectQualifier        = cellSemtypeVal
	atomMappingObject                 = predefTypeEnv.atomMappingObject()
	mappingSubtypeObject              = bddAtom(atomMappingObject)
	bddRecAtomObjectReadonly          = 1
	objectRoRecAtom                   = new(createRecAtom(bddRecAtomObjectReadonly))
	mappingSubtypeObjectRo            = bddAtom(objectRoRecAtom)
	mappingArrayRo                    = getBasicSubtype(btList, listSubtypeMappingRo)
	atomCellMappingArrayRo            = predefTypeEnv.atomCellMappingArrayRO()
	cellSemtypeListSubtypeMappingRo   = getBasicSubtype(btCell, bddAtom(atomCellMappingArrayRo))
	atomListThreeElementRo            = predefTypeEnv.atomListThreeElementRO()
	listSubtypeThreeElementRo         = bddAtom(atomListThreeElementRo)
	ValReadonly                       = createComplexSemType(valueTypeInherentlyImmutable, basicSubtypeFrom(btList, bddSubtypeRo), basicSubtypeFrom(btMapping, bddSubtypeRo), basicSubtypeFrom(btTable, listSubtypeThreeElementRo), basicSubtypeFrom(btXML, xmlSubtypeRo), basicSubtypeFrom(btObject, mappingSubtypeObjectRo))
	ReadonlyInner                     = Union(ValReadonly, Undef)
	cellAtomicInnerRo                 = predefTypeEnv.cellAtomicInnerRO()
	atomCellInnerRo                   = predefTypeEnv.atomCellInnerRO()
	cellSemtypeInnerRo                = getBasicSubtype(btCell, bddAtom(atomCellInnerRo))
	atomCellValRo                     = predefTypeEnv.atomCellValRO()
	cellSemtypeValRo                  = getBasicSubtype(btCell, bddAtom(atomCellValRo))
	atomMappingObjectMemberRo         = predefTypeEnv.atomMappingObjectMemberRO()
	mappingSemtypeObjectMemberRo      = getBasicSubtype(btMapping, bddAtom(atomMappingObjectMemberRo))
	atomCellObjectMemberRo            = predefTypeEnv.atomCellObjectMemberRO()
	cellSemtypeObjectMemberRo         = getBasicSubtype(btCell, bddAtom(atomCellObjectMemberRo))
	listAtomicTwoElement              = predefTypeEnv.listAtomicTwoElement()
	atomListTwoElement                = predefTypeEnv.atomListTwoElement()
	listSubtypeTwoElement             = bddAtom(atomListTwoElement)
	mappingArray                      = getBasicSubtype(btList, listSubtypeMapping)
	atomCellMappingArray              = predefTypeEnv.atomCellMappingArray()
	cellSemtypeListSubtypeMapping     = getBasicSubtype(btCell, bddAtom(atomCellMappingArray))
	atomListThreeElement              = predefTypeEnv.atomListThreeElement()
	listSubtypeThreeElement           = bddAtom(atomListThreeElement)
	mappingAtomicRo                   = predefTypeEnv.mappingAtomicRO()
	mappingAtomicObjectRo             = predefTypeEnv.getMappingAtomicObjectRO()
	listAtomicRo                      = predefTypeEnv.listAtomicRO()
)

func basicTypeUnion(bitset basicTypeBitSet) SemType {
	return bitset.semType()
}

func basicType(code basicTypeCode) SemType {
	return basicTypeBitSet(1 << code.Code()).semType()
}

func getBasicSubtype(code basicTypeCode, data properSubtypeData) SemType {
	if code == btCell {
		return createComplexSemTypeWithAllBitSetSomeBitSetSubtypeDataList(0, Cell.all(), []properSubtypeData{data})
	}
	return createComplexSemTypeWithAllBitSetSomeBitSetSubtypeDataList(0, 1<<code.Code(), []properSubtypeData{data})
}
