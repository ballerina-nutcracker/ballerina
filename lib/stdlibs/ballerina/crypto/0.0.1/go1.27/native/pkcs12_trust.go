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
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"fmt"
	"strings"
	"unicode/utf16"
)

var (
	oidDataContentType    = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 7, 1}
	oidEncDataContentType = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 7, 6}
	oidCertBagType        = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 12, 10, 1, 3}
	oidFriendlyName       = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 9, 20}
	oidX509Certificate    = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 9, 22, 1}
)

type p12PFX struct {
	Version  int
	AuthSafe p12ContentInfo
	MacData  asn1.RawValue `asn1:"optional"`
}

type p12ContentInfo struct {
	ContentType asn1.ObjectIdentifier
	Content     asn1.RawValue `asn1:"tag:0,explicit,optional"`
}

type p12EncryptedData struct {
	Version              int
	EncryptedContentInfo p12EncryptedContentInfo
}

type p12EncryptedContentInfo struct {
	ContentType                asn1.ObjectIdentifier
	ContentEncryptionAlgorithm pkix.AlgorithmIdentifier
	EncryptedContent           []byte `asn1:"tag:0,optional"`
}

type p12SafeBag struct {
	BagID         asn1.ObjectIdentifier
	BagValue      asn1.RawValue  `asn1:"tag:0,explicit"`
	BagAttributes []p12Attribute `asn1:"set,optional"`
}

type p12Attribute struct {
	AttrType   asn1.ObjectIdentifier
	AttrValues asn1.RawValue `asn1:"set"`
}

type p12CertBag struct {
	CertID    asn1.ObjectIdentifier
	CertValue asn1.RawValue `asn1:"tag:0,explicit"`
}

// decodeCertFromTrustStore parses a PKCS#12 trust store and returns the certificate
// matching alias (case-insensitive). Falls back to the first certificate if not found.
func decodeCertFromTrustStore(p12Data []byte, password, alias string) (*x509.Certificate, error) {
	var pfx p12PFX
	if _, err := asn1.Unmarshal(p12Data, &pfx); err != nil {
		return nil, fmt.Errorf("failed to parse PFX structure: %w", err)
	}

	if !pfx.AuthSafe.ContentType.Equal(oidDataContentType) {
		return nil, fmt.Errorf("unexpected AuthenticatedSafe content type: %s", pfx.AuthSafe.ContentType)
	}

	// AuthSafe Content is an OCTET STRING wrapping a SEQUENCE OF ContentInfo.
	var authSafeData []byte
	if _, err := asn1.Unmarshal(pfx.AuthSafe.Content.Bytes, &authSafeData); err != nil {
		return nil, fmt.Errorf("failed to parse AuthenticatedSafe data: %w", err)
	}

	var contentInfos []p12ContentInfo
	if _, err := asn1.Unmarshal(authSafeData, &contentInfos); err != nil {
		return nil, fmt.Errorf("failed to parse ContentInfo sequence: %w", err)
	}

	var firstCert *x509.Certificate

	for _, ci := range contentInfos {
		var safeBagDER []byte

		switch {
		case ci.ContentType.Equal(oidDataContentType):
			// Unencrypted: Content is OCTET STRING wrapping SafeContents.
			if _, err := asn1.Unmarshal(ci.Content.Bytes, &safeBagDER); err != nil {
				continue
			}

		case ci.ContentType.Equal(oidEncDataContentType):
			var encData p12EncryptedData
			if _, err := asn1.Unmarshal(ci.Content.Bytes, &encData); err != nil {
				continue
			}
			plain, err := p12DecryptContent(encData.EncryptedContentInfo.ContentEncryptionAlgorithm, encData.EncryptedContentInfo.EncryptedContent, password)
			if err != nil {
				return nil, fmt.Errorf("failed to decrypt safe contents: %w", err)
			}
			safeBagDER = plain

		default:
			continue
		}

		cert, err := p12ExtractCert(safeBagDER, alias, &firstCert)
		if err != nil {
			return nil, err
		}
		if cert != nil {
			return cert, nil
		}
	}

	if firstCert != nil {
		return firstCert, nil
	}
	return nil, fmt.Errorf("no certificate found in trust store")
}

// p12DecryptContent decrypts safe contents using the given algorithm identifier.
func p12DecryptContent(algID pkix.AlgorithmIdentifier, data []byte, password string) ([]byte, error) {
	oid := algID.Algorithm
	switch {
	case oid.Equal(oidPBEWithSHAAnd3KeyTripleDESCBC):
		return pkcs12PBEDecrypt3DES(algID, data, password, 24)
	case oid.Equal(oidPBEWithSHAAnd2KeyTripleDESCBC):
		return pkcs12PBEDecrypt3DES(algID, data, password, 16)
	case oid.Equal(oidPBEWithSHAAnd40BitRC2CBC):
		return pkcs12PBEDecryptRC2(algID, data, password)
	default:
		return nil, fmt.Errorf("unsupported PKCS#12 encryption algorithm: %s", oid)
	}
}

// p12ExtractCert parses SafeContents DER, extracts certificates, and looks for alias match.
// Returns the matching cert immediately; updates firstCert with the first cert seen.
func p12ExtractCert(safeBagDER []byte, alias string, firstCert **x509.Certificate) (*x509.Certificate, error) {
	var bags []p12SafeBag
	if _, err := asn1.Unmarshal(safeBagDER, &bags); err != nil {
		return nil, fmt.Errorf("failed to parse SafeContents: %w", err)
	}

	for _, bag := range bags {
		if !bag.BagID.Equal(oidCertBagType) {
			continue
		}
		var certBag p12CertBag
		if _, err := asn1.Unmarshal(bag.BagValue.Bytes, &certBag); err != nil {
			continue
		}
		if !certBag.CertID.Equal(oidX509Certificate) {
			continue
		}
		var certDER []byte
		if _, err := asn1.Unmarshal(certBag.CertValue.Bytes, &certDER); err != nil {
			continue
		}
		cert, err := x509.ParseCertificate(certDER)
		if err != nil {
			continue
		}
		if *firstCert == nil {
			*firstCert = cert
		}
		if alias != "" && strings.EqualFold(p12ExtractFriendlyName(bag.BagAttributes), alias) {
			return cert, nil
		}
	}
	return nil, nil
}

// p12ExtractFriendlyName finds the friendlyName attribute and decodes its BMP string value.
func p12ExtractFriendlyName(attributes []p12Attribute) string {
	for _, attr := range attributes {
		if !attr.AttrType.Equal(oidFriendlyName) {
			continue
		}
		// AttrValues is a SET; the inner element is a BMPString.
		// asn1.Unmarshal on a SET gives us the first element's raw bytes in Bytes.
		var bmpRaw asn1.RawValue
		if _, err := asn1.Unmarshal(attr.AttrValues.Bytes, &bmpRaw); err != nil {
			continue
		}
		name, err := decodeBMPString(bmpRaw.Bytes)
		if err != nil {
			continue
		}
		return name
	}
	return ""
}

// decodeBMPString decodes a UTF-16 BE byte slice into a Go string.
func decodeBMPString(b []byte) (string, error) {
	if len(b)%2 != 0 {
		return "", fmt.Errorf("BMP string length must be even")
	}
	u16 := make([]uint16, len(b)/2)
	for i := range u16 {
		u16[i] = uint16(b[i*2])<<8 | uint16(b[i*2+1])
	}
	return string(utf16.Decode(u16)), nil
}
