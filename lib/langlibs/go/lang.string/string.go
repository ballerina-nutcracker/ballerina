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

package stringruntime

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/ballerina-nutcracker/ballerina/runtime"
	"github.com/ballerina-nutcracker/ballerina/runtime/extern"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
	"github.com/ballerina-nutcracker/ballerina/values"
)

const (
	orgName    = "ballerina"
	moduleName = "lang.string"
)

func stringLength(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	return int64(utf8.RuneCountInString(args[0].(string))), nil
}

// runeSearch finds substr in s starting from startIndex (a codepoint index,
// clamped to 0 when negative, per lang.string:indexOf semantics). It reports
// the match as a codepoint index rather than a byte offset, without ever
// allocating a copy of s.
func runeSearch(s, substr string, startIndex int64) (int64, bool) {
	if startIndex < 0 {
		startIndex = 0
	}
	// Walk to the byte offset of the startIndex'th rune, tracking how many
	// runes were skipped along the way (this is min(startIndex, RuneCount(s))
	// whether or not the loop finds an exact match, since it stops
	// incrementing once byteOffset is set but keeps counting otherwise).
	byteOffset := len(s)
	skipped := int64(0)
	for i := range s {
		if skipped == startIndex {
			byteOffset = i
			break
		}
		skipped++
	}
	// UTF-8 is self-synchronizing, so a byte-level match of a valid UTF-8
	// substring always starts on a codepoint boundary.
	matchOffset := strings.Index(s[byteOffset:], substr)
	if matchOffset < 0 {
		return 0, false
	}
	// Count only the unseen span up to the match; the skipped prefix's rune
	// count is already known, so it isn't rescanned.
	return skipped + int64(utf8.RuneCountInString(s[byteOffset:byteOffset+matchOffset])), true
}

func stringIndexOf(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	idx, found := runeSearch(args[0].(string), args[1].(string), args[2].(int64))
	if !found {
		return nil, nil
	}
	return idx, nil
}

func stringIncludes(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	_, found := runeSearch(args[0].(string), args[1].(string), args[2].(int64))
	return found, nil
}

func stringToBytes(byteArrTy semtypes.SemType) extern.NativeFunc {
	return func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
		return values.ByteSliceToList(byteArrTy, ctx.TypeEnv(), []byte(args[0].(string))), nil
	}
}

func stringFromBytes(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	list := args[0].(*values.List)
	data := list.ToByteSlice()
	if !utf8.Valid(data) {
		return values.NewErrorWithMessage("invalid UTF-8 byte array"), nil
	}
	return string(data), nil
}

func stringSubstring(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	s := args[0].(string)
	startIndex := args[1].(int64)
	endIndex := args[2].(int64)
	runes := []rune(s)
	length := int64(len(runes))
	if startIndex < 0 || startIndex > length || endIndex < startIndex || endIndex > length {
		panic(values.NewErrorWithMessage(fmt.Sprintf("string index out of range: startIndex=%d endIndex=%d length=%d", startIndex, endIndex, length)))
	}
	return string(runes[startIndex:endIndex]), nil
}

func stringEqualsIgnoreCaseASCII(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	s1 := args[0].(string)
	s2 := args[1].(string)
	return equalsIgnoreCaseASCII(s1, s2), nil
}

func stringToLowerASCII(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	s := args[0].(string)
	return mapASCII(s, func(r rune) rune {
		if r >= 'A' && r <= 'Z' {
			return r + 32
		}
		return r
	}), nil
}

func stringToUpperASCII(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	s := args[0].(string)
	return mapASCII(s, func(r rune) rune {
		if r >= 'a' && r <= 'z' {
			return r - 32
		}
		return r
	}), nil
}

// stringTrim strips ASCII whitespace only (space, \t, \n, \v, \f, \r);
// strings.TrimSpace is Unicode-aware and would also strip U+0085/U+00A0.
func stringTrim(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	s := args[0].(string)
	return strings.Trim(s, " \t\n\v\f\r"), nil
}

func initStringModule(rt *runtime.Runtime) {
	env := rt.GetTypeEnv()
	ld := semtypes.NewListDefinition()
	byteArrTy := ld.Define(env, nil, semtypes.ListRest(semtypes.Byte))

	runtime.RegisterExternFunction(rt, orgName, moduleName, "length", stringLength)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "indexOf", stringIndexOf)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "includes", stringIncludes)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "toBytes", stringToBytes(byteArrTy))
	runtime.RegisterExternFunction(rt, orgName, moduleName, "fromBytes", stringFromBytes)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "substring", stringSubstring)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "equalsIgnoreCaseAscii", stringEqualsIgnoreCaseASCII)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "toLowerAscii", stringToLowerASCII)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "toUpperAscii", stringToUpperASCII)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "trim", stringTrim)
}

// equalsIgnoreCaseASCII compares byte-for-byte rather than decoding runes:
// non-ASCII bytes (>= 0x80 in UTF-8, whether lead or continuation) never fall
// in the 'A'-'Z'/'a'-'z' ranges, so folding case per byte and requiring exact
// equality otherwise is safe, and it's zero-allocation unlike a []rune-based
// comparison.
func equalsIgnoreCaseASCII(s1, s2 string) bool {
	if len(s1) != len(s2) {
		return false
	}
	for i := 0; i < len(s1); i++ {
		a, b := s1[i], s2[i]
		if a == b {
			continue
		}
		if a >= 'A' && a <= 'Z' {
			a += 'a' - 'A'
		}
		if b >= 'A' && b <= 'Z' {
			b += 'a' - 'A'
		}
		if a != b {
			return false
		}
	}
	return true
}

func mapASCII(s string, f func(rune) rune) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		b.WriteRune(f(r))
	}
	return b.String()
}

func init() {
	runtime.RegisterModuleInitializer(initStringModule)
}
