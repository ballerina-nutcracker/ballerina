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

// Port of jBallerina's FromString and of the token conversions in JsonCreator: they turn the text of a
// JSON scalar token into a value of the expected type.

package native

import (
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/ballerina-nutcracker/ballerina/decimal"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
	"github.com/ballerina-nutcracker/ballerina/values"
)

var (
	javaDoublePattern = regexp.MustCompile(
		`^[+-]?(NaN|Infinity|((\d+(\.\d*)?|\.\d+)([eE][+-]?\d+)?)|(0[xX]([0-9a-fA-F]+(\.[0-9a-fA-F]*)?|\.[0-9a-fA-F]+)[pP][+-]?\d+))[fFdD]?$`)
	javaBigDecimalPattern = regexp.MustCompile(`^[+-]?(\d+(\.\d*)?|\.\d+)([eE][+-]?\d+)?$`)

	basicTypesWithoutString = []semtypes.SemType{semtypes.Nil, semtypes.Boolean, semtypes.Int, semtypes.Float, semtypes.Decimal}
	basicJSONTypes          = []semtypes.SemType{semtypes.Nil, semtypes.Boolean, semtypes.Int, semtypes.Float, semtypes.Decimal, semtypes.String}
)

// unionPriority is FromString's TYPE_PRIORITY_ORDER; any other member keeps its place after these.
var unionPriority = map[typeKind]int{
	kindInt:     0,
	kindFloat:   1,
	kindDecimal: 2,
	kindNil:     3,
	kindBoolean: 4,
	kindJSON:    5,
	kindString:  6,
}

// convertStringToken converts the content of a JSON string token.
func (t *typer) convertStringToken(value string, ty semtypes.SemType) (values.BalValue, error) {
	ti := t.classify(ty)
	switch ti.kind {
	case kindString, kindAnydata, kindJSON:
		return value, nil
	case kindChar:
		if utf8.RuneCountInString(value) != 1 {
			return nil, &returnedError{}
		}
		return value, nil
	case kindFinite:
		if semtypes.IsSubtypeSimple(ty, semtypes.String) && t.contains(ty, value) {
			return value, nil
		}
		return nil, &returnedError{}
	case kindUnion:
		for _, member := range ti.members {
			if converted, err := t.convertStringToken(value, member); err == nil {
				return converted, nil
			}
		}
		return nil, &returnedError{}
	default:
		return nil, &returnedError{}
	}
}

// convertNonStringToken converts the text of a JSON number, boolean or null token.
func (t *typer) convertNonStringToken(value string, ty semtypes.SemType) (values.BalValue, error) {
	typeName := t.typeName(ty)
	switch {
	case value == "":
		return nil, incompatibleTypeError(typeName, value)
	case value[0] == 't' && value != "true", value[0] == 'f' && value != "false", value[0] == 'n' && value != "null":
		return nil, incompatibleTypeError(typeName, value)
	case !strings.ContainsRune("tfn+-0123456789", rune(value[0])):
		return nil, incompatibleTypeError(typeName, value)
	}
	return t.convertNonStringValue(value, ty)
}

func (t *typer) convertNonStringValue(value string, ty semtypes.SemType) (values.BalValue, error) {
	ti := t.classify(ty)
	switch ti.kind {
	case kindAnydata, kindJSON:
		return t.stringToUnion(value, basicTypesWithoutString, ty)
	case kindString, kindChar:
		return nil, &returnedError{}
	case kindFinite:
		return t.stringToSingleton(value, ti, false)
	case kindUnion:
		members := slices.DeleteFunc(slices.Clone(ti.members), func(member semtypes.SemType) bool {
			return t.classify(member).kind == kindString
		})
		return t.stringToUnion(value, members, ty)
	default:
		return t.fromString(value, ti)
	}
}

// fromString is FromString.fromStringWithType. A *returnedError result is FromString's returned error
// value; a *jsonError is an error it throws.
func (t *typer) fromString(value string, ti typeInfo) (values.BalValue, error) {
	switch ti.kind {
	case kindInt:
		if n, ok := parseJavaLong(value); ok {
			return n, nil
		}
	case kindIntSubtype:
		n, ok := parseJavaLong(value)
		if !ok {
			break
		}
		if !t.contains(ti.ty, n) {
			return nil, incompatibleTypeError(t.typeName(ti.ty), value)
		}
		return n, nil
	case kindFloat:
		if f, ok := parseJavaDouble(value); ok {
			return f, nil
		}
	case kindDecimal:
		if d, ok := parseJavaBigDecimal(value); ok {
			return d, nil
		}
	case kindChar:
		if utf8.RuneCountInString(value) != 1 {
			return nil, incompatibleTypeError(t.typeName(ti.ty), value)
		}
		return value, nil
	case kindString:
		return value, nil
	case kindBoolean:
		switch {
		case strings.EqualFold(value, "true") || value == "1":
			return true, nil
		case strings.EqualFold(value, "false") || value == "0":
			return false, nil
		}
		return nil, cannotConvertError(value, "boolean")
	case kindNil:
		if strings.EqualFold(value, "null") || value == "()" {
			return nil, nil
		}
		return nil, cannotConvertError(value, "()")
	case kindFinite:
		return t.stringToSingleton(value, ti, true)
	case kindUnion:
		return t.stringToUnion(value, ti.members, ti.ty)
	case kindJSON, kindAnydata:
		return t.stringToUnion(value, basicJSONTypes, ti.ty)
	}
	return nil, cannotConvertError(value, t.typeName(ti.ty))
}

// stringToSingleton matches value against the singletons of a finite type by their string form, as
// jBallerina does, and converts it to the type of the matching singleton. Only FromString lets a non-string
// token match a string singleton.
func (t *typer) stringToSingleton(value string, ti typeInfo, allowString bool) (values.BalValue, error) {
	for _, basic := range scalarBasicTypes {
		if !semtypes.IsSubtypeSimple(ti.ty, basic.ty) {
			continue
		}
		if basic.kind == kindString && !allowString {
			break
		}
		converted, err := t.fromString(value, typeInfo{kind: basic.kind, ty: basic.ty})
		if err == nil && t.contains(ti.ty, converted) && javaValueOf(converted) == value {
			return converted, nil
		}
	}
	return nil, cannotConvertError(value, t.typeName(ti.ty))
}

func (t *typer) stringToUnion(value string, members []semtypes.SemType, unionTy semtypes.SemType) (values.BalValue, error) {
	infos := make([]typeInfo, len(members))
	for i, member := range members {
		infos[i] = t.classify(member)
	}
	slices.SortStableFunc(infos, func(a, b typeInfo) int {
		return priorityOf(a) - priorityOf(b)
	})
	for _, info := range infos {
		if converted, err := t.fromString(value, info); err == nil {
			return converted, nil
		}
	}
	return nil, cannotConvertError(value, t.typeName(unionTy))
}

func priorityOf(ti typeInfo) int {
	if p, ok := unionPriority[ti.kind]; ok {
		return p
	}
	return len(unionPriority)
}

func parseJavaLong(value string) (int64, bool) {
	n, err := strconv.ParseInt(value, 10, 64)
	return n, err == nil
}

// parseJavaDouble accepts what Double.parseDouble accepts, except the float and double suffixes, which
// FromString rejects.
func parseJavaDouble(value string) (float64, bool) {
	if strings.ContainsAny(value[max(len(value)-1, 0):], "fFdD") {
		return 0, false
	}
	trimmed := strings.TrimFunc(value, func(r rune) bool { return r <= ' ' })
	if !javaDoublePattern.MatchString(trimmed) {
		return 0, false
	}
	if strings.TrimLeft(trimmed, "+-") == "NaN" {
		trimmed = "NaN"
	}
	f, err := strconv.ParseFloat(trimmed, 64)
	if err != nil && !isRangeError(err) {
		return 0, false
	}
	return f, true
}

func isRangeError(err error) bool {
	numErr, ok := err.(*strconv.NumError)
	return ok && numErr.Err == strconv.ErrRange
}

// parseJavaBigDecimal accepts what new BigDecimal(String) accepts and rounds it to decimal128.
func parseJavaBigDecimal(value string) (*decimal.Decimal, bool) {
	if !javaBigDecimalPattern.MatchString(value) {
		return nil, false
	}
	d, err := decimal.FromLiteral(value)
	return d, err == nil
}
