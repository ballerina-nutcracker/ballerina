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
	"crypto/des"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"fmt"
	"math/big"

	"golang.org/x/crypto/pbkdf2"
)

var (
	oidPBEWithSHAAnd3KeyTripleDESCBC = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 12, 1, 3}
	oidPBEWithSHAAnd2KeyTripleDESCBC = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 12, 1, 4}
	oidPBEWithSHAAnd40BitRC2CBC      = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 12, 1, 6}
	oidPBES2                         = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 5, 13}
	oidPBKDF2                        = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 5, 12}
	oidHMACWithSHA1                  = asn1.ObjectIdentifier{1, 2, 840, 113549, 2, 7}
	oidHMACWithSHA256                = asn1.ObjectIdentifier{1, 2, 840, 113549, 2, 9}
	oidAES128CBC                     = asn1.ObjectIdentifier{2, 16, 840, 1, 101, 3, 4, 1, 2}
	oidAES256CBC                     = asn1.ObjectIdentifier{2, 16, 840, 1, 101, 3, 4, 1, 42}
	oidDESEDE3CBC                    = asn1.ObjectIdentifier{1, 2, 840, 113549, 3, 7}
)

var bigOne = big.NewInt(1)

type encryptedPrivateKeyInfo struct {
	AlgID pkix.AlgorithmIdentifier
	Data  []byte
}

type pbes2Params struct {
	KDF    pkix.AlgorithmIdentifier
	Cipher pkix.AlgorithmIdentifier
}

type pkcs8PBKDF2Params struct {
	Salt       asn1.RawValue
	Iterations int
	KeyLength  int                      `asn1:"optional"`
	PRF        pkix.AlgorithmIdentifier `asn1:"optional"`
}

type pbeParams struct {
	Salt       []byte
	Iterations int
}

// decryptEncryptedPKCS8 decrypts an EncryptedPrivateKeyInfo DER block and returns
// the decrypted PrivateKeyInfo DER, ready for x509.ParsePKCS8PrivateKey.
func decryptEncryptedPKCS8(der []byte, password string) (any, error) {
	var info encryptedPrivateKeyInfo
	if _, err := asn1.Unmarshal(der, &info); err != nil {
		return nil, fmt.Errorf("failed to parse EncryptedPrivateKeyInfo: %w", err)
	}

	algOID := info.AlgID.Algorithm
	switch {
	case algOID.Equal(oidPBEWithSHAAnd3KeyTripleDESCBC):
		return decryptPKCS12PBE(info, password, 24)
	case algOID.Equal(oidPBEWithSHAAnd2KeyTripleDESCBC):
		return decryptPKCS12PBE(info, password, 16)
	case algOID.Equal(oidPBEWithSHAAnd40BitRC2CBC):
		return decryptPKCS12PBERC2(info, password)
	case algOID.Equal(oidPBES2):
		return decryptPBES2(info, password)
	default:
		return nil, fmt.Errorf("unsupported PKCS#8 encryption algorithm: %s", algOID)
	}
}

// decryptPKCS12PBE handles PBEWithSHAAnd3KeyTripleDES-CBC and PBEWithSHAAnd2KeyTripleDES-CBC.
// keySize is 24 for 3-key 3DES and 16 for 2-key 3DES (where K3 = K1).
func decryptPKCS12PBE(info encryptedPrivateKeyInfo, password string, keySize int) (any, error) {
	plain, err := pkcs12PBEDecrypt3DES(info.AlgID, info.Data, password, keySize)
	if err != nil {
		return nil, fmt.Errorf("PKCS#12 PBE decryption failed (wrong password?): %w", err)
	}
	return x509.ParsePKCS8PrivateKey(plain)
}

// decryptPKCS12PBERC2 handles PBEWithSHAAnd40BitRC2-CBC (PKCS#12 PBE with RC2-40).
func decryptPKCS12PBERC2(info encryptedPrivateKeyInfo, password string) (any, error) {
	plain, err := pkcs12PBEDecryptRC2(info.AlgID, info.Data, password)
	if err != nil {
		return nil, err
	}
	return x509.ParsePKCS8PrivateKey(plain)
}

// pkcs12PBEDecryptRC2 decrypts data encrypted with pbeWithSHAAnd40BitRC2-CBC.
// Exported at package level so pkcs12_trust.go can also use it.
func pkcs12PBEDecryptRC2(algID pkix.AlgorithmIdentifier, data []byte, password string) ([]byte, error) {
	var params pbeParams
	if _, err := asn1.Unmarshal(algID.Parameters.FullBytes, &params); err != nil {
		return nil, fmt.Errorf("failed to parse RC2-40 PBE params: %w", err)
	}
	pwdBMP := bmpEncodePassword(password)
	const u, v = 20, 64
	key := pkcs12KDF(params.Salt, pwdBMP, params.Iterations, 1, 5, u, v)
	iv := pkcs12KDF(params.Salt, pwdBMP, params.Iterations, 2, 8, u, v)
	block, err := rc2New(key, 40)
	if err != nil {
		return nil, fmt.Errorf("failed to create RC2 cipher: %w", err)
	}
	return cbcDecryptAndUnpad(data, block, iv)
}

// pkcs12PBEDecrypt3DES decrypts data encrypted with pbeWithSHAAnd3KeyTripleDES-CBC
// or pbeWithSHAAnd2KeyTripleDES-CBC. keySize is 24 or 16 respectively.
func pkcs12PBEDecrypt3DES(algID pkix.AlgorithmIdentifier, data []byte, password string, keySize int) ([]byte, error) {
	var params pbeParams
	if _, err := asn1.Unmarshal(algID.Parameters.FullBytes, &params); err != nil {
		return nil, fmt.Errorf("failed to parse 3DES PBE params: %w", err)
	}
	pwdBMP := bmpEncodePassword(password)
	const u, v = 20, 64
	key := pkcs12KDF(params.Salt, pwdBMP, params.Iterations, 1, keySize, u, v)
	iv := pkcs12KDF(params.Salt, pwdBMP, params.Iterations, 2, 8, u, v)
	if keySize == 16 {
		key = append(key, key[:8]...)
	}
	block, err := des.NewTripleDESCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create 3DES cipher: %w", err)
	}
	return cbcDecryptAndUnpad(data, block, iv)
}

// decryptPBES2 handles PBES2 (PBKDF2 + AES-CBC / 3DES-CBC).
func decryptPBES2(info encryptedPrivateKeyInfo, password string) (any, error) {
	var params pbes2Params
	if _, err := asn1.Unmarshal(info.AlgID.Parameters.FullBytes, &params); err != nil {
		return nil, fmt.Errorf("failed to parse PBES2 params: %w", err)
	}

	if !params.KDF.Algorithm.Equal(oidPBKDF2) {
		return nil, fmt.Errorf("unsupported PBES2 KDF: %s", params.KDF.Algorithm)
	}

	var kdfParams pkcs8PBKDF2Params
	if _, err := asn1.Unmarshal(params.KDF.Parameters.FullBytes, &kdfParams); err != nil {
		return nil, fmt.Errorf("failed to parse PBKDF2 params: %w", err)
	}

	var salt []byte
	if _, err := asn1.Unmarshal(kdfParams.Salt.FullBytes, &salt); err != nil {
		return nil, fmt.Errorf("failed to parse PBKDF2 salt: %w", err)
	}

	// Determine PRF hash and derive key.
	hashFunc := sha1.New
	keySize := 16
	prf := kdfParams.PRF.Algorithm
	switch {
	case prf.Equal(oidHMACWithSHA256):
		hashFunc = sha256.New
		keySize = 32
	case len(prf) != 0 && !prf.Equal(oidHMACWithSHA1):
		return nil, fmt.Errorf("unsupported PBKDF2 PRF: %s", prf)
	}

	// Key size can be overridden by the KeyLength field.
	if kdfParams.KeyLength > 0 {
		keySize = kdfParams.KeyLength
	}

	key := pbkdf2.Key([]byte(password), salt, kdfParams.Iterations, keySize, hashFunc)

	// Parse cipher IV from encryptionScheme parameters.
	encOID := params.Cipher.Algorithm
	var iv []byte
	if _, err := asn1.Unmarshal(params.Cipher.Parameters.FullBytes, &iv); err != nil {
		return nil, fmt.Errorf("failed to parse cipher IV: %w", err)
	}

	var block cipher.Block
	var blockErr error
	switch {
	case encOID.Equal(oidAES128CBC):
		block, blockErr = aes.NewCipher(key[:16])
	case encOID.Equal(oidAES256CBC):
		block, blockErr = aes.NewCipher(key[:32])
	case encOID.Equal(oidDESEDE3CBC):
		block, blockErr = des.NewTripleDESCipher(key)
	default:
		return nil, fmt.Errorf("unsupported PBES2 cipher: %s", encOID)
	}
	if blockErr != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", blockErr)
	}

	plain, err := cbcDecryptAndUnpad(info.Data, block, iv)
	if err != nil {
		return nil, fmt.Errorf("PBES2 decryption failed (wrong password?): %w", err)
	}

	return x509.ParsePKCS8PrivateKey(plain)
}

// cbcDecryptAndUnpad decrypts data using CBC mode and removes PKCS#7 padding.
func cbcDecryptAndUnpad(data []byte, block cipher.Block, iv []byte) ([]byte, error) {
	blockSize := block.BlockSize()
	if len(data) == 0 || len(data)%blockSize != 0 {
		return nil, fmt.Errorf("encrypted data length not a multiple of block size")
	}
	decrypted := make([]byte, len(data))
	cipher.NewCBCDecrypter(block, iv).CryptBlocks(decrypted, data)

	padLen := int(decrypted[len(decrypted)-1])
	if padLen == 0 || padLen > blockSize || padLen > len(decrypted) {
		return nil, fmt.Errorf("invalid PKCS#7 padding")
	}
	return decrypted[:len(decrypted)-padLen], nil
}

// bmpEncodePassword encodes a UTF-8 password as UTF-16 BE with a null terminator,
// as required by the PKCS#12 PBE key derivation function.
func bmpEncodePassword(s string) []byte {
	runes := []rune(s)
	out := make([]byte, (len(runes)+1)*2)
	for i, r := range runes {
		out[i*2] = byte(r >> 8)
		out[i*2+1] = byte(r)
	}
	return out
}

// pkcs12KDF implements RFC 7292 Appendix B.2 key derivation.
// id=1 for key material, id=2 for IV. u and v are the hash's output/block sizes in bytes.
func pkcs12KDF(salt, password []byte, iterations int, id byte, size, u, v int) []byte {
	D := bytes.Repeat([]byte{id}, v)
	S := pkcs12FillRepeats(salt, v)
	P := pkcs12FillRepeats(password, v)
	I := append(S, P...)

	c := (size + u - 1) / u
	A := make([]byte, c*u)

	for i := 0; i < c; i++ {
		Ai := sha1.Sum(append(D, I...))
		for j := 1; j < iterations; j++ {
			Ai = sha1.Sum(Ai[:])
		}
		copy(A[i*u:], Ai[:])

		if i < c-1 {
			// Build B: repeat Ai to fill v bytes.
			var B []byte
			for len(B) < v {
				B = append(B, Ai[:]...)
			}
			B = B[:v]

			Bbi := new(big.Int).SetBytes(B)
			Ij := new(big.Int)
			for j := 0; j < len(I)/v; j++ {
				Ij.SetBytes(I[j*v : (j+1)*v])
				Ij.Add(Ij, Bbi)
				Ij.Add(Ij, bigOne)
				Ijb := Ij.Bytes()
				if len(Ijb) > v {
					Ijb = Ijb[len(Ijb)-v:]
				}
				if len(Ijb) < v {
					tmp := make([]byte, v)
					copy(tmp[v-len(Ijb):], Ijb)
					Ijb = tmp
				}
				copy(I[j*v:(j+1)*v], Ijb)
			}
		}
	}
	return A[:size]
}

// pkcs12FillRepeats returns v*ceil(len(pattern)/v) bytes of pattern repeated.
func pkcs12FillRepeats(pattern []byte, v int) []byte {
	if len(pattern) == 0 {
		return nil
	}
	outputLen := v * ((len(pattern) + v - 1) / v)
	return bytes.Repeat(pattern, (outputLen+len(pattern)-1)/len(pattern))[:outputLen]
}
