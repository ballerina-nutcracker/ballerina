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

	"ballerina-lang-go/runtime"
	"ballerina-lang-go/runtime/extern"
	"ballerina-lang-go/semtypes"
	"ballerina-lang-go/values"
)

const (
	orgName    = "ballerina"
	moduleName = "lang.string"
)

func stringLength(args []values.BalValue) (values.BalValue, error) {
	return int64(utf8.RuneCountInString(args[0].(string))), nil
}

func stringToBytes(byteArrTy semtypes.SemType, ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
	return values.ByteSliceToList(byteArrTy, ctx.TypeCtx, []byte(args[0].(string))), nil
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
	byteArrTy := ld.DefineListTypeWrappedWithEnvSemType(env, semtypes.BYTE)

	runtime.RegisterExternFunction(rt, orgName, moduleName, "length", func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
		return stringLength(args)
	})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "toBytes", func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
		return stringToBytes(byteArrTy, ctx, args)
	})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "fromBytes", func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
		return stringFromBytes(args)
	})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "substring", func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
		s := args[0].(string)
		startIndex := args[1].(int64)
		endIndex := args[2].(int64)
		runes := []rune(s)
		length := int64(len(runes))
		if startIndex < 0 || startIndex > length || endIndex < startIndex || endIndex > length {
			panic(values.NewErrorWithMessage(fmt.Sprintf("string index out of range: startIndex=%d endIndex=%d length=%d", startIndex, endIndex, length)))
		}
		return string(runes[startIndex:endIndex]), nil
	})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "equalsIgnoreCaseAscii", func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
		s1 := args[0].(string)
		s2 := args[1].(string)
		return equalsIgnoreCaseASCII(s1, s2), nil
	})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "toLowerAscii", func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
		s := args[0].(string)
		return mapASCII(s, func(r rune) rune {
			if r >= 'A' && r <= 'Z' {
				return r + 32
			}
			return r
		}), nil
	})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "toUpperAscii", func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
		s := args[0].(string)
		return mapASCII(s, func(r rune) rune {
			if r >= 'a' && r <= 'z' {
				return r - 32
			}
			return r
		}), nil
	})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "trim", func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
		s, ok := args[0].(string)
		if !ok {
			return nil, fmt.Errorf("first argument must be a string")
		}
		return strings.TrimSpace(s), nil
	})

}

func equalsIgnoreCaseASCII(s1, s2 string) bool {
	r1 := []rune(s1)
	r2 := []rune(s2)
	if len(r1) != len(r2) {
		return false
	}
	for i, a := range r1 {
		b := r2[i]
		if a == b {
			continue
		}
		if a >= 'A' && a <= 'Z' && a+32 == b {
			continue
		}
		if a >= 'a' && a <= 'z' && a-32 == b {
			continue
		}
		return false
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
