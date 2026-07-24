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
	"crypto/rand"
	"crypto/sha1" //nolint:gosec
	"crypto/sha256"
	"crypto/sha512"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"hash"
	"regexp"
	"strconv"
	"strings"

	"ballerina/runtime"
	"ballerina/runtime/extern"
	"ballerina/values"
	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/crypto/pbkdf2"
)

// pbkdf2HashPattern matches: $pbkdf2-<alg>$i=<iter>$<b64salt>$<b64hash>
var pbkdf2HashPattern = regexp.MustCompile(`^\$pbkdf2-(\w+)\$i=(\d+)\$([A-Za-z0-9+/=]+)\$([A-Za-z0-9+/=]+)$`)

// argon2Pattern matches: $argon2id$v=19$m=<mem>,t=<iter>,p=<par>$<b64salt>$<b64hash>
// Also matches without v= for compatibility.
var argon2Pattern = regexp.MustCompile(`^\$argon2id(?:\$v=\d+)?\$m=(\d+),t=(\d+),p=(\d+)\$([A-Za-z0-9+/]+={0,2})\$([A-Za-z0-9+/]+={0,2})$`)

func registerPasswordFunctions(rt *runtime.Runtime, _ cryptoTypes) {
	runtime.RegisterExternFunction(rt, orgName, moduleName, "hashBcrypt",
		func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
			password, _ := args[0].(string)
			workFactor := int(args[1].(int64))
			hash, err := bcrypt.GenerateFromPassword([]byte(password), workFactor)
			if err != nil {
				return cryptoError(fmt.Sprintf("Error occurred while hashing password: %s", err.Error())), nil
			}
			return string(hash), nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "verifyBcrypt",
		func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
			password, _ := args[0].(string)
			hashed, _ := args[1].(string)
			err := bcrypt.CompareHashAndPassword([]byte(hashed), []byte(password))
			if err != nil {
				if err == bcrypt.ErrMismatchedHashAndPassword {
					return false, nil
				}
				return cryptoError(fmt.Sprintf("Error occurred while verifying password: %s", err.Error())), nil
			}
			return true, nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "hashArgon2",
		func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
			password, _ := args[0].(string)
			iterations := uint32(args[1].(int64))
			memory := uint32(args[2].(int64))
			parallelism := uint8(args[3].(int64))
			const keyLen = 32
			salt := make([]byte, 16)
			if _, err := rand.Read(salt); err != nil {
				return cryptoError(fmt.Sprintf("Error occurred while hashing password: %s", err.Error())), nil
			}
			key := argon2.IDKey([]byte(password), salt, iterations, memory, parallelism, keyLen)
			saltB64 := base64.RawStdEncoding.EncodeToString(salt)
			hashB64 := base64.RawStdEncoding.EncodeToString(key)
			encoded := fmt.Sprintf("$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s",
				memory, iterations, parallelism, saltB64, hashB64)
			return encoded, nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "verifyArgon2",
		func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
			password, _ := args[0].(string)
			hashed, _ := args[1].(string)
			m := argon2Pattern.FindStringSubmatch(hashed)
			if m == nil {
				return cryptoError("Error occurred while verifying Argon2 password: invalid hash format"), nil
			}
			memory, _ := strconv.ParseUint(m[1], 10, 32)
			iterations, _ := strconv.ParseUint(m[2], 10, 32)
			parallelism, _ := strconv.ParseUint(m[3], 10, 8)
			salt, err := base64.RawStdEncoding.DecodeString(m[4])
			if err != nil {
				return cryptoError(fmt.Sprintf("Error occurred while verifying Argon2 password: %s", err.Error())), nil
			}
			storedHash, err := base64.RawStdEncoding.DecodeString(m[5])
			if err != nil {
				return cryptoError(fmt.Sprintf("Error occurred while verifying Argon2 password: %s", err.Error())), nil
			}
			derived := argon2.IDKey([]byte(password), salt, uint32(iterations), uint32(memory), uint8(parallelism), uint32(len(storedHash)))
			return subtle.ConstantTimeCompare(derived, storedHash) == 1, nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "hashPbkdf2",
		func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
			password, _ := args[0].(string)
			iterations := int(args[1].(int64))
			alg, _ := args[2].(string)
			newHash, keyLen, algName, err := pbkdf2Params(alg)
			if err != nil {
				return cryptoError(err.Error()), nil
			}
			salt := make([]byte, 16)
			if _, err := rand.Read(salt); err != nil {
				return cryptoError(fmt.Sprintf("Error occurred while hashing password: %s", err.Error())), nil
			}
			key := pbkdf2.Key([]byte(password), salt, iterations, keyLen, newHash)
			saltB64 := base64.StdEncoding.EncodeToString(salt)
			hashB64 := base64.StdEncoding.EncodeToString(key)
			return fmt.Sprintf("$pbkdf2-%s$i=%d$%s$%s", algName, iterations, saltB64, hashB64), nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "verifyPbkdf2",
		func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
			password, _ := args[0].(string)
			hashed, _ := args[1].(string)
			m := pbkdf2HashPattern.FindStringSubmatch(hashed)
			if m == nil {
				return cryptoError("Error occurred while verifying PBKDF2 password: invalid hash format"), nil
			}
			algStr := m[1]
			iterations, _ := strconv.Atoi(m[2])
			salt, err := base64.StdEncoding.DecodeString(m[3])
			if err != nil {
				return cryptoError(fmt.Sprintf("Error occurred while verifying PBKDF2 password: %s", err.Error())), nil
			}
			storedHash, err := base64.StdEncoding.DecodeString(m[4])
			if err != nil {
				return cryptoError(fmt.Sprintf("Error occurred while verifying PBKDF2 password: %s", err.Error())), nil
			}
			newHash, _, _, err := pbkdf2Params(algStr)
			if err != nil {
				return cryptoError(err.Error()), nil
			}
			derived := pbkdf2.Key([]byte(password), salt, iterations, len(storedHash), newHash)
			return subtle.ConstantTimeCompare(derived, storedHash) == 1, nil
		})
}

func pbkdf2Params(alg string) (func() hash.Hash, int, string, error) {
	switch strings.ToUpper(alg) {
	case "SHA1":
		return sha1.New, 20, "SHA1", nil //nolint:gosec
	case "SHA256":
		return sha256.New, 32, "SHA256", nil
	case "SHA512":
		return sha512.New, 64, "SHA512", nil
	default:
		return nil, 0, "", fmt.Errorf("error occurred while hashing password: unsupported algorithm %q", alg)
	}
}
