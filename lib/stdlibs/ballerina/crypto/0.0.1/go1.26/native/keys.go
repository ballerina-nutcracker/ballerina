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
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"math/big"
	"time"

	"ballerina-lang-go/decimal"
	"ballerina-lang-go/runtime"
	"ballerina-lang-go/runtime/extern"
	"ballerina-lang-go/semtypes"
	"ballerina-lang-go/values"
	"golang.org/x/crypto/pkcs12"
)

func registerKeyFunctions(rt *runtime.Runtime, types cryptoTypes) {
	runtime.RegisterExternFunction(rt, orgName, moduleName, "decodeRsaPrivateKeyFromKeyStore",
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			ks, _ := args[0].(*values.Map)
			alias, _ := args[1].(string)
			keyPwd, _ := args[2].(string)
			path := mapString(ks, "path")
			ksPwd := mapString(ks, "password")
			data, err := rt.Platform().FS.ReadFile(path)
			if err != nil {
				return cryptoError(fmt.Sprintf("PKCS12 KeyStore not found at: %s", path)), nil
			}
			key, _, err := pkcs12.Decode(data, ksPwd)
			if err != nil {
				// Try with key password if store password failed
				key, _, err = pkcs12.Decode(data, keyPwd)
				if err != nil {
					return cryptoError(fmt.Sprintf("Key cannot be recovered by using given key alias: %s", alias)), nil
				}
			}
			rsaKey, ok := key.(*rsa.PrivateKey)
			if !ok {
				return cryptoError("Not a valid RSA key"), nil
			}
			return buildPrivateKeyMap(types, ctx, rsaKey, "RSA"), nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "decodeEcPrivateKeyFromKeyStore",
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			ks, _ := args[0].(*values.Map)
			alias, _ := args[1].(string)
			keyPwd, _ := args[2].(string)
			path := mapString(ks, "path")
			ksPwd := mapString(ks, "password")
			data, err := rt.Platform().FS.ReadFile(path)
			if err != nil {
				return cryptoError(fmt.Sprintf("PKCS12 KeyStore not found at: %s", path)), nil
			}
			key, _, err := pkcs12.Decode(data, ksPwd)
			if err != nil {
				key, _, err = pkcs12.Decode(data, keyPwd)
				if err != nil {
					return cryptoError(fmt.Sprintf("Key cannot be recovered by using given key alias: %s", alias)), nil
				}
			}
			ecKey, ok := key.(*ecdsa.PrivateKey)
			if !ok {
				return cryptoError("Not a valid EC key"), nil
			}
			return buildPrivateKeyMap(types, ctx, ecKey, "RSA"), nil // algorithm reported as RSA for EC in jBallerina
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "decodeRsaPrivateKeyFromKeyFile",
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			keyFile, _ := args[0].(string)
			var keyPwd string
			if args[1] != nil {
				keyPwd, _ = args[1].(string)
			}
			data, err := rt.Platform().FS.ReadFile(keyFile)
			if err != nil {
				return cryptoError(fmt.Sprintf("Key file not found at: %s", keyFile)), nil
			}
			key, err := parsePrivateKeyPEM(data, keyPwd)
			if err != nil {
				return cryptoError(fmt.Sprintf("Unable to do private key operations: %s", err.Error())), nil
			}
			rsaKey, ok := key.(*rsa.PrivateKey)
			if !ok {
				return cryptoError("Not a valid RSA key"), nil
			}
			return buildPrivateKeyMap(types, ctx, rsaKey, "RSA"), nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "decodeRsaPrivateKeyFromContent",
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			content := args[0].(*values.List).ToByteSlice()
			var keyPwd string
			if args[1] != nil {
				keyPwd, _ = args[1].(string)
			}
			key, err := parsePrivateKeyPEM(content, keyPwd)
			if err != nil {
				return cryptoError(fmt.Sprintf("Failed to parse private key information from the given input: %s", err.Error())), nil
			}
			rsaKey, ok := key.(*rsa.PrivateKey)
			if !ok {
				return cryptoError("Not a valid RSA key"), nil
			}
			return buildPrivateKeyMap(types, ctx, rsaKey, "RSA"), nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "decodeEcPrivateKeyFromKeyFile",
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			keyFile, _ := args[0].(string)
			var keyPwd string
			if args[1] != nil {
				keyPwd, _ = args[1].(string)
			}
			data, err := rt.Platform().FS.ReadFile(keyFile)
			if err != nil {
				return cryptoError(fmt.Sprintf("Key file not found at: %s", keyFile)), nil
			}
			key, err := parsePrivateKeyPEM(data, keyPwd)
			if err != nil {
				return cryptoError(fmt.Sprintf("Unable to do private key operations: %s", err.Error())), nil
			}
			ecKey, ok := key.(*ecdsa.PrivateKey)
			if !ok {
				return cryptoError("Not a valid EC key"), nil
			}
			return buildPrivateKeyMap(types, ctx, ecKey, "RSA"), nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "decodeRsaPublicKeyFromTrustStore",
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			ts, _ := args[0].(*values.Map)
			alias, _ := args[1].(string)
			path := mapString(ts, "path")
			tsPwd := mapString(ts, "password")
			data, err := rt.Platform().FS.ReadFile(path)
			if err != nil {
				return cryptoError(fmt.Sprintf("PKCS12 KeyStore not found at: %s", path)), nil
			}
			cert, err := decodeCertFromTrustStore(data, tsPwd, alias)
			if err != nil {
				return cryptoError(fmt.Sprintf("Key cannot be recovered by using given key alias: %s", alias)), nil
			}
			rsaPub, ok := cert.PublicKey.(*rsa.PublicKey)
			if !ok {
				return cryptoError("Not a valid RSA key"), nil
			}
			return buildPublicKeyMap(types, ctx, rsaPub, "RSA", cert), nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "decodeEcPublicKeyFromTrustStore",
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			ts, _ := args[0].(*values.Map)
			alias, _ := args[1].(string)
			path := mapString(ts, "path")
			tsPwd := mapString(ts, "password")
			data, err := rt.Platform().FS.ReadFile(path)
			if err != nil {
				return cryptoError(fmt.Sprintf("PKCS12 KeyStore not found at: %s", path)), nil
			}
			cert, err := decodeCertFromTrustStore(data, tsPwd, alias)
			if err != nil {
				return cryptoError(fmt.Sprintf("Key cannot be recovered by using given key alias: %s", alias)), nil
			}
			ecPub, ok := cert.PublicKey.(*ecdsa.PublicKey)
			if !ok {
				return cryptoError("Not a valid EC key"), nil
			}
			return buildPublicKeyMap(types, ctx, ecPub, "RSA", cert), nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "decodeRsaPublicKeyFromCertFile",
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			certFile, _ := args[0].(string)
			data, err := rt.Platform().FS.ReadFile(certFile)
			if err != nil {
				return cryptoError(fmt.Sprintf("Key file not found at: %s", certFile)), nil
			}
			cert, err := parseCertificatePEM(data)
			if err != nil {
				return cryptoError(fmt.Sprintf("Unable to do private key operations: %s", err.Error())), nil
			}
			rsaPub, ok := cert.PublicKey.(*rsa.PublicKey)
			if !ok {
				return cryptoError("Not a valid RSA key"), nil
			}
			return buildPublicKeyMap(types, ctx, rsaPub, "RSA", cert), nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "decodeRsaPublicKeyFromContent",
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			content := args[0].(*values.List).ToByteSlice()
			cert, err := parseCertificatePEM(content)
			if err != nil {
				return cryptoError(fmt.Sprintf("Failed to parse private key information from the given input: %s", err.Error())), nil
			}
			rsaPub, ok := cert.PublicKey.(*rsa.PublicKey)
			if !ok {
				return cryptoError("Not a valid RSA key"), nil
			}
			return buildPublicKeyMap(types, ctx, rsaPub, "RSA", cert), nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "decodeEcPublicKeyFromCertFile",
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			certFile, _ := args[0].(string)
			data, err := rt.Platform().FS.ReadFile(certFile)
			if err != nil {
				return cryptoError(fmt.Sprintf("Key file not found at: %s", certFile)), nil
			}
			cert, err := parseCertificatePEM(data)
			if err != nil {
				return cryptoError(fmt.Sprintf("Unable to do private key operations: %s", err.Error())), nil
			}
			ecPub, ok := cert.PublicKey.(*ecdsa.PublicKey)
			if !ok {
				return cryptoError("Not a valid EC key"), nil
			}
			return buildPublicKeyMap(types, ctx, ecPub, "RSA", cert), nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "buildRsaPublicKey",
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			modHex, _ := args[0].(string)
			expHex, _ := args[1].(string)
			modBytes, ok1 := new(big.Int).SetString(modHex, 16)
			expBytes, ok2 := new(big.Int).SetString(expHex, 16)
			if !ok1 || !ok2 {
				return cryptoError("Invalid RSA modulus or exponent hex string"), nil
			}
			pub := &rsa.PublicKey{N: modBytes, E: int(expBytes.Int64())}
			return buildPublicKeyMap(types, ctx, pub, "RSA", nil), nil
		})
}

// parsePrivateKeyPEM parses a PEM-encoded private key (PKCS8, PKCS8-encrypted, PKCS1, EC).
func parsePrivateKeyPEM(data []byte, password string) (any, error) {
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("no PEM block found")
	}
	// PKCS#8 EncryptedPrivateKeyInfo ("BEGIN ENCRYPTED PRIVATE KEY") uses its own
	// ASN.1 wrapper and is not detected by the deprecated x509.IsEncryptedPEMBlock.
	if block.Type == "ENCRYPTED PRIVATE KEY" {
		return decryptEncryptedPKCS8(block.Bytes, password)
	}
	der := block.Bytes
	if x509.IsEncryptedPEMBlock(block) { //nolint:staticcheck
		var err error
		der, err = x509.DecryptPEMBlock(block, []byte(password)) //nolint:staticcheck
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt PEM block: %s", err.Error())
		}
	}
	if key, err := x509.ParsePKCS8PrivateKey(der); err == nil {
		return key, nil
	}
	if key, err := x509.ParsePKCS1PrivateKey(der); err == nil {
		return key, nil
	}
	if key, err := x509.ParseECPrivateKey(der); err == nil {
		return key, nil
	}
	return nil, fmt.Errorf("unsupported private key format")
}

// parseCertificatePEM parses a PEM-encoded X.509 certificate.
func parseCertificatePEM(data []byte) (*x509.Certificate, error) {
	block, _ := pem.Decode(data)
	if block == nil {
		// Try DER directly.
		cert, err := x509.ParseCertificate(data)
		if err != nil {
			return nil, fmt.Errorf("no valid certificate found")
		}
		return cert, nil
	}
	return x509.ParseCertificate(block.Bytes)
}

// buildPrivateKeyMap constructs a Ballerina PrivateKey record, associating the
// Go key with it via keyData (see keydata.go) so sign/decrypt externs can
// retrieve it.
func buildPrivateKeyMap(types cryptoTypes, ctx *extern.Context, key any, algorithm string) *values.Map {
	m := values.NewMap(types.keyMapTy, semtypes.ToMappingAtomicType(ctx.TypeCtx, types.keyMapTy), false, []values.MapEntry{
		{Key: "algorithm", Value: algorithm},
	})
	setKeyData(m, key)
	return m
}

// buildPublicKeyMap constructs a Ballerina PublicKey record with optional certificate.
func buildPublicKeyMap(types cryptoTypes, ctx *extern.Context, key any, algorithm string, cert *x509.Certificate) *values.Map {
	entries := []values.MapEntry{{Key: "algorithm", Value: algorithm}}
	if cert != nil {
		entries = append(entries, values.MapEntry{Key: "certificate", Value: buildCertMap(types, ctx, cert)})
	}
	m := values.NewMap(types.keyMapTy, semtypes.ToMappingAtomicType(ctx.TypeCtx, types.keyMapTy), false, entries)
	setKeyData(m, key)
	return m
}

// buildCertMap converts an x509.Certificate to a Ballerina Certificate record.
func buildCertMap(types cryptoTypes, ctx *extern.Context, cert *x509.Certificate) *values.Map {
	sigBytes := values.ByteSliceToList(types.byteArrTy, ctx.TypeCtx, cert.Signature)
	return values.NewMap(types.keyMapTy, semtypes.ToMappingAtomicType(ctx.TypeCtx, types.keyMapTy), false, []values.MapEntry{
		{Key: "version", Value: int64(cert.Version)},
		{Key: "serial", Value: cert.SerialNumber.Int64()},
		{Key: "issuer", Value: cert.Issuer.String()},
		{Key: "subject", Value: cert.Subject.String()},
		{Key: "notBefore", Value: goTimeToUtc(types, ctx, cert.NotBefore)},
		{Key: "notAfter", Value: goTimeToUtc(types, ctx, cert.NotAfter)},
		{Key: "signature", Value: sigBytes},
		{Key: "signingAlgorithm", Value: cert.SignatureAlgorithm.String()},
	})
}

// goTimeToUtc converts a Go time.Time to a Ballerina time:Utc tuple [int, decimal].
func goTimeToUtc(types cryptoTypes, ctx *extern.Context, t time.Time) *values.List {
	t = t.UTC()
	nanos := decimal.FromInt64(int64(t.Nanosecond()))
	nanosPerSec := decimal.FromInt64(1_000_000_000)
	frac, _ := nanos.Quo(nanosPerSec)
	return values.NewList(types.utcTy, semtypes.ToListAtomicType(ctx.TypeCtx, types.utcTy), true, nil, 2, []values.BalValue{t.Unix(), frac})
}
