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

// Port of jBallerina's FromString and of the string conversions in DataUtils and XmlParser.

package native

import (
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/ballerina-nutcracker/ballerina/decimal"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
	"github.com/ballerina-nutcracker/ballerina/values"
)

var (
	javaDoublePattern = regexp.MustCompile(
		`^[+-]?(NaN|Infinity|((\d+(\.\d*)?|\.\d+)([eE][+-]?\d+)?)|(0[xX]([0-9a-fA-F]+(\.[0-9a-fA-F]*)?|\.[0-9a-fA-F]+)[pP][+-]?\d+))[fFdD]?$`)
	javaBigDecimalPattern = regexp.MustCompile(`^[+-]?(\d+(\.\d*)?|\.\d+)([eE][+-]?\d+)?$`)
)

// unionPriority is FromString's TYPE_PRIORITY_ORDER. A member that is not listed sorts before every listed
// one, as List.indexOf returns -1 for it.
var unionPriority = map[typeKind]int{
	kindInt:     0,
	kindFloat:   1,
	kindDecimal: 2,
	kindNil:     3,
	kindBoolean: 4,
	kindJSON:    5,
	kindString:  6,
}

func priorityOf(ti typeInfo) int {
	if p, ok := unionPriority[ti.kind]; ok {
		return p
	}
	return -1
}

var basicJSONTypes = []semtypes.SemType{semtypes.Nil, semtypes.Boolean, semtypes.Int, semtypes.Float, semtypes.Decimal,
	semtypes.String}

// fromString is FromString.fromStringWithType.
func (t *typer) fromString(value string, ty semtypes.SemType) (values.BalValue, error) {
	ti := t.classify(ty)
	switch ti.kind {
	case kindString:
		return value, nil
	case kindInt:
		if n, ok := parseJavaLong(value); ok {
			return n, nil
		}
	case kindByte:
		if n, ok := parseJavaLong(value); ok && n >= 0 && n <= 255 {
			return n, nil
		}
	case kindFloat:
		if f, ok := parseJavaDouble(value, false); ok {
			return f, nil
		}
	case kindDecimal:
		if d, ok := parseJavaBigDecimal(value); ok {
			return d, nil
		}
	case kindBoolean:
		switch {
		case strings.EqualFold(value, "true") || value == "1":
			return true, nil
		case strings.EqualFold(value, "false") || value == "0":
			return false, nil
		}
		return nil, cannotConvertError("string", value, "boolean")
	case kindNil:
		if strings.EqualFold(value, "null") || value == "()" {
			return nil, nil
		}
		return nil, cannotConvertError("string", value, "()")
	case kindUnion:
		return t.stringToUnion(value, ti.members, ty)
	case kindJSON:
		return t.stringToUnion(value, basicJSONTypes, ty)
	case kindFinite:
		return t.stringToFinite(value, ti)
	case kindXMLText:
		return values.NewXMLText(value), nil
	}
	return nil, cannotConvertError("string", value, t.typeName(ty))
}

func (t *typer) stringToUnion(value string, members []semtypes.SemType, unionTy semtypes.SemType) (values.BalValue, error) {
	infos := make([]typeInfo, len(members))
	for i, member := range members {
		infos[i] = t.classify(member)
	}
	slices.SortStableFunc(infos, func(a, b typeInfo) int { return priorityOf(a) - priorityOf(b) })
	for _, info := range infos {
		if converted, err := t.fromString(value, info.ty); err == nil {
			return converted, nil
		}
	}
	return nil, cannotConvertError("string", value, t.typeName(unionTy))
}

// stringToFinite matches value against the singletons of a finite type the way jBallerina does: decimals and
// floats compare numerically, every other singleton by its string form.
func (t *typer) stringToFinite(value string, ti typeInfo) (values.BalValue, error) {
	for _, candidate := range t.singletonCandidates(value, ti.ty) {
		if t.contains(ti.ty, candidate.match) {
			return t.fromString(value, candidate.basic)
		}
	}
	return nil, cannotConvertError("string", value, t.typeName(ti.ty))
}

type singletonCandidate struct {
	match values.BalValue
	basic semtypes.SemType
}

func (t *typer) singletonCandidates(value string, ty semtypes.SemType) []singletonCandidate {
	var candidates []singletonCandidate
	has := func(basic semtypes.SemType) bool {
		return !semtypes.IsEmpty(t.tc, semtypes.Intersect(ty, basic))
	}
	if has(semtypes.String) {
		candidates = append(candidates, singletonCandidate{value, semtypes.String})
	}
	if n, ok := parseJavaLong(value); ok && has(semtypes.Int) && strconv.FormatInt(n, 10) == value {
		candidates = append(candidates, singletonCandidate{n, semtypes.Int})
	}
	if f, ok := parseJavaDouble(value, true); ok && has(semtypes.Float) {
		candidates = append(candidates, singletonCandidate{f, semtypes.Float})
	}
	if d, ok := parseJavaBigDecimal(value); ok && has(semtypes.Decimal) {
		candidates = append(candidates, singletonCandidate{d, semtypes.Decimal})
	}
	if (value == "true" || value == "false") && has(semtypes.Boolean) {
		candidates = append(candidates, singletonCandidate{value == "true", semtypes.Boolean})
	}
	if value == "null" && has(semtypes.Nil) {
		candidates = append(candidates, singletonCandidate{nil, semtypes.Nil})
	}
	return candidates
}

// convertStringToExpType is DataUtils.convertStringToExpType, used by the traversal.
func (t *typer) convertStringToExpType(value string, ty semtypes.SemType) (values.BalValue, error) {
	ti := t.classify(ty)
	switch ti.kind {
	case kindAnydata, kindJSON:
		return t.fromString(value, t.jsonTy)
	case kindArray:
		return t.convertStringToExpType(value, ti.element)
	case kindUnion:
		for _, member := range ti.members {
			if converted, err := t.convertStringToExpType(value, member); err == nil {
				return converted, nil
			}
		}
		return nil, unionFieldCannotConvertError(t.typeName(ty))
	case kindXML, kindXMLElement, kindXMLText, kindXMLOther:
		return nil, cannotConvertError("string", value, t.typeName(ty))
	}
	return t.fromString(value, ty)
}

// parserConvertStringToExpType is XmlParser.convertStringToExpType: it does not special-case anydata or json.
func (t *typer) parserConvertStringToExpType(value string, ty semtypes.SemType) (values.BalValue, error) {
	if ti := t.classify(ty); ti.kind == kindArray {
		ty = ti.element
	}
	return t.fromString(value, ty)
}

// convertStringToRestExpType is XmlParser.convertStringToRestExpType.
func (t *typer) convertStringToRestExpType(value string, ty semtypes.SemType) (values.BalValue, error) {
	ti := t.classify(ty)
	switch ti.kind {
	case kindArray:
		return t.convertStringToRestExpType(value, ti.element)
	case kindInt, kindFloat, kindDecimal, kindString, kindBoolean, kindUnion, kindFinite:
		return t.parserConvertStringToExpType(value, ty)
	case kindAnydata, kindJSON:
		return t.parserConvertStringToExpType(value, t.jsonTy)
	}
	return nil, invalidRestTypeError(t.typeName(ty))
}

func parseJavaLong(value string) (int64, bool) {
	n, err := strconv.ParseInt(value, 10, 64)
	return n, err == nil
}

// parseJavaDouble accepts what Double.parseDouble accepts; FromString additionally rejects a float or
// double suffix, which Double.valueOf tolerates.
func parseJavaDouble(value string, allowSuffix bool) (float64, bool) {
	if value == "" || !allowSuffix && strings.ContainsAny(value[len(value)-1:], "fFdD") {
		return 0, false
	}
	trimmed := strings.TrimFunc(value, func(r rune) bool { return r <= ' ' })
	if !javaDoublePattern.MatchString(trimmed) {
		return 0, false
	}
	trimmed = strings.TrimRight(trimmed, "fFdD")
	f, err := strconv.ParseFloat(trimmed, 64)
	if err != nil {
		if numErr, ok := err.(*strconv.NumError); !ok || numErr.Err != strconv.ErrRange {
			return 0, false
		}
	}
	return f, true
}

func parseJavaBigDecimal(value string) (*decimal.Decimal, bool) {
	if !javaBigDecimalPattern.MatchString(value) {
		return nil, false
	}
	d, err := decimal.FromLiteral(value)
	return d, err == nil
}
