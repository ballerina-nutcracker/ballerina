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
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"fmt"

	"ballerina-lang-go/runtime"
	"ballerina-lang-go/runtime/extern"
	"ballerina-lang-go/values"
)

// validGcmTagSizes mirrors Java's VALID_GCM_TAG_SIZES (in bits).
var validGcmTagSizes = map[int64]bool{32: true, 64: true, 96: true, 104: true, 112: true, 120: true, 128: true}

func registerAesFunctions(rt *runtime.Runtime, types cryptoTypes) {
	runtime.RegisterExternFunction(rt, orgName, moduleName, "encryptAesCbc",
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			input := listToBytes(args[0].(*values.List))
			key := listToBytes(args[1].(*values.List))
			iv := listToBytes(args[2].(*values.List))
			padding, _ := args[3].(string)
			block, err := aes.NewCipher(key)
			if err != nil {
				return cryptoError(fmt.Sprintf("Error occurred while AES-CBC encrypt: %s", err.Error())), nil
			}
			padded := pkcs7Pad(input, aes.BlockSize)
			_ = padding
			out := make([]byte, len(padded))
			cipher.NewCBCEncrypter(block, iv).CryptBlocks(out, padded)
			return bytesToList(types.byteArrTy, ctx, out), nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "decryptAesCbc",
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			input := listToBytes(args[0].(*values.List))
			key := listToBytes(args[1].(*values.List))
			iv := listToBytes(args[2].(*values.List))
			block, err := aes.NewCipher(key)
			if err != nil {
				return cryptoError(fmt.Sprintf("Error occurred while AES-CBC decrypt: %s", err.Error())), nil
			}
			if len(input)%aes.BlockSize != 0 {
				return cryptoError("Error occurred while AES-CBC decrypt: input length is not a multiple of block size"), nil
			}
			out := make([]byte, len(input))
			cipher.NewCBCDecrypter(block, iv).CryptBlocks(out, input)
			unpadded, err := pkcs7Unpad(out)
			if err != nil {
				return cryptoError(fmt.Sprintf("Error occurred while AES-CBC decrypt: %s", err.Error())), nil
			}
			return bytesToList(types.byteArrTy, ctx, unpadded), nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "encryptAesEcb",
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			input := listToBytes(args[0].(*values.List))
			key := listToBytes(args[1].(*values.List))
			block, err := aes.NewCipher(key)
			if err != nil {
				return cryptoError(fmt.Sprintf("Error occurred while AES-ECB encrypt: %s", err.Error())), nil
			}
			padded := pkcs7Pad(input, aes.BlockSize)
			out := make([]byte, len(padded))
			for i := 0; i < len(padded); i += aes.BlockSize {
				block.Encrypt(out[i:i+aes.BlockSize], padded[i:i+aes.BlockSize])
			}
			return bytesToList(types.byteArrTy, ctx, out), nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "decryptAesEcb",
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			input := listToBytes(args[0].(*values.List))
			key := listToBytes(args[1].(*values.List))
			block, err := aes.NewCipher(key)
			if err != nil {
				return cryptoError(fmt.Sprintf("Error occurred while AES-ECB decrypt: %s", err.Error())), nil
			}
			if len(input)%aes.BlockSize != 0 {
				return cryptoError("Error occurred while AES-ECB decrypt: input length is not a multiple of block size"), nil
			}
			out := make([]byte, len(input))
			for i := 0; i < len(input); i += aes.BlockSize {
				block.Decrypt(out[i:i+aes.BlockSize], input[i:i+aes.BlockSize])
			}
			unpadded, err := pkcs7Unpad(out)
			if err != nil {
				return cryptoError(fmt.Sprintf("Error occurred while AES-ECB decrypt: %s", err.Error())), nil
			}
			return bytesToList(types.byteArrTy, ctx, unpadded), nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "encryptAesGcm",
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			input := listToBytes(args[0].(*values.List))
			key := listToBytes(args[1].(*values.List))
			iv := listToBytes(args[2].(*values.List))
			tagSize := args[4].(int64)
			if !validGcmTagSizes[tagSize] {
				return cryptoError(fmt.Sprintf("Error occurred while AES-GCM encrypt: invalid tag size %d bits", tagSize)), nil
			}
			block, err := aes.NewCipher(key)
			if err != nil {
				return cryptoError(fmt.Sprintf("Error occurred while AES-GCM encrypt: %s", err.Error())), nil
			}
			gcm, err := newGCM(block, len(iv), int(tagSize/8))
			if err != nil {
				return cryptoError(fmt.Sprintf("Error occurred while AES-GCM encrypt: %s", err.Error())), nil
			}
			out := gcm.Seal(nil, iv, input, nil)
			return bytesToList(types.byteArrTy, ctx, out), nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "decryptAesGcm",
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			input := listToBytes(args[0].(*values.List))
			key := listToBytes(args[1].(*values.List))
			iv := listToBytes(args[2].(*values.List))
			tagSize := args[4].(int64)
			if !validGcmTagSizes[tagSize] {
				return cryptoError(fmt.Sprintf("Error occurred while AES-GCM decrypt: invalid tag size %d bits", tagSize)), nil
			}
			block, err := aes.NewCipher(key)
			if err != nil {
				return cryptoError(fmt.Sprintf("Error occurred while AES-GCM decrypt: %s", err.Error())), nil
			}
			gcm, err := newGCM(block, len(iv), int(tagSize/8))
			if err != nil {
				return cryptoError(fmt.Sprintf("Error occurred while AES-GCM decrypt: %s", err.Error())), nil
			}
			out, err := gcm.Open(nil, iv, input, nil)
			if err != nil {
				return cryptoError(fmt.Sprintf("Error occurred while AES-GCM decrypt: %s", err.Error())), nil
			}
			return bytesToList(types.byteArrTy, ctx, out), nil
		})
}

// newGCM creates a GCM AEAD that accepts the given nonce and tag sizes.
// Go's standard library exposes NewGCMWithNonceSize (fixed 16-byte tag) and
// NewGCMWithTagSize (fixed 12-byte nonce) but not both together.
func newGCM(block cipher.Block, nonceSize, tagSize int) (cipher.AEAD, error) {
	const standardNonce = 12
	const standardTag = 16
	if tagSize == standardTag {
		return cipher.NewGCMWithNonceSize(block, nonceSize)
	}
	if nonceSize == standardNonce {
		return cipher.NewGCMWithTagSize(block, tagSize)
	}
	return nil, fmt.Errorf("non-standard nonce (%d bytes) and tag (%d bytes) combination not supported", nonceSize, tagSize)
}

func pkcs7Pad(data []byte, blockSize int) []byte {
	padding := blockSize - len(data)%blockSize
	return append(data, bytes.Repeat([]byte{byte(padding)}, padding)...)
}

func pkcs7Unpad(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty input")
	}
	padding := int(data[len(data)-1])
	if padding == 0 || padding > len(data) {
		return nil, fmt.Errorf("invalid padding")
	}
	return data[:len(data)-padding], nil
}
