// Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
//
// WSO2 LLC. licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except
// in compliance with the License.
//
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
	"encoding/pem"
	"errors"

	"github.com/ballerina-nutcracker/ballerina/platform/pal"
	"github.com/ballerina-nutcracker/ballerina/values"
	"golang.org/x/crypto/pkcs12"
)

// keyStoreToPEM reads a PKCS12 key store and returns its certificate chain and private key
// as PEM. JKS stores are not readable from Go.
func keyStoreToPEM(platform pal.Platform, path, password string) ([]byte, []byte, error) {
	data, err := readKeyStore(platform, path)
	if err != nil {
		return nil, nil, err
	}
	blocks, err := pkcs12.ToPEM(data, password)
	if err != nil {
		return nil, nil, keyStoreError(path, err.Error())
	}
	var certs, keys bytes.Buffer
	for _, block := range blocks {
		block.Headers = nil
		if block.Type == "CERTIFICATE" {
			_ = pem.Encode(&certs, block)
		} else {
			_ = pem.Encode(&keys, block)
		}
	}
	if keys.Len() == 0 {
		return nil, nil, keyStoreError(path, "No private key found in the keystore.")
	}
	return certs.Bytes(), keys.Bytes(), nil
}

// trustedCertsPEM returns the CA certificates from a PEM file path or a PKCS12 trust store.
func trustedCertsPEM(platform pal.Platform, path, password string, cert values.BalValue) ([]byte, error) {
	if _, isPath := cert.(string); isPath {
		data, err := platform.FS.ReadFile(path)
		if err != nil {
			return nil, errors.New("TrustStore PEM file does not exist: " + path)
		}
		if len(bytes.TrimSpace(data)) == 0 {
			return nil, errors.New("TrustStore PEM file is empty: " + path)
		}
		if block, _ := pem.Decode(data); block == nil {
			return nil, errors.New("PEM file contains no certificates: " + path)
		}
		return data, nil
	}
	data, err := readKeyStore(platform, path)
	if err != nil {
		return nil, err
	}
	blocks, err := pkcs12.ToPEM(data, password)
	if err != nil {
		return nil, keyStoreError(path, err.Error())
	}
	var certs bytes.Buffer
	for _, block := range blocks {
		if block.Type == "CERTIFICATE" {
			block.Headers = nil
			_ = pem.Encode(&certs, block)
		}
	}
	return certs.Bytes(), nil
}

// keyStoreError mirrors FtpUtil.loadKeyStore's "Failed to load KeyStore from path" wording.
func keyStoreError(path, detail string) error {
	return errors.New("Failed to load KeyStore from path: " + path + ". " + detail)
}

func readKeyStore(platform pal.Platform, path string) ([]byte, error) {
	data, err := platform.FS.ReadFile(path)
	if err != nil {
		return nil, keyStoreError(path, "KeyStore file does not exist: "+path)
	}
	if len(data) == 0 {
		return nil, keyStoreError(path, "KeyStore file is empty: "+path)
	}
	if bytes.HasPrefix(data, []byte("-----BEGIN ")) {
		return nil, keyStoreError(path, "File looks like a PEM (starts with '-----BEGIN '); expected a PKCS12 keystore.")
	}
	return data, nil
}
