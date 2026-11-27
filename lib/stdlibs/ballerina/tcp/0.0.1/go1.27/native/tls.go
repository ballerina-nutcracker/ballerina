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
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"slices"
	"time"

	"github.com/ballerina-nutcracker/ballerina/decimal"
	"github.com/ballerina-nutcracker/ballerina/platform/pal"
	"github.com/ballerina-nutcracker/ballerina/runtime"
	"github.com/ballerina-nutcracker/ballerina/values"
)

func decimalToDuration(d *decimal.Decimal) time.Duration {
	return time.Duration(d.Float64() * float64(time.Second))
}

// tlsVersionNames lists the protocol names Java accepts, in version order; TLS 1.0 is TLSv1.
var tlsVersionNames = []string{"TLSv1", "TLSv1.1", "TLSv1.2", "TLSv1.3"}

// versionRange returns the range spanning the listed versions. crypto/tls enables a min–max range only,
// so a list that skips a version is an error rather than enabling the version it skips.
func versionRange(list *values.List) (minVersion, maxVersion uint16, err *values.Error) {
	if list == nil {
		return 0, 0, nil
	}
	listed := make([]bool, len(tlsVersionNames))
	lo, hi := len(tlsVersionNames), -1
	for i := 0; i < list.Len(); i++ {
		name, _ := list.Get(i).(string)
		idx := slices.Index(tlsVersionNames, name)
		if idx < 0 {
			return 0, 0, tcpError("Unsupported protocol: " + name)
		}
		listed[idx] = true
		lo, hi = min(lo, idx), max(hi, idx)
	}
	if hi < 0 {
		return 0, 0, nil
	}
	for idx := lo + 1; idx < hi; idx++ {
		if !listed[idx] {
			return 0, 0, tcpError("Unsupported protocol versions: " + tlsVersionNames[lo] + " and " + tlsVersionNames[hi] +
				" cannot be enabled without " + tlsVersionNames[idx])
		}
	}
	return tls.VersionTLS10 + uint16(lo), tls.VersionTLS10 + uint16(hi), nil
}

func cipherNames(v values.BalValue) []string {
	list, ok := v.(*values.List)
	if !ok || list.Len() == 0 {
		return nil
	}
	names := make([]string, 0, list.Len())
	for i := 0; i < list.Len(); i++ {
		if s, ok := list.Get(i).(string); ok {
			names = append(names, s)
		}
	}
	return names
}

// noUsableCipher reports a list naming no suite crypto/tls supports; Java fails its handshake, where crypto/tls
// would fall back to its default suites.
func noUsableCipher(names []string) bool {
	return len(names) > 0 && !slices.ContainsFunc(names, isCipherSuite)
}

func noCipherError() *values.Error {
	return tcpError("No appropriate protocol (protocol is disabled or cipher suites are inappropriate)")
}

func isCipherSuite(name string) bool {
	named := func(suite *tls.CipherSuite) bool { return suite.Name == name }
	return slices.ContainsFunc(tls.CipherSuites(), named) || slices.ContainsFunc(tls.InsecureCipherSuites(), named)
}

func protocolVersionsFrom(secureSocket *values.Map) *values.List {
	v, ok := secureSocket.Get("protocol")
	if !ok {
		return nil
	}
	protoMap, ok := v.(*values.Map)
	if !ok {
		return nil
	}
	versions, ok := protoMap.Get("versions")
	if !ok {
		return nil
	}
	list, _ := versions.(*values.List)
	return list
}

// buildClientTLSConfig translates tcp:ClientSecureSocket into a pal.TLSConfig.
// Returns (nil, nil) when secureSocket is absent or explicitly disabled.
func buildClientTLSConfig(rt *runtime.Runtime, secureSocket *values.Map) (*pal.TLSConfig, *values.Error) {
	if secureSocket == nil {
		return nil, nil
	}
	if v, ok := secureSocket.Get("enable"); ok {
		if enable, ok := v.(bool); ok && !enable {
			return nil, nil
		}
	}

	var tlsCfg pal.TLSConfig
	if v, ok := secureSocket.Get("cert"); ok && v != nil {
		switch cert := v.(type) {
		case string:
			if cert != "" {
				data, err := rt.Platform().FS.ReadFile(cert)
				if err != nil || !x509.NewCertPool().AppendCertsFromPEM(data) {
					return nil, tcpError("File does not contain valid certificates: " + cert)
				}
				tlsCfg.CACertPEM = data
			}
		case *values.Map:
			return nil, tcpError("secureSocket.cert: crypto:TrustStore is not yet supported; " +
				"use a PEM certificate file path string instead")
		}
	}
	if v, ok := secureSocket.Get("verifyHostName"); ok {
		if verify, ok := v.(bool); ok && !verify {
			tlsCfg.SkipHostnameVerification = true
		}
	}
	if v, ok := secureSocket.Get("ciphers"); ok {
		tlsCfg.CipherSuiteNames = cipherNames(v)
	}
	if d, ok := decimalArg(secureSocket, "handshakeTimeout"); ok {
		tlsCfg.HandshakeTimeout = decimalToDuration(d)
	}
	var verErr *values.Error
	if tlsCfg.MinVersion, tlsCfg.MaxVersion, verErr = versionRange(protocolVersionsFrom(secureSocket)); verErr != nil {
		return nil, verErr
	}
	return &tlsCfg, nil
}

// buildListenerTLSConfig translates tcp:ListenerSecureSocket into a
// pal.ServerTLSConfig. TLS is enabled purely by ListenerConfiguration.secureSocket
// being present — unlike the client side there is no separate enable flag.
// The cert and key files are read by loadListenerKeyPair at start(), as in jBallerina.
func buildListenerTLSConfig(secureSocket *values.Map) (cfg *pal.ServerTLSConfig, certPath, keyPath string, _ *values.Error) {
	keyVal, ok := secureSocket.Get("key")
	if !ok {
		return nil, "", "", tcpError("secureSocket.key is required")
	}
	keyMap, ok := keyVal.(*values.Map)
	if !ok {
		return nil, "", "", tcpError("secureSocket.key: crypto:KeyStore is not yet supported; " +
			"use a CertKey (certFile/keyFile) value instead")
	}
	certFile, _ := keyMap.Get("certFile")
	keyFile, _ := keyMap.Get("keyFile")
	certPath, _ = certFile.(string)
	keyPath, _ = keyFile.(string)
	if certPath == "" || keyPath == "" {
		return nil, "", "", tcpError("secureSocket.key: certFile and keyFile are required")
	}

	tlsCfg := &pal.ServerTLSConfig{}
	if v, ok := secureSocket.Get("ciphers"); ok {
		if tlsCfg.CipherSuiteNames = cipherNames(v); noUsableCipher(tlsCfg.CipherSuiteNames) {
			return nil, "", "", noCipherError()
		}
	}
	var verErr *values.Error
	if tlsCfg.MinVersion, tlsCfg.MaxVersion, verErr = versionRange(protocolVersionsFrom(secureSocket)); verErr != nil {
		return nil, "", "", verErr
	}
	return tlsCfg, certPath, keyPath, nil
}

func loadListenerKeyPair(rt *runtime.Runtime, cfg *pal.ServerTLSConfig, certPath, keyPath string) *values.Error {
	certPEM, err := rt.Platform().FS.ReadFile(certPath)
	if err != nil || !x509.NewCertPool().AppendCertsFromPEM(certPEM) {
		return tcpError("File does not contain valid certificates: " + certPath)
	}
	keyPEM, err := rt.Platform().FS.ReadFile(keyPath)
	if err != nil || !containsPrivateKey(keyPEM) {
		return tcpError("File does not contain valid private key: " + keyPath)
	}
	if _, err := tls.X509KeyPair(certPEM, keyPEM); err != nil {
		return tcpError("File does not contain a private key matching the certificate: " + keyPath)
	}
	cfg.CertPEM, cfg.KeyPEM = certPEM, keyPEM
	return nil
}

func containsPrivateKey(data []byte) bool {
	for block, rest := pem.Decode(data); block != nil; block, rest = pem.Decode(rest) {
		if _, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
			return true
		}
		if _, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
			return true
		}
		if _, err := x509.ParseECPrivateKey(block.Bytes); err == nil {
			return true
		}
	}
	return false
}

func decimalArg(m *values.Map, key string) (*decimal.Decimal, bool) {
	v, ok := m.Get(key)
	if !ok {
		return nil, false
	}
	d, ok := v.(*decimal.Decimal)
	return d, ok
}
