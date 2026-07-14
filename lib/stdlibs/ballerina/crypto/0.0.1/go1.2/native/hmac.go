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
	"crypto/hmac"
	"crypto/md5"  //nolint:gosec
	"crypto/sha1" //nolint:gosec
	"crypto/sha256"
	"crypto/sha512"
	"hash"

	"ballerina-lang-go/runtime"
	"ballerina-lang-go/runtime/extern"
	"ballerina-lang-go/values"
)

func registerHmacFunctions(rt *runtime.Runtime, types cryptoTypes) {
	runtime.RegisterExternFunction(rt, orgName, moduleName, "hmacMd5",
		hmacFunc(func() hash.Hash { return md5.New() }, types)) //nolint:gosec

	runtime.RegisterExternFunction(rt, orgName, moduleName, "hmacSha1",
		hmacFunc(func() hash.Hash { return sha1.New() }, types)) //nolint:gosec

	runtime.RegisterExternFunction(rt, orgName, moduleName, "hmacSha256",
		hmacFunc(func() hash.Hash { return sha256.New() }, types))

	runtime.RegisterExternFunction(rt, orgName, moduleName, "hmacSha384",
		hmacFunc(func() hash.Hash { return sha512.New384() }, types))

	runtime.RegisterExternFunction(rt, orgName, moduleName, "hmacSha512",
		hmacFunc(func() hash.Hash { return sha512.New() }, types))
}

func hmacFunc(newHash func() hash.Hash, types cryptoTypes) extern.NativeFunc {
	return func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
		input := listToBytes(args[0].(*values.List))
		key := listToBytes(args[1].(*values.List))
		h := hmac.New(newHash, key)
		h.Write(input)
		return bytesToList(types.byteArrTy, ctx, h.Sum(nil)), nil
	}
}
