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

package native

import (
	"crypto/sha256"
	"crypto/subtle"
	"fmt"
	"io"

	"ballerina/runtime"
	"ballerina/runtime/extern"
	"ballerina/values"
	"golang.org/x/crypto/hkdf"
)

func registerKdfFunctions(rt *runtime.Runtime, types cryptoTypes) {
	runtime.RegisterExternFunction(rt, orgName, moduleName, "hkdfSha256",
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			input := args[0].(*values.List).ToByteSlice()
			length := int(args[1].(int64))
			salt := args[2].(*values.List).ToByteSlice()
			info := args[3].(*values.List).ToByteSlice()
			if length <= 0 {
				return cryptoError("Error occurred while HKDF: length must be positive"), nil
			}
			var saltArg []byte
			if len(salt) > 0 {
				saltArg = salt
			}
			r := hkdf.New(sha256.New, input, saltArg, info)
			key := make([]byte, length)
			if _, err := io.ReadFull(r, key); err != nil {
				return cryptoError(fmt.Sprintf("Error occurred while HKDF: %s", err.Error())), nil
			}
			return values.ByteSliceToList(types.byteArrTy, ctx.TypeCtx, key), nil
		})
}

func registerUtilFunctions(rt *runtime.Runtime, _ cryptoTypes) {
	runtime.RegisterExternFunction(rt, orgName, moduleName, "equalConstantTime",
		func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
			a := hashValueToBytes(args[0])
			b := hashValueToBytes(args[1])
			return subtle.ConstantTimeCompare(a, b) == 1, nil
		})
}

// hashValueToBytes converts a Ballerina HashValue (byte[]|string) to Go []byte.
func hashValueToBytes(v values.BalValue) []byte {
	switch t := v.(type) {
	case *values.List:
		return t.ToByteSlice()
	case string:
		return []byte(t)
	default:
		return nil
	}
}
