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
	"crypto/md5"  //nolint:gosec
	"crypto/sha1" //nolint:gosec
	"crypto/sha256"
	"crypto/sha512"
	"fmt"
	"hash"
	"hash/crc32"

	"ballerina/runtime"
	"ballerina/runtime/extern"
	"ballerina/values"
	"golang.org/x/crypto/sha3"
)

func registerHashFunctions(rt *runtime.Runtime, types cryptoTypes) {
	runtime.RegisterExternFunction(rt, orgName, moduleName, "hashMd5",
		hashFunc(func() hash.Hash { return md5.New() }, types)) //nolint:gosec

	runtime.RegisterExternFunction(rt, orgName, moduleName, "hashSha1",
		hashFunc(func() hash.Hash { return sha1.New() }, types)) //nolint:gosec

	runtime.RegisterExternFunction(rt, orgName, moduleName, "hashSha256",
		hashFunc(func() hash.Hash { return sha256.New() }, types))

	runtime.RegisterExternFunction(rt, orgName, moduleName, "hashSha384",
		hashFunc(func() hash.Hash { return sha512.New384() }, types))

	runtime.RegisterExternFunction(rt, orgName, moduleName, "hashSha512",
		hashFunc(func() hash.Hash { return sha512.New() }, types))

	runtime.RegisterExternFunction(rt, orgName, moduleName, "hashKeccak256",
		hashFunc(func() hash.Hash { return sha3.NewLegacyKeccak256() }, types))

	runtime.RegisterExternFunction(rt, orgName, moduleName, "crc32b",
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			input := args[0].(*values.List).ToByteSlice()
			sum := crc32.ChecksumIEEE(input)
			return fmt.Sprintf("%08X", sum), nil
		})
}

// hashFunc returns an extern that computes a hash using the given factory.
// Salt (args[1]) is written first, then input (args[0]) — matching jBallerina.
func hashFunc(newHash func() hash.Hash, types cryptoTypes) extern.NativeFunc {
	return func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
		input := args[0].(*values.List).ToByteSlice()
		h := newHash()
		if args[1] != nil {
			salt := args[1].(*values.List).ToByteSlice()
			h.Write(salt)
		}
		h.Write(input)
		return values.ByteSliceToList(types.byteArrTy, ctx.TypeCtx, h.Sum(nil)), nil
	}
}
