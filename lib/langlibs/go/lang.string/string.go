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

func stringLength(args []values.BalValue) (values.BalValue, error) {
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

func stringIndexOf(args []values.BalValue) (values.BalValue, error) {
	idx, found := runeSearch(args[0].(string), args[1].(string), args[2].(int64))
	if !found {
		return nil, nil
	}
	return idx, nil
}

func stringIncludes(args []values.BalValue) (values.BalValue, error) {
	_, found := runeSearch(args[0].(string), args[1].(string), args[2].(int64))
	return found, nil
}

func stringToBytes(byteArrTy semtypes.SemType, ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
	return values.ByteSliceToList(byteArrTy, ctx.TypeEnv(), []byte(args[0].(string))), nil
}

func stringFromBytes(args []values.BalValue) (values.BalValue, error) {
	list := args[0].(*values.List)
	data := list.ToByteSlice()
	if !utf8.Valid(data) {
		return values.NewErrorWithMessage("invalid UTF-8 byte array"), nil
	}
	return string(data), nil
}

func initStringModule(rt *runtime.Runtime) {
	env := rt.GetTypeEnv()
	ld := semtypes.NewListDefinition()
	byteArrTy := ld.Define(env, nil, semtypes.ListRest(semtypes.Byte))

	runtime.RegisterExternFunction(rt, orgName, moduleName, "length", func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
		return stringLength(args)
	})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "indexOf", func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
		return stringIndexOf(args)
	})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "includes", func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
		return stringIncludes(args)
	})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "toBytes", func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
		return stringToBytes(byteArrTy, ctx, args)
	})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "fromBytes", func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
		return stringFromBytes(args)
	})
}

func init() {
	runtime.RegisterModuleInitializer(initStringModule)
}
