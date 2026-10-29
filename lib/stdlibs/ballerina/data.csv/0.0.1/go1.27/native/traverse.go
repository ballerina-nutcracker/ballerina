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

// Port of jBallerina's CsvTraversal: it converts rows that are already Ballerina values (the records of
// transform, the string lists of parseList, and parsed rows bound for a union type) to the target type.

package native

import (
	"slices"
	"strconv"
	"strings"

	"github.com/ballerina-nutcracker/ballerina/decimal"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
	"github.com/ballerina-nutcracker/ballerina/values"
)

// valueStatus is the outcome of converting one value: jBallerina's UnMappedValue and SkipMappedValue.
type valueStatus int

const (
	valueMapped valueStatus = iota
	valueUnmapped
	valueSkipped
)

type traversal struct {
	cfg      *config
	t        *typer
	conv     *converter
	failSafe *failSafeLogger

	shape                     *rowShape
	headers                   []string
	headersForArrayConversion []string
	addHeadersForOutput       bool
	isFirstRowIsHeader        bool
	isFirstRowInserted        bool
	rows                      []values.BalValue
	currentRowIndex           int

	fieldHierarchy map[string]*recordField
	currentField   *recordField

	// skipLines and skipErr are cfg.skipDataRows, which every row consults.
	skipLines skipSet
	skipErr   error
}

func newTraversal(cfg *config, t *typer, failSafe *failSafeLogger) *traversal {
	skipLines, skipErr := cfg.skipDataRows()
	return &traversal{
		cfg:            cfg,
		t:              t,
		conv:           cfg.converter(t),
		failSafe:       failSafe,
		fieldHierarchy: make(map[string]*recordField),
		skipLines:      skipLines,
		skipErr:        skipErr,
	}
}

// traverse converts source rows to root, a list type or a union of list types.
func (tr *traversal) traverse(source []values.BalValue, root semtypes.SemType) (values.BalValue, error) {
	if tr.cfg.headerRows > 1 && tr.cfg.customHeaders == nil {
		return nil, noCustomHeadersError()
	}
	return tr.traverseCsv(source, root)
}

func (tr *traversal) traverseCsv(source []values.BalValue, root semtypes.SemType) (values.BalValue, error) {
	rootInfo := tr.t.classify(root)
	switch rootInfo.kind {
	case kindUnion:
		for _, member := range rootInfo.members {
			memberInfo := tr.t.classify(member)
			if !memberInfo.isList() {
				continue
			}
			attempt := newTraversal(tr.cfg, tr.t, tr.failSafe)
			if result, err := attempt.traverseArray(source, memberInfo); err == nil {
				return result, nil
			}
		}
		return nil, sourceCannotConvertError(tr.t.typeName(root))
	case kindArray:
		return tr.traverseArray(source, rootInfo)
	}
	return nil, invalidTypeError(tr.t.typeName(root))
}

func (tr *traversal) traverseArray(source []values.BalValue, root typeInfo) (values.BalValue, error) {
	rootShape := tr.t.rowShapeOf(root)
	if rootShape.size != -1 && rootShape.size > len(source) {
		return nil, arraySizeError(len(source))
	}
	elem := tr.t.rowShapeOf(rootShape.restInfo)
	tr.shape = &elem
	var err error
	switch {
	case elem.isMapping():
		err = tr.traverseWithMappingRows(source, rootShape.size)
	case elem.isList():
		err = tr.traverseWithListRows(source, rootShape.size)
	case elem.kind == kindUnion:
		return tr.traverseWithUnionElement(source, root.ty, elem.members)
	default:
		return nil, sourceCannotConvertError(tr.t.typeName(root.ty))
	}
	if err != nil {
		return nil, err
	}
	return buildList(tr.t, root.ty, root.list, rootShape.readonly, tr.rows)
}

// traverseWithUnionElement tries each member of the element union against the whole input: the first
// member every row converts to wins.
func (tr *traversal) traverseWithUnionElement(source []values.BalValue, root semtypes.SemType,
	members []semtypes.SemType) (values.BalValue, error) {
	for _, member := range members {
		memberArray := defineArray(tr.t.tc.Env(), member)
		attempt := newTraversal(tr.cfg, tr.t, tr.failSafe)
		result, err := attempt.traverseCsv(source, memberArray)
		if err == nil {
			return result, nil
		}
		if tr.failSafe != nil && !isFatal(err) {
			if logErr := tr.failSafe.log(err, attempt.currentRowIndex+1, 1, rowsString(source)); logErr != nil {
				return nil, logErr
			}
		}
	}
	return nil, sourceCannotConvertError(tr.t.typeName(root))
}

func (tr *traversal) isFull(rootSize int) bool {
	return rootSize != -1 && rootSize-1 < len(tr.rows)
}

func (tr *traversal) isSkipped(rowNumber int) (bool, error) {
	if tr.skipErr != nil {
		return false, tr.skipErr
	}
	return tr.skipLines.contains(int64(rowNumber)), nil
}

// tolerate logs a row error that failSafe allows and reports whether the traversal continues.
func (tr *traversal) tolerate(err error, row values.BalValue) error {
	if tr.failSafe == nil || isFatal(err) {
		return err
	}
	return tr.failSafe.log(err, tr.currentRowIndex+1, 1, javaValueOf(row))
}

func (tr *traversal) traverseWithMappingRows(source []values.BalValue, rootSize int) error {
	headerRows := int(tr.cfg.headerRows)
	rowNumber := 0
	for i, row := range source {
		tr.currentRowIndex = i
		if tr.isFull(rootSize) {
			break
		}
		if i < headerRows && i != headerRows-1 {
			continue
		}
		err := tr.addMappingRow(i, row, &rowNumber)
		if err != nil {
			if err := tr.tolerate(err, row); err != nil {
				return err
			}
			if i >= headerRows {
				rowNumber++
			}
		}
	}
	return nil
}

func (tr *traversal) addMappingRow(i int, row values.BalValue, rowNumber *int) error {
	headerRows := int(tr.cfg.headerRows)
	if i >= headerRows {
		skip, err := tr.isSkipped(*rowNumber + 1)
		if err != nil {
			return err
		}
		if skip {
			*rowNumber++
			return nil
		}
	}
	value, err := tr.mappingRow(row)
	if err != nil {
		return err
	}
	if !tr.isFirstRowIsHeader {
		tr.rows = append(tr.rows, value)
	}
	if i >= headerRows {
		*rowNumber++
	}
	return nil
}

func (tr *traversal) traverseWithListRows(source []values.BalValue, rootSize int) error {
	rowNumber := 0
	for i, row := range source {
		tr.currentRowIndex = i
		if tr.isFull(rootSize) {
			break
		}
		err := tr.addListRow(i, row, &rowNumber)
		if err != nil {
			if err := tr.tolerate(err, row); err != nil {
				return err
			}
			rowNumber++
		}
	}
	return nil
}

func (tr *traversal) addListRow(i int, row values.BalValue, rowNumber *int) error {
	headerRows := int(tr.cfg.headerRows)
	_, isMap := row.(*values.Map)
	if !tr.addHeadersForOutput && tr.cfg.outputWithHeaders &&
		(isMap || tr.cfg.customHeaders != nil || i == headerRows-1) {
		if err := tr.insertHeaderRow(row); err != nil {
			return err
		}
	}
	if i < headerRows {
		return nil
	}
	skip, err := tr.isSkipped(*rowNumber + 1)
	if err != nil {
		return err
	}
	if skip {
		*rowNumber++
		return nil
	}
	value, err := tr.listRow(row)
	if err != nil {
		return err
	}
	if !tr.isFirstRowIsHeader {
		tr.rows = append(tr.rows, value)
	}
	*rowNumber++
	return nil
}

func (tr *traversal) insertHeaderRow(row values.BalValue) error {
	if tr.headers == nil {
		switch row := row.(type) {
		case *values.List:
			headers, err := tr.headersForList(row)
			if err != nil {
				return err
			}
			tr.headers = headers
		case *values.Map:
			headers, err := tr.keysForMappingRow(row)
			if err != nil {
				return err
			}
			tr.headers = headers
		}
	}
	builder := newRowBuilder(tr.shape)
	for i, header := range tr.headers {
		memberType, ok, err := tr.elementType(i)
		if err != nil {
			return err
		}
		if !ok {
			break
		}
		if err := tr.insertToList(builder, header, memberType, i); err != nil {
			return err
		}
	}
	if tr.isFirstRowIsHeader {
		return nil
	}
	value, err := builder.build(tr.t)
	if err != nil {
		return err
	}
	tr.rows = append(tr.rows, value)
	tr.addHeadersForOutput = true
	return nil
}

// headersForList is CsvUtils.createHeadersForParseLists: custom headers, the header row, or "1", "2", ...
func (tr *traversal) headersForList(row *values.List) ([]string, error) {
	if tr.cfg.customHeaders != nil {
		if len(tr.cfg.customHeaders) != row.Len() {
			return nil, invalidHeaderCountError()
		}
		return tr.cfg.customHeaders, nil
	}
	switch {
	case tr.cfg.headerRows == 1:
		headers := make([]string, row.Len())
		for i := range headers {
			headers[i] = javaValueOf(row.Get(i))
		}
		return headers, nil
	case tr.cfg.headerRows > 1:
		return nil, noCustomHeadersError()
	}
	headers := make([]string, row.Len())
	for i := range headers {
		headers[i] = strconv.Itoa(i + 1)
	}
	return headers, nil
}

// keysForMappingRow orders the keys of a record row for a list target: headerOrder, else the keys of the
// first row.
func (tr *traversal) keysForMappingRow(row *values.Map) ([]string, error) {
	if tr.cfg.headerOrder != nil {
		if len(tr.cfg.headerOrder) != row.Len() {
			return nil, invalidHeaderCountError()
		}
		return tr.cfg.headerOrder, nil
	}
	if tr.headersForArrayConversion == nil {
		tr.headersForArrayConversion = row.Keys()
	}
	return tr.headersForArrayConversion, nil
}

// elementType is the type of the list member at index; false means the value is projected away.
func (tr *traversal) elementType(index int) (typeInfo, bool, error) {
	if ti, ok := tr.shape.memberType(index); ok {
		return ti, true, nil
	}
	if tr.cfg.allowDataProjection {
		return typeInfo{}, false, nil
	}
	if tr.shape.kind == kindArray {
		return typeInfo{}, false, arraySizeError(tr.shape.size)
	}
	return typeInfo{}, false, tupleSizeError(tr.shape.size)
}

func (tr *traversal) listRow(row values.BalValue) (values.BalValue, error) {
	builder := newRowBuilder(tr.shape)
	expectedSize := tr.shape.size
	if tr.shape.hasRest && tr.shape.kind == kindTuple {
		expectedSize = -1
	}
	switch row := row.(type) {
	case *values.Map:
		if expectedSize == -1 {
			expectedSize = row.Len()
		}
		keys, err := tr.keysForMappingRow(row)
		if err != nil {
			return nil, err
		}
		for index, key := range keys {
			v, ok := row.Get(key)
			if !ok {
				return nil, varyingHeadersError()
			}
			if stop, err := tr.insertListMember(builder, v, index, expectedSize); err != nil {
				return nil, err
			} else if stop {
				break
			}
		}
	case *values.List:
		if expectedSize == -1 {
			expectedSize = row.Len()
		}
		for index := 0; index < row.Len(); index++ {
			if stop, err := tr.insertListMember(builder, row.Get(index), index, expectedSize); err != nil {
				return nil, err
			} else if stop {
				break
			}
		}
	}
	return builder.build(tr.t)
}

func (tr *traversal) insertListMember(builder *rowBuilder, v values.BalValue, index, expectedSize int) (bool, error) {
	if tr.cfg.allowDataProjection && index >= expectedSize {
		return true, nil
	}
	memberType, ok, err := tr.elementType(index)
	if err != nil || !ok {
		return false, err
	}
	return false, tr.insertToList(builder, v, memberType, index)
}

func (tr *traversal) insertToList(builder *rowBuilder, v values.BalValue, ti typeInfo, index int) error {
	tr.currentField = nil
	converted, status, err := tr.convertValue(ti, v, false)
	if err != nil {
		return err
	}
	if status == valueUnmapped {
		return invalidTypeForArrayError(formatMessageArg(v), index, tr.t.typeName(ti.ty))
	}
	builder.set(index, converted)
	return nil
}

func (tr *traversal) mappingRow(row values.BalValue) (values.BalValue, error) {
	builder := newRowBuilder(tr.shape)
	tr.fieldHierarchy = tr.shape.remainingFields()
	switch source := row.(type) {
	case *values.Map:
		for _, key := range source.Keys() {
			v, _ := source.Get(key)
			if err := tr.insertMappingMember(builder, key, v); err != nil {
				return nil, err
			}
		}
	default:
		row, _ := source.(*values.List)
		tr.isFirstRowIsHeader = false
		if tr.headers == nil {
			headers, err := tr.headersForList(row)
			if err != nil {
				return nil, err
			}
			tr.headers = headers
			if !tr.isFirstRowInserted && tr.cfg.headerRows >= 1 {
				tr.isFirstRowIsHeader = true
				tr.isFirstRowInserted = true
				return nil, nil
			}
		}
		if row.Len() < len(tr.headers) {
			return nil, varyingHeadersError()
		}
		if !tr.headersMatchRecord() {
			return nil, arrayToMapError(javaValueOf(row), tr.t.typeName(tr.shape.ty))
		}
		if row.Len() != len(tr.headers) {
			return nil, varyingHeadersError()
		}
		for i, key := range tr.headers {
			if err := tr.insertMappingMember(builder, key, row.Get(i)); err != nil {
				return nil, err
			}
		}
	}
	if err := tr.checkRequiredFields(); err != nil {
		return nil, err
	}
	return builder.build(tr.t)
}

// headersMatchRecord reports whether list rows can bind to the target: always for maps and open records,
// and for a closed record when at least one header names one of its fields.
func (tr *traversal) headersMatchRecord() bool {
	if tr.shape.kind == kindMap || tr.shape.hasRest {
		return true
	}
	for _, header := range tr.headers {
		if _, ok := tr.fieldHierarchy[header]; ok {
			return true
		}
	}
	return false
}

func (tr *traversal) insertMappingMember(builder *rowBuilder, key string, v values.BalValue) error {
	ti := tr.shape.restInfo
	tr.currentField = nil
	if tr.shape.kind == kindRecord {
		f, declared := tr.fieldHierarchy[key]
		if !declared {
			if tr.shape.hasRest {
				converted, status, err := tr.convertValue(tr.shape.restInfo, v, false)
				if err != nil {
					return err
				}
				if status != valueUnmapped {
					builder.put(key, converted)
				}
				return nil
			}
			if tr.cfg.allowDataProjection {
				return nil
			}
			return noFieldForHeaderError(key)
		}
		delete(tr.fieldHierarchy, key)
		tr.currentField = f
		ti = f.info
	}
	converted, status, err := tr.convertValue(ti, v, false)
	switch {
	case err != nil:
		return err
	case status == valueMapped:
		builder.put(key, converted)
		return nil
	case tr.shape.kind == kindMap || status == valueSkipped:
		return nil
	}
	return invalidTypeForFieldError(formatMessageArg(v), key)
}

func (tr *traversal) checkRequiredFields() error {
	for _, name := range tr.shape.fieldOrder {
		f, remaining := tr.fieldHierarchy[name]
		if !remaining || (tr.cfg.absentAsNilableType && f.nilable) {
			continue
		}
		if !f.optional {
			return missingFieldError(f.name)
		}
	}
	return nil
}

// convertValue is convertCsvValueIntoExpectedType. A string value is converted with FromString when
// stringConversion is set; otherwise only values whose own type is compatible with the target convert.
func (tr *traversal) convertValue(ti typeInfo, v values.BalValue, recursive bool) (values.BalValue, valueStatus, error) {
	if !recursive && tr.cfg.nilAsOptionalField && !tr.t.isNilable(ti.ty) && tr.isNullValue(v) &&
		tr.currentField != nil && tr.currentField.optional {
		return nil, valueSkipped, nil
	}
	if s, isString := v.(string); isString && tr.cfg.stringConversion {
		if converted, ok := tr.conv.fromString(s, ti); ok {
			return converted, valueMapped, nil
		}
		return nil, valueUnmapped, nil
	}
	switch ti.kind {
	case kindUnion:
		for _, member := range scalarsFirst(ti.memberInfos) {
			if !member.isBasic() {
				return nil, valueUnmapped, onlyBasicTypesError(tr.t.typeName(member.ty))
			}
			converted, status, err := tr.convertValue(member, v, true)
			if err != nil {
				return nil, status, err
			}
			if status == valueMapped {
				return converted, valueMapped, nil
			}
		}
		return nil, valueUnmapped, nil
	case kindRecord, kindMap, kindArray, kindTuple, kindUnsupported:
		return nil, valueUnmapped, invalidExpectedTypeError(tr.t.typeName(ti.ty))
	}
	if !isCompatible(ti.kind, v) {
		return nil, valueUnmapped, nil
	}
	converted, err := tr.convertBasic(ti, v)
	if err != nil {
		return nil, valueUnmapped, err
	}
	return converted, valueMapped, nil
}

func (tr *traversal) isNullValue(v values.BalValue) bool {
	switch v := v.(type) {
	case nil:
		return true
	case string:
		return tr.conv.isNullValue(v)
	}
	return false
}

// scalarsFirst orders union members with the basic types before the structured ones. Declaration order,
// which jBallerina follows, is not available here.
func scalarsFirst(members []typeInfo) []typeInfo {
	ordered := slices.Clone(members)
	slices.SortStableFunc(ordered, func(a, b typeInfo) int {
		return structuredRank(a) - structuredRank(b)
	})
	return ordered
}

func structuredRank(ti typeInfo) int {
	if ti.isBasic() {
		return 0
	}
	return 1
}

// isCompatible is CsvUtils.checkTypeCompatibility for a value that is not converted from a string.
func isCompatible(kind typeKind, v values.BalValue) bool {
	switch v.(type) {
	case string:
		return kind == kindString || kind == kindChar || kind == kindJSON || kind == kindAnydata
	case int64, float64, *decimal.Decimal:
		return kind == kindInt || kind == kindIntSubtype || kind == kindFloat || kind == kindDecimal ||
			kind == kindJSON || kind == kindAnydata
	case bool:
		return kind == kindBoolean || kind == kindJSON || kind == kindAnydata
	case nil:
		return kind == kindNil || kind == kindJSON || kind == kindAnydata
	}
	return false
}

// convertBasic is ValueUtils.convert for a compatible basic value. A numeric value out of the range of
// the target, or a string that is not a single character for a string:Char target, fails the whole call
// with the runtime's conversion error, as in jBallerina.
func (tr *traversal) convertBasic(ti typeInfo, v values.BalValue) (values.BalValue, error) {
	switch ti.kind {
	case kindJSON, kindAnydata, kindString, kindBoolean, kindNil:
		return v, nil
	}
	converted, convErr := values.CloneWithType(tr.t.tc, v, ti.ty)
	if convErr != nil {
		return nil, &conversionError{value: convErr}
	}
	return converted, nil
}

// rowsString renders the whole source the way jBallerina logs it when a union member fails.
func rowsString(rows []values.BalValue) string {
	rendered := make([]string, len(rows))
	for i, row := range rows {
		rendered[i] = javaValueOf(row)
	}
	return "[" + strings.Join(rendered, ",") + "]"
}
