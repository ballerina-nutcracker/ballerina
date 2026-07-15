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
	"crypto"
	"crypto/ecdsa"
	"crypto/md5" //nolint:gosec
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1" //nolint:gosec
	"crypto/sha256"
	"crypto/sha512"
	"fmt"
	"hash"

	"ballerina-lang-go/runtime"
	"ballerina-lang-go/runtime/extern"
	"ballerina-lang-go/values"
)

func registerRsaFunctions(rt *runtime.Runtime, types cryptoTypes) {
	runtime.RegisterExternFunction(rt, orgName, moduleName, "encryptRsaEcb",
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			input := args[0].(*values.List).ToByteSlice()
			keyMap, _ := args[1].(*values.Map)
			padding, _ := args[2].(string)
			oaepHash := oaepHashForPadding(padding)
			switch k := keyDataOf(keyMap).(type) {
			case *rsa.PublicKey:
				var out []byte
				var err error
				if oaepHash != nil {
					out, err = rsa.EncryptOAEP(oaepHash, rand.Reader, k, input, nil)
				} else {
					out, err = rsa.EncryptPKCS1v15(rand.Reader, k, input) //nolint:gosec,staticcheck
				}
				if err != nil {
					return cryptoError(fmt.Sprintf("Error occurred while RSA encrypt: %s", err.Error())), nil
				}
				return values.ByteSliceToList(types.byteArrTy, ctx.TypeCtx, out), nil
			case *rsa.PrivateKey:
				pub := &k.PublicKey
				var out []byte
				var err error
				if oaepHash != nil {
					out, err = rsa.EncryptOAEP(oaepHash, rand.Reader, pub, input, nil)
				} else {
					out, err = rsa.EncryptPKCS1v15(rand.Reader, pub, input) //nolint:gosec,staticcheck
				}
				if err != nil {
					return cryptoError(fmt.Sprintf("Error occurred while RSA encrypt: %s", err.Error())), nil
				}
				return values.ByteSliceToList(types.byteArrTy, ctx.TypeCtx, out), nil
			default:
				return cryptoError("Uninitialized public key"), nil
			}
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "decryptRsaEcb",
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			input := args[0].(*values.List).ToByteSlice()
			keyMap, _ := args[1].(*values.Map)
			padding, _ := args[2].(string)
			oaepHash := oaepHashForPadding(padding)
			switch k := keyDataOf(keyMap).(type) {
			case *rsa.PrivateKey:
				var out []byte
				var err error
				if oaepHash != nil {
					out, err = rsa.DecryptOAEP(oaepHash, rand.Reader, k, input, nil)
				} else {
					out, err = rsa.DecryptPKCS1v15(rand.Reader, k, input) //nolint:gosec,staticcheck
				}
				if err != nil {
					return cryptoError(fmt.Sprintf("Error occurred while RSA decrypt: %s", err.Error())), nil
				}
				return values.ByteSliceToList(types.byteArrTy, ctx.TypeCtx, out), nil
			default:
				return cryptoError("Uninitialized private key"), nil
			}
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "signRsaMd5",
		rsaSignFunc(crypto.MD5, func() hash.Hash { return md5.New() }, types)) //nolint:gosec

	runtime.RegisterExternFunction(rt, orgName, moduleName, "signRsaSha1",
		rsaSignFunc(crypto.SHA1, func() hash.Hash { return sha1.New() }, types)) //nolint:gosec

	runtime.RegisterExternFunction(rt, orgName, moduleName, "signRsaSha256",
		rsaSignFunc(crypto.SHA256, func() hash.Hash { return sha256.New() }, types))

	runtime.RegisterExternFunction(rt, orgName, moduleName, "signRsaSha384",
		rsaSignFunc(crypto.SHA384, func() hash.Hash { return sha512.New384() }, types))

	runtime.RegisterExternFunction(rt, orgName, moduleName, "signRsaSha512",
		rsaSignFunc(crypto.SHA512, func() hash.Hash { return sha512.New() }, types))

	runtime.RegisterExternFunction(rt, orgName, moduleName, "signRsaSsaPss256",
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			input := args[0].(*values.List).ToByteSlice()
			keyMap, _ := args[1].(*values.Map)
			privKey, ok := keyDataOf(keyMap).(*rsa.PrivateKey)
			if !ok {
				return cryptoError("Uninitialized private key: not an RSA key"), nil
			}
			h := sha256.New()
			h.Write(input)
			sig, err := rsa.SignPSS(rand.Reader, privKey, crypto.SHA256, h.Sum(nil),
				&rsa.PSSOptions{SaltLength: rsa.PSSSaltLengthEqualsHash})
			if err != nil {
				return cryptoError(fmt.Sprintf("Error occurred while calculating signature: %s", err.Error())), nil
			}
			return values.ByteSliceToList(types.byteArrTy, ctx.TypeCtx, sig), nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "signSha256withEcdsa",
		ecdsaSignFunc(func() hash.Hash { return sha256.New() }, types))

	runtime.RegisterExternFunction(rt, orgName, moduleName, "signSha384withEcdsa",
		ecdsaSignFunc(func() hash.Hash { return sha512.New384() }, types))

	runtime.RegisterExternFunction(rt, orgName, moduleName, "verifyRsaMd5Signature",
		rsaVerifyFunc(crypto.MD5, func() hash.Hash { return md5.New() })) //nolint:gosec

	runtime.RegisterExternFunction(rt, orgName, moduleName, "verifyRsaSha1Signature",
		rsaVerifyFunc(crypto.SHA1, func() hash.Hash { return sha1.New() })) //nolint:gosec

	runtime.RegisterExternFunction(rt, orgName, moduleName, "verifyRsaSha256Signature",
		rsaVerifyFunc(crypto.SHA256, func() hash.Hash { return sha256.New() }))

	runtime.RegisterExternFunction(rt, orgName, moduleName, "verifyRsaSha384Signature",
		rsaVerifyFunc(crypto.SHA384, func() hash.Hash { return sha512.New384() }))

	runtime.RegisterExternFunction(rt, orgName, moduleName, "verifyRsaSha512Signature",
		rsaVerifyFunc(crypto.SHA512, func() hash.Hash { return sha512.New() }))

	runtime.RegisterExternFunction(rt, orgName, moduleName, "verifyRsaSsaPss256Signature",
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			data := args[0].(*values.List).ToByteSlice()
			sig := args[1].(*values.List).ToByteSlice()
			keyMap, _ := args[2].(*values.Map)
			pubKey, ok := keyDataOf(keyMap).(*rsa.PublicKey)
			if !ok {
				return cryptoError("Uninitialized public key: not an RSA key"), nil
			}
			h := sha256.New()
			h.Write(data)
			err := rsa.VerifyPSS(pubKey, crypto.SHA256, h.Sum(nil), sig,
				&rsa.PSSOptions{SaltLength: rsa.PSSSaltLengthEqualsHash})
			if err != nil {
				return false, nil
			}
			return true, nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "verifySha256withEcdsaSignature",
		ecdsaVerifyFunc(func() hash.Hash { return sha256.New() }))

	runtime.RegisterExternFunction(rt, orgName, moduleName, "verifySha384withEcdsaSignature",
		ecdsaVerifyFunc(func() hash.Hash { return sha512.New384() }))
}

func rsaSignFunc(hashID crypto.Hash, newHash func() hash.Hash, types cryptoTypes) extern.NativeFunc {
	return func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
		input := args[0].(*values.List).ToByteSlice()
		keyMap, _ := args[1].(*values.Map)
		privKey, ok := keyDataOf(keyMap).(*rsa.PrivateKey)
		if !ok {
			return cryptoError("Uninitialized private key: not an RSA key"), nil
		}
		h := newHash()
		h.Write(input)
		sig, err := rsa.SignPKCS1v15(rand.Reader, privKey, hashID, h.Sum(nil))
		if err != nil {
			return cryptoError(fmt.Sprintf("Error occurred while calculating signature: %s", err.Error())), nil
		}
		return values.ByteSliceToList(types.byteArrTy, ctx.TypeCtx, sig), nil
	}
}

func rsaVerifyFunc(hashID crypto.Hash, newHash func() hash.Hash) extern.NativeFunc {
	return func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
		data := args[0].(*values.List).ToByteSlice()
		sig := args[1].(*values.List).ToByteSlice()
		keyMap, _ := args[2].(*values.Map)
		pubKey, ok := keyDataOf(keyMap).(*rsa.PublicKey)
		if !ok {
			return cryptoError("Uninitialized public key: not an RSA key"), nil
		}
		h := newHash()
		h.Write(data)
		err := rsa.VerifyPKCS1v15(pubKey, hashID, h.Sum(nil), sig)
		if err != nil {
			return false, nil
		}
		return true, nil
	}
}

func ecdsaSignFunc(newHash func() hash.Hash, types cryptoTypes) extern.NativeFunc {
	return func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
		input := args[0].(*values.List).ToByteSlice()
		keyMap, _ := args[1].(*values.Map)
		privKey, ok := keyDataOf(keyMap).(*ecdsa.PrivateKey)
		if !ok {
			return cryptoError("Uninitialized private key: not an EC key"), nil
		}
		h := newHash()
		h.Write(input)
		sig, err := ecdsa.SignASN1(rand.Reader, privKey, h.Sum(nil))
		if err != nil {
			return cryptoError(fmt.Sprintf("Error occurred while calculating signature: %s", err.Error())), nil
		}
		return values.ByteSliceToList(types.byteArrTy, ctx.TypeCtx, sig), nil
	}
}

func ecdsaVerifyFunc(newHash func() hash.Hash) extern.NativeFunc {
	return func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
		data := args[0].(*values.List).ToByteSlice()
		sig := args[1].(*values.List).ToByteSlice()
		keyMap, _ := args[2].(*values.Map)
		pubKey, ok := keyDataOf(keyMap).(*ecdsa.PublicKey)
		if !ok {
			return cryptoError("Uninitialized public key: not an EC key"), nil
		}
		h := newHash()
		h.Write(data)
		return ecdsa.VerifyASN1(pubKey, h.Sum(nil), sig), nil
	}
}

// oaepHashForPadding returns the hash to use with OAEP, or nil for PKCS1.
func oaepHashForPadding(padding string) hash.Hash {
	switch padding {
	case "OAEPwithMD5andMGF1":
		return md5.New() //nolint:gosec
	case "OAEPWithSHA1AndMGF1":
		return sha1.New() //nolint:gosec
	case "OAEPWithSHA256AndMGF1":
		return sha256.New()
	case "OAEPwithSHA384andMGF1":
		return sha512.New384()
	case "OAEPwithSHA512andMGF1":
		return sha512.New()
	default:
		return nil
	}
}
