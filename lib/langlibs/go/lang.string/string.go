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

func stringLength(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	return int64(utf8.RuneCountInString(args[0].(string))), nil
}

// runeSearch finds substr in s starting from startIndex, a codepoint index
// (clamped to 0 when negative); the returned index is a codepoint index too.
func runeSearch(s, substr string, startIndex int64) (int64, bool) {
	if startIndex < 0 {
		startIndex = 0
	}
	byteOffset := len(s)
	runeOffset := int64(0)
	for i := range s {
		if runeOffset == startIndex {
			byteOffset = i
			break
		}
		runeOffset++
	}
	// UTF-8 is self-synchronizing, so a byte-level match of a valid UTF-8
	// substring always starts on a codepoint boundary.
	matchOffset := strings.Index(s[byteOffset:], substr)
	if matchOffset < 0 {
		return 0, false
	}
	// runeOffset already counts the runes before byteOffset, so only the
	// span up to the match needs counting here.
	return runeOffset + int64(utf8.RuneCountInString(s[byteOffset:byteOffset+matchOffset])), true
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

// stringToBytesFn binds byteArrTy (resolved once at module init) into the
// extern.NativeFunc registered for toBytes.
func stringToBytesFn(byteArrTy semtypes.SemType) extern.NativeFunc {
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

func initStringModule(rt *runtime.Runtime) {
	env := rt.GetTypeEnv()
	ld := semtypes.NewListDefinition()
	byteArrTy := ld.Define(env, nil, semtypes.ListRest(semtypes.Byte))
	runtime.RegisterExternFunction(rt, orgName, moduleName, "length", stringLength)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "indexOf", stringIndexOf)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "includes", stringIncludes)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "toBytes", stringToBytesFn(byteArrTy))
	runtime.RegisterExternFunction(rt, orgName, moduleName, "fromBytes", stringFromBytes)
}

func init() {
	runtime.RegisterModuleInitializer(initStringModule)
}
