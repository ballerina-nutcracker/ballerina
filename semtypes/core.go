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
	"math/bits"

	"ballerina-lang-go/common"
	"ballerina-lang-go/decimal"
)

const (
	MAX_VALUE = int64(^uint(0) >> 1) // Platform max int (typically 2^63-1 on 64-bit systems)
	MIN_VALUE = -MAX_VALUE - 1       // Platform min int
)

func bitCount(b basicTypeBitSet) int {
	return bits.OnesCount(uint(b & basicTypeMask))
}

func IsZero(t SemType) bool {
	return (t.allBits & semTypeMarker) == 0
}

func sameSemType(t1, t2 SemType) bool {
	return !IsZero(t1) && !IsZero(t2) && sameComplexSemType(t1, t2)
}

func cellAtomType(atom atom) *cellAtomicType {
	ta := atom.(*typeAtom)
	atomicType := ta.AtomicType
	if cellAtomicType, ok := atomicType.(*cellAtomicType); ok {
		return cellAtomicType
	}
	panic("expected cell atomic type")
}

func Diff(t1, t2 SemType) SemType {
	all1, some1 := t1.all(), t1.some()
	all2, some2 := t2.all(), t2.some()
	if some1 == 0 && some2 == 0 {
		return basicTypeUnion(all1 & ^all2)
	}
	if IsNever(t1) {
		return t1
	}
	if some2 == 0 && all2 == ValueTypeMask {
		return NEVER
	}
	all := all1 & ^(all2 | some2)
	someBitSet := (all1 | some1) & ^all2
	someBitSet = someBitSet & ^all
	some := someBitSet
	if some == 0 {
		return basicTypeUnion(all)
	}
	var subtypes []basicSubtype
	it := newSubtypePairs(t1, t2, some)
	for it.hasNext() {
		pair := it.next()
		code := pair.BasicTypeCode
		data1 := pair.SubtypeData1
		data2 := pair.SubtypeData2
		var data SubtypeData
		if data1 == nil {
			data = ops[code.Code()].complement(data2)
		} else if data2 == nil {
			data = data1
		} else {
			data = ops[code.Code()].Diff(data1, data2)
		}
		if allOrNothing, ok := data.(allOrNothingSubtype); !ok {
			subtypes = append(subtypes, basicSubtypeFrom(code, data.(ProperSubtypeData)))
		} else if allOrNothing.IsAllSubtype() {
			c := code.Code()
			all = all | (1 << c)
		}
	}
	if len(subtypes) == 0 {
		return basicTypeUnion(all)
	}
	return createComplexSemType(all, subtypes...)
}

func getComplexSubtypeData(t SemType, code BasicTypeCode) SubtypeData {
	c := basicTypeBitSet(1 << code.Code())
	if (t.all() & c) != 0 {
		return createAll()
	}
	if (t.some() & c) == 0 {
		return createNothing()
	}
	loBits := t.some() & (c - 1)
	var index int
	if loBits == 0 {
		index = 0
	} else {
		index = bits.OnesCount(uint(loBits))
	}
	return t.subtypeDataList()[index]
}

func Union(t1, t2 SemType) SemType {
	all1, some1 := t1.all(), t1.some()
	all2, some2 := t2.all(), t2.some()
	if some1 == 0 && some2 == 0 {
		return basicTypeUnion(all1 | all2)
	}
	all := all1 | all2
	some := (some1 | some2) & ^all
	if some == 0 {
		return basicTypeUnion(all)
	}
	var subtypes []basicSubtype
	it := newSubtypePairs(t1, t2, some)
	for it.hasNext() {
		pair := it.next()
		code := pair.BasicTypeCode
		data1 := pair.SubtypeData1
		data2 := pair.SubtypeData2
		var data SubtypeData
		if data1 == nil {
			data = data2
		} else if data2 == nil {
			data = data1
		} else {
			data = ops[code.Code()].Union(data1, data2)
		}
		if allOrNothing, ok := data.(allOrNothingSubtype); ok && allOrNothing.IsAllSubtype() {
			c := code.Code()
			all = all | (1 << c)
		} else {
			subtypes = append(subtypes, basicSubtypeFrom(code, data.(ProperSubtypeData)))
		}
	}
	if len(subtypes) == 0 {
		return basicTypeUnion(all)
	}
	return createComplexSemType(all, subtypes...)
}

func Intersect(t1, t2 SemType) SemType {
	all1, some1 := t1.all(), t1.some()
	all2, some2 := t2.all(), t2.some()
	if some1 == 0 && some2 == 0 {
		return basicTypeUnion(all1 & all2)
	}
	if some1 == 0 {
		if all1 == 0 {
			return t1
		}
		if all1 == ValueTypeMask {
			return t2
		}
	}
	if some2 == 0 {
		if all2 == 0 {
			return t2
		}
		if all2 == ValueTypeMask {
			return t1
		}
	}
	all := all1 & all2
	some := (some1 | all1) & (some2 | all2)
	some = some & ^all
	if some == 0 {
		return basicTypeUnion(all)
	}
	var subtypes []basicSubtype
	it := newSubtypePairs(t1, t2, some)
	for it.hasNext() {
		pair := it.next()
		code := pair.BasicTypeCode
		data1 := pair.SubtypeData1
		data2 := pair.SubtypeData2
		var data SubtypeData
		if data1 == nil {
			data = data2
		} else if data2 == nil {
			data = data1
		} else {
			data = ops[code.Code()].Intersect(data1, data2)
		}
		if allOrNothing, ok := data.(allOrNothingSubtype); !ok || allOrNothing.IsAllSubtype() {
			subtypes = append(subtypes, basicSubtypeFrom(code, data.(ProperSubtypeData)))
		}
	}
	if len(subtypes) == 0 {
		return basicTypeUnion(all)
	}
	return createComplexSemType(all, subtypes...)
}

func intersectMemberSemTypes(env Env, t1, t2 SemType) SemType {
	c1 := getCellAtomicType(t1)
	c2 := getCellAtomicType(t2)
	common.Assert(func() bool { return c1 != nil && c2 != nil })
	atomicType := intersectCellAtomicType(c1, c2)
	var mut CellMutability
	if sameSemType(atomicType.Ty, UNDEF) {
		mut = CellMutability_CELL_MUT_NONE
	} else {
		mut = atomicType.Mut
	}
	return cellContainingWithEnvSemTypeCellMutability(env, atomicType.Ty, mut)
}

func complement(t SemType) SemType {
	return Diff(VAL, t)
}

func IsNever(t SemType) bool {
	return t.all() == 0 && t.some() == 0
}

func IsEmpty(cx Context, t SemType) bool {
	if t.some() == 0 {
		return t.all() == 0
	}
	if t.all() != 0 {
		return false
	}
	some := t.some()
	for _, data := range t.subtypeDataList() {
		code := basicTypeCodeFrom(bits.TrailingZeros(uint(some)))
		if !ops[code.Code()].IsEmpty(cx, data) {
			return false
		}
		some &^= 1 << code.Code()
	}
	return true
}

func IsSubtype(cx Context, t1, t2 SemType) bool {
	return IsEmpty(cx, Diff(t1, t2))
}

func IsSubtypeSimple(t1 SemType, t2 SemType) bool {
	return (widenToBasicTypeBits(t1) & ^t2.all()) == 0
}

func IsSameType(cx Context, t1, t2 SemType) bool {
	return IsSubtype(cx, t1, t2) && IsSubtype(cx, t2, t1)
}

// NBasicTypes returns the number of basic types to which the given type belongs to
func NBasicTypes(t SemType) int {
	return bitCount(widenToBasicTypeBits(t))
}

func widenToBasicTypeBits(t SemType) basicTypeBitSet {
	return t.all() | t.some()
}

func WidenToBasicTypes(t SemType) SemType {
	return widenToBasicTypeBits(t).semType()
}

func wideUnsigned(t SemType) SemType {
	if t.some() == 0 {
		return t
	}
	if !IsSubtypeSimple(t, INT) {
		return t
	}
	data := intSubtypeWidenUnsigned(subtypeData(t, BTInt))
	if _, ok := data.(allOrNothingSubtype); ok {
		return INT
	}
	return getBasicSubtype(BTInt, data.(ProperSubtypeData))
}

func getBooleanSubtype(t SemType) SubtypeData {
	return subtypeData(t, BTBoolean)
}

func getIntSubtype(t SemType) SubtypeData {
	return subtypeData(t, BTInt)
}

func getFloatSubtype(t SemType) SubtypeData {
	return subtypeData(t, BTFloat)
}

func getDecimalSubtype(t SemType) SubtypeData {
	return subtypeData(t, BTDecimal)
}

func getStringSubtype(t SemType) SubtypeData {
	return subtypeData(t, BTString)
}

func ListMemberTypeInnerVal(cx Context, t, k SemType) SemType {
	if t.some() == 0 {
		if (t.all() & LIST.all()) != 0 {
			return VAL
		}
		return NEVER
	}
	keyData := getIntSubtype(k)
	if isNothingSubtype(keyData) {
		return NEVER
	}
	return bddListMemberTypeInnerVal(cx, getComplexSubtypeData(t, BTList).(Bdd), keyData, VAL)
}

var LIST_MEMBER_TYPES_ALL = listMemberTypesFrom([]intRange{rangeFrom(0, int64(MAX_VALUE))}, []SemType{VAL})

var LIST_MEMBER_TYPES_NONE = listMemberTypesFrom([]intRange{}, []SemType{})

func ListAllMemberTypesInner(cx Context, t SemType) ListMemberTypes {
	if t.some() == 0 {
		if (t.all() & LIST.all()) != 0 {
			return LIST_MEMBER_TYPES_ALL
		}
		return LIST_MEMBER_TYPES_NONE
	}

	ct := t
	ranges := []intRange{}
	types := []SemType{}

	allRanges := bddListAllRanges(cx, getComplexSubtypeData(ct, BTList).(Bdd), []intRange{})
	for _, r := range allRanges {
		m := ListMemberTypeInnerVal(cx, t, IntConst(r.Min))
		if !IsNever(m) {
			ranges = append(ranges, r)
			types = append(types, m)
		}
	}
	return listMemberTypesFrom(ranges, types)
}

func bddListAllRanges(cx Context, b Bdd, accum []intRange) []intRange {
	if allOrNothing, ok := b.(*bddAllOrNothing); ok {
		if allOrNothing.IsAll() {
			return accum
		} else {
			return []intRange{}
		}
	} else {
		bn := b.(bddNode)
		listMemberTypes := listAtomicTypeAllMemberTypesInnerVal(cx.ListAtomType(bn.atom()))
		return distinctRanges(bddListAllRanges(cx, bn.left(),
			distinctRanges(listMemberTypes.Ranges, accum)),
			distinctRanges(bddListAllRanges(cx, bn.middle(), accum),
				bddListAllRanges(cx, bn.right(), accum)))
	}
}

func distinctRanges(range1, range2 []intRange) []intRange {
	combined := combineRanges(range1, range2)
	rangeResult := make([]intRange, len(combined))
	for i := range combined {
		rangeResult[i] = combined[i].Range
	}
	return rangeResult
}

func combineRanges(ranges1, ranges2 []intRange) []combinedRange {
	combined := []combinedRange{}
	i1 := 0
	i2 := 0
	len1 := len(ranges1)
	len2 := len(ranges2)
	cur := int64(MIN_VALUE)

	// This iterates over the boundaries between ranges
	for {
		for i1 < len1 && cur > int64(ranges1[i1].Max) {
			i1 += 1
		}
		for i2 < len2 && cur > int64(ranges2[i2].Max) {
			i2 += 1
		}

		var next *int64 = nil
		if i1 < len1 {
			next = nextBoundary(cur, ranges1[i1], next)
		}
		if i2 < len2 {
			next = nextBoundary(cur, ranges2[i2], next)
		}

		var max int64
		if next == nil {
			max = int64(MAX_VALUE)
		} else {
			max = *next - 1
		}

		var in1 int64 = -1
		if i1 < len1 {
			r := ranges1[i1]
			if cur >= int64(r.Min) && max <= int64(r.Max) {
				in1 = int64(i1)
			}
		}

		var in2 int64 = -1
		if i2 < len2 {
			r := ranges2[i2]
			if cur >= int64(r.Min) && max <= int64(r.Max) {
				in2 = int64(i2)
			}
		}

		if in1 != -1 || in2 != -1 {
			combined = append(combined, combinedRangeFrom(rangeFrom(cur, max), in1, in2))
		}

		if next == nil {
			break
		}
		cur = *next
	}
	return combined
}

func nextBoundary(cur int64, r intRange, next *int64) *int64 {
	if (int64(r.Min) > cur) && (next == nil || int64(r.Min) < *next) {
		result := int64(r.Min)
		return &result
	}
	if r.Max != int64(MAX_VALUE) {
		i := int64(r.Max) + 1
		if i > cur && (next == nil || i < *next) {
			return &i
		}
	}
	return next
}

func listAtomicTypeAllMemberTypesInnerVal(atomicType *ListAtomicType) ListMemberTypes {
	ranges := []intRange{}
	types := []SemType{}

	cellInitial := atomicType.Members.initial
	initialLength := int64(len(cellInitial))

	initial := make([]SemType, 0, initialLength)
	for i := range cellInitial {
		initial = append(initial, cellInnerVal(cellInitial[i]))
	}

	fixedLength := int64(atomicType.Members.FixedLength)
	if initialLength != 0 {
		types = append(types, initial...)
		for i := range initialLength {
			ranges = append(ranges, rangeFrom(i, i))
		}
		if initialLength < fixedLength {
			ranges[initialLength-1] = rangeFrom(initialLength-1, fixedLength-1)
		}
	}

	rest := cellInnerVal(atomicType.rest)
	if !IsNever(rest) {
		types = append(types, rest)
		ranges = append(ranges, rangeFrom(fixedLength, MAX_VALUE))
	}

	return listMemberTypesFrom(ranges, types)
}

func ToObjectAtomicType(cx Context, t SemType) *MappingAtomicType {
	mappingTy := convertObjectToMappingTy(cx, t)
	if IsZero(mappingTy) {
		return nil
	}
	return ToMappingAtomicType(cx, mappingTy)
}

func ToMappingAtomicType(cx Context, t SemType) *MappingAtomicType {
	mappingAtomicInner := MAPPING_ATOMIC_INNER
	if t.some() == 0 {
		if t.all() == MAPPING.all() {
			return &mappingAtomicInner
		}
		return nil
	}
	if !IsSubtypeSimple(t, MAPPING) {
		return nil
	}
	return bddMappingAtomicType(cx, getComplexSubtypeData(t, BTMapping).(Bdd))
}

func bddMappingAtomicType(cx Context, bdd Bdd) *MappingAtomicType {
	var result *MappingAtomicType
	pathCount := 0
	valid := bddEveryPositive(cx, bdd, conjunctionNil, conjunctionNil,
		func(cx Context, pos conjunctionHandle, neg conjunctionHandle) bool {
			pathCount++
			if pathCount > 1 || neg != conjunctionNil || pos == conjunctionNil || cx.conjunctionNext(pos) != conjunctionNil {
				return false
			}
			result = cx.MappingAtomType(cx.conjunctionAtom(pos))
			return result != nil
		})
	if !valid || pathCount != 1 {
		return nil
	}
	return result
}

func MappingMemberTypeInnerVal(cx Context, t, k SemType) SemType {
	return Diff(MappingMemberTypeInner(cx, t, k), UNDEF)
}

func MappingMemberTypeInner(cx Context, t, k SemType) SemType {
	if t.some() == 0 {
		if (t.all() & MAPPING.all()) != 0 {
			return VAL
		}
		return UNDEF
	}
	keyData := getStringSubtype(k)
	if isNothingSubtype(keyData) {
		return UNDEF
	}
	return bddMappingMemberTypeInnerCore(cx, getComplexSubtypeData(t, BTMapping).(Bdd), keyData,
		INNER)
}

func ToListAtomicType(cx Context, t SemType) *ListAtomicType {
	listAtomicInner := LIST_ATOMIC_INNER
	if t.some() == 0 {
		if t.all() == LIST.all() {
			return &listAtomicInner
		}
		return nil
	}
	env := cx.Env()
	if !IsSubtypeSimple(t, LIST) {
		return nil
	}
	return bddListAtomicType(env,
		getComplexSubtypeData(t, BTList).(Bdd),
		listAtomicInner)
}

func bddListAtomicType(env Env, bdd Bdd, top ListAtomicType) *ListAtomicType {
	if allOrNothing, ok := bdd.(*bddAllOrNothing); ok {
		if allOrNothing.IsAll() {
			return &top
		}
		return nil
	}
	bn := bdd.(bddNode)
	if bddNodeSimple, ok := bn.(*bddNodeSimple); ok {
		result := env.listAtomType(bddNodeSimple.atom())
		return result
	}
	return nil
}

func cellInnerVal(t SemType) SemType {
	return Diff(cellInner(t), UNDEF)
}

func cellInner(t SemType) SemType {
	cat := getCellAtomicType(t)
	common.Assert(func() bool { return cat != nil })
	return cat.Ty
}

func cellContainingInnerVal(env Env, t SemType) SemType {
	cat := getCellAtomicType(t)
	common.Assert(func() bool { return cat != nil })
	return cellContainingWithEnvSemTypeCellMutability(env, Diff(cat.Ty, UNDEF), cat.Mut)
}

func getCellAtomicType(t SemType) *cellAtomicType {
	if t.some() == 0 {
		if t.all() == CELL.all() {
			return CELL_ATOMIC_VAL
		}
		return nil
	}
	if !IsSubtypeSimple(t, CELL) {
		return nil
	}
	return bddCellAtomicType(getComplexSubtypeData(t, BTCell).(Bdd), CELL_ATOMIC_VAL)
}

func bddCellAtomicType(bdd Bdd, top *cellAtomicType) *cellAtomicType {
	if allOrNothing, ok := bdd.(*bddAllOrNothing); ok {
		if allOrNothing.IsAll() {
			return top
		}
		return nil
	}
	bn := bdd.(bddNode)
	leftBdd := bn.left()
	middleBdd := bn.middle()
	rightBdd := bn.right()

	if leftAll, ok := leftBdd.(*bddAllOrNothing); ok && leftAll.IsAll() {
		if middleNothing, ok := middleBdd.(*bddAllOrNothing); ok && middleNothing.IsNothing() {
			if rightNothing, ok := rightBdd.(*bddAllOrNothing); ok && rightNothing.IsNothing() {
				return cellAtomType(bn.atom())
			}
		}
	}
	return nil
}

func SingleShape(t SemType) common.Optional[Value] {
	if sameSemType(t, NIL) {
		return common.OptionalOf(valueFrom(nil))
	} else if t.some() == 0 {
		return common.OptionalEmpty[Value]()
	} else if IsSubtypeSimple(t, INT) {
		sd := getComplexSubtypeData(t, BTInt)
		value := intSubtypeSingleValue(sd)
		if value.IsEmpty() {
			return common.OptionalEmpty[Value]()
		} else {
			return common.OptionalOf(valueFrom(value.Get()))
		}
	} else if IsSubtypeSimple(t, FLOAT) {
		sd := getComplexSubtypeData(t, BTFloat)
		value := floatSubtypeSingleValue(sd)
		if value.IsEmpty() {
			return common.OptionalEmpty[Value]()
		} else {
			return common.OptionalOf(valueFrom(value.Get()))
		}
	} else if IsSubtypeSimple(t, STRING) {
		sd := getComplexSubtypeData(t, BTString)
		value := stringSubtypeSingleValue(sd)
		if value.IsEmpty() {
			return common.OptionalEmpty[Value]()
		} else {
			return common.OptionalOf(valueFrom(value.Get()))
		}
	} else if IsSubtypeSimple(t, BOOLEAN) {
		sd := getComplexSubtypeData(t, BTBoolean)
		value := booleanSubtypeSingleValue(sd)
		if value.IsEmpty() {
			return common.OptionalEmpty[Value]()
		} else {
			return common.OptionalOf(valueFrom(value.Get()))
		}
	} else if IsSubtypeSimple(t, DECIMAL) {
		sd := getComplexSubtypeData(t, BTDecimal)
		value := decimalSubtypeSingleValue(sd)
		if value.IsEmpty() {
			return common.OptionalEmpty[Value]()
		} else {
			d := value.Get()
			return common.OptionalOf(valueFrom(&d))
		}
	}
	return common.OptionalEmpty[Value]()
}

func singleton(v any) SemType {
	if v == nil {
		return NIL
	}

	if lng, ok := v.(int64); ok {
		return IntConst(lng)
	} else if d, ok := v.(float64); ok {
		return FloatConst(d)
	} else if s, ok := v.(string); ok {
		return StringConst(s)
	} else if b, ok := v.(bool); ok {
		return BooleanConst(b)
	} else if r, ok := v.(*decimal.Decimal); ok {
		return DecimalConst(*r)
	} else {
		panic("Unsupported type: " + fmt.Sprintf("%T", v))
	}
}

func containsConst(t SemType, v any) bool {
	if v == nil {
		return containsNil(t)
	} else if lng, ok := v.(int64); ok {
		return containsConstInt(t, lng)
	} else if d, ok := v.(float64); ok {
		return containsConstFloat(t, d)
	} else if s, ok := v.(string); ok {
		return containsConstString(t, s)
	} else if b, ok := v.(bool); ok {
		return containsConstBoolean(t, b)
	} else {
		return containsConstDecimal(t, v.(decimal.Decimal))
	}
}

func containsNil(t SemType) bool {
	if t.some() == 0 {
		return (t.all() & (1 << BTNil.Code())) != 0
	}
	complexSubtypeData := getComplexSubtypeData(t, BTNil).(allOrNothingSubtype)
	return complexSubtypeData.IsAllSubtype()
}

func containsConstString(t SemType, s string) bool {
	if t.some() == 0 {
		return (t.all() & (1 << BTString.Code())) != 0
	}
	return stringSubtypeContains(getComplexSubtypeData(t, BTString), s)
}

func containsConstInt(t SemType, n int64) bool {
	if t.some() == 0 {
		return (t.all() & (1 << BTInt.Code())) != 0
	}
	return intSubtypeContains(getComplexSubtypeData(t, BTInt), n)
}

func containsConstFloat(t SemType, n float64) bool {
	if t.some() == 0 {
		return (t.all() & (1 << BTFloat.Code())) != 0
	}
	return floatSubtypeContains(getComplexSubtypeData(t, BTFloat), newEnumerableFloatFromFloat64(n))
}

func containsConstDecimal(t SemType, n decimal.Decimal) bool {
	if t.some() == 0 {
		return (t.all() & (1 << BTDecimal.Code())) != 0
	}
	return decimalSubtypeContains(getComplexSubtypeData(t, BTDecimal), enumerableDecimalFrom(n))
}

func containsConstBoolean(t SemType, b bool) bool {
	if t.some() == 0 {
		return (t.all() & (1 << BTBoolean.Code())) != 0
	}
	return booleanSubtypeContains(getComplexSubtypeData(t, BTBoolean), b)
}

func SingleNumericType(semType SemType) common.Optional[SemType] {
	numType := Intersect(semType, NUMBER)
	if IsNever(numType) {
		return common.OptionalEmpty[SemType]()
	}
	if IsSubtypeSimple(numType, INT) {
		return common.OptionalOf(INT)
	}
	if IsSubtypeSimple(numType, FLOAT) {
		return common.OptionalOf(FLOAT)
	}
	if IsSubtypeSimple(numType, DECIMAL) {
		return common.OptionalOf(DECIMAL)
	}
	return common.OptionalEmpty[SemType]()
}

func subtypeData(s SemType, code BasicTypeCode) SubtypeData {
	if s.some() == 0 {
		if (s.all() & (1 << code.Code())) != 0 {
			return createAll()
		}
		return createNothing()
	}
	return getComplexSubtypeData(s, code)
}

func TypeCheckContext(env Env) Context {
	return ContextFrom(env)
}

func CreateJSON(context Context) SemType {
	return context.Env().preallocatedTypeVals.json
}

func CreateAnydata(context Context) SemType {
	return context.Env().preallocatedTypeVals.anydata
}

func CreateCloneable(context Context) SemType {
	memo := context.cloneableMemo()
	env := context.Env()

	if !IsZero(memo) {
		return memo
	}
	listDef := &ListDefinition{}
	mapDef := &MappingDefinition{}
	tableTy := tableContainingDefault(env, mapDef.GetSemType(env))
	ad := Union(VAL_READONLY, Union(XML, Union(listDef.GetSemType(env), Union(tableTy,
		mapDef.GetSemType(env)))))
	listDef.DefineListTypeWrappedWithEnvSemType(env, ad)
	mapDef.DefineMappingTypeWrapped(env, []Field{}, ad)
	context.setCloneableMemo(ad)
	return ad
}

func CreateOrdered(context Context) SemType {
	memo := context.orderedMemo()
	env := context.Env()

	if !IsZero(memo) {
		return memo
	}
	listDef := &ListDefinition{}
	ordered := Union(NIL, Union(BOOLEAN, Union(INT, Union(FLOAT, Union(DECIMAL, Union(STRING, listDef.GetSemType(env)))))))
	listDef.DefineListTypeWrappedWithEnvSemType(env, ordered)
	context.setOrderedMemo(ordered)
	return ordered
}

func createIsolatedObject(context Context) SemType {
	memo := context.isolatedObjectMemo()
	if !IsZero(memo) {
		return memo
	}

	quals := ObjectQualifiersFrom(true, false, NetworkQualifierNone)
	od := NewObjectDefinition()
	isolatedObj := od.Define(context.Env(), quals, []Member{})
	context.setIsolatedObjectMemo(isolatedObj)
	return isolatedObj
}

func CreateServiceObject(context Context) SemType {
	memo := context.serviceObjectMemo()
	if !IsZero(memo) {
		return memo
	}

	quals := ObjectQualifiersFrom(false, false, NetworkQualifierService)
	od := NewObjectDefinition()
	serviceObj := od.Define(context.Env(), quals, []Member{})
	context.setServiceObjectMemo(serviceObj)
	return serviceObj
}

func CreateClientObject(context Context) SemType {
	memo := context.clientObjectMemo()
	if !IsZero(memo) {
		return memo
	}

	quals := ObjectQualifiersFrom(false, false, NetworkQualifierClient)
	od := NewObjectDefinition()
	clientObj := od.Define(context.Env(), quals, []Member{})
	context.setClientObjectMemo(clientObj)
	return clientObj
}

func CreateIterable(context Context) SemType {
	memo := context.iterableMemo()
	if !IsZero(memo) {
		return memo
	}
	env := context.Env()

	// Build the broadest next() return type: record {| (any|error) value; |}|error?
	valueField := FieldFrom("value", VAL, false, false)
	md := NewMappingDefinition()
	recordTy := md.DefineMappingTypeWrapped(env, []Field{valueField}, NEVER)
	nextReturnTy := Union(recordTy, Union(ERROR, NIL))

	// next() function type: () -> nextReturnTy
	ld := NewListDefinition()
	emptyParams := ld.DefineListTypeWrapped(env, []SemType{}, 0, NEVER, CellMutability_CELL_MUT_NONE)
	fd := NewFunctionDefinition()
	nextFnTy := fd.Define(env, emptyParams, nextReturnTy, FunctionQualifiersFrom(env, false, false))

	// Iterator object type: object { public function next() ... }
	iteratorMembers := []Member{
		{Name: "next", ValueTy: nextFnTy, Kind: MemberKindMethod, Visibility: VisibilityPublic, Immutable: true},
	}
	iterOd := NewObjectDefinition()
	iteratorTy := iterOd.Define(env, ObjectQualifiersDEFAULT, iteratorMembers)

	// iterator() function type: () -> iteratorTy
	ld2 := NewListDefinition()
	emptyParams2 := ld2.DefineListTypeWrapped(env, []SemType{}, 0, NEVER, CellMutability_CELL_MUT_NONE)
	fd2 := NewFunctionDefinition()
	iteratorFnTy := fd2.Define(env, emptyParams2, iteratorTy, FunctionQualifiersFrom(env, false, false))

	// Iterable object type: object { public function iterator() ... }
	iterableMembers := []Member{
		{Name: "iterator", ValueTy: iteratorFnTy, Kind: MemberKindMethod, Visibility: VisibilityPublic, Immutable: true},
	}
	iterableOd := NewObjectDefinition()
	iterableTy := iterableOd.Define(env, ObjectQualifiersDEFAULT, iterableMembers)

	context.setIterableMemo(iterableTy)
	return iterableTy
}

func createBasicSemType(typeCode BasicTypeCode, subtypeData SubtypeData) SemType {
	if _, ok := subtypeData.(allOrNothingSubtype); ok {
		if isAllSubtype(subtypeData) {
			return basicTypeBitSet(1 << typeCode.Code()).semType()
		} else {
			return NEVER
		}
	} else {
		return createComplexSemType(0,
			basicSubtypeFrom(typeCode, subtypeData.(ProperSubtypeData)))
	}
}

func mappingAtomicTypesInUnion(cx Context, t SemType) common.Optional[[]MappingAtomicType] {
	matList := []MappingAtomicType{}
	mappingAtomicInner := MAPPING_ATOMIC_INNER
	if t.some() == 0 {
		if t.all() == MAPPING.all() {
			matList = append(matList, mappingAtomicInner)
			return common.OptionalOf(matList)
		}
		return common.OptionalEmpty[[]MappingAtomicType]()
	}
	env := cx.Env()
	if !IsSubtypeSimple(t, MAPPING) {
		return common.OptionalEmpty[[]MappingAtomicType]()
	}
	if collectBddMappingAtomicTypesInUnion(env,
		getComplexSubtypeData(t, BTMapping).(Bdd),
		mappingAtomicInner, &matList) {
		return common.OptionalOf(matList)
	}
	return common.OptionalEmpty[[]MappingAtomicType]()
}

func collectBddMappingAtomicTypesInUnion(env Env, bdd Bdd, top MappingAtomicType, matList *[]MappingAtomicType) bool {
	if allOrNothing, ok := bdd.(*bddAllOrNothing); ok {
		if allOrNothing.IsAll() {
			*matList = append(*matList, top)
			return true
		}
		return false
	}
	bn := bdd.(bddNode)
	if bddNodeSimple, ok := bn.(*bddNodeSimple); ok {
		*matList = append(*matList, *env.mappingAtomType(bddNodeSimple.atom()))
		return true
	}

	bddNodeImpl := bdd.(*bddNodeImpl)
	leftBdd := bddNodeImpl.left()
	rightBdd := bddNodeImpl.right()

	if leftNode, ok := leftBdd.(*bddAllOrNothing); ok && leftNode.IsAll() {
		if rightNode, ok := rightBdd.(*bddAllOrNothing); ok && rightNode.IsNothing() {
			*matList = append(*matList, *env.mappingAtomType(bddNodeImpl.atom()))
			return collectBddMappingAtomicTypesInUnion(env, bddNodeImpl.middle(), top, matList)
		}
	}

	return false
}

func Comparable(cx Context, t1, t2 SemType) bool {
	semType := Diff(Union(t1, t2), NIL)
	if IsSubtypeSimple(semType, SIMPLE_OR_STRING) {
		nOrderings := bitCount(widenToBasicTypeBits(semType))
		return nOrderings <= 1
	}
	if IsSubtypeSimple(semType, LIST) {
		return comparableNillableList(cx, t1, t2)
	}
	return false
}

// t1, t2 must be subtype of LIST|?
// According to the spec
// [T...] is ordered, if T is ordered;
// [] is ordered;
// [T, rest] is ordered if T is ordered and [rest] is ordered.
func comparableNillableList(cx Context, t1, t2 SemType) bool {
	b1, ok1 := listSubtypeBdd(t1)
	b2, ok2 := listSubtypeBdd(t2)
	var memo *comparableMemo
	if ok1 && ok2 {
		if memoized := cx.comparableMemo(b1, b2); memoized != nil {
			return memoized.comparable
		}
		// We assume recursive types aren't comparable. We need this because spec defines list ordering
		// inductively.
		memo = &comparableMemo{comparable: false}
		cx.setComparableMemo(b1, b2, memo)
	}
	listMemberTypes1 := ListAllMemberTypesInner(cx, t1)
	listMemberTypes2 := ListAllMemberTypesInner(cx, t2)
	ranges1 := listMemberTypes1.Ranges
	ranges2 := listMemberTypes2.Ranges
	memberTypes1 := listMemberTypes1.SemTypes
	memberTypes2 := listMemberTypes2.SemTypes
	for _, combinedRange := range combineRanges(ranges1, ranges2) {
		i1 := combinedRange.I1
		i2 := combinedRange.I2
		if i1 != -1 && i2 != -1 && !Comparable(cx, memberTypes1[i1], memberTypes2[i2]) {
			return false
		}
	}
	if memo != nil {
		memo.comparable = true
	}
	return true
}

func listSubtypeBdd(t SemType) (Bdd, bool) {
	if t.some() == 0 {
		return nil, false
	}
	bdd, ok := getComplexSubtypeData(t, BTList).(Bdd)
	if !ok {
		// can happen for all or nothing case. No need to memoize them though I am not
		// sure if we reach that point we are at a valid state
		return nil, false
	}
	return bdd, true
}

func ContainsUndef(t SemType) bool {
	if t.some() == 0 {
		return (t.all() & (1 << BTUndef.Code())) != 0
	}
	switch data := getComplexSubtypeData(t, BTUndef).(type) {
	case allOrNothingSubtype:
		return data.isAll
	case *bool:
		return *data
	default:
		panic("unexpected subtype data")
	}
}

// CreateIsolated returns the top type of isolated values:
// `readonly | isolated object {}`. Used by isolation analysis to test
// whether an expression's static type is intrinsically isolated.
func CreateIsolated(cx Context) SemType {
	if IsZero(cx._isolatedMemo) {
		cx._isolatedMemo = Union(VAL_READONLY, createIsolatedObject(cx))
	}
	return cx._isolatedMemo
}
