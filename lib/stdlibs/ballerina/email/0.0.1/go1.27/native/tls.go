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
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"io/fs"
	"slices"

	"github.com/ballerina-nutcracker/ballerina/platform/pal"
	"github.com/ballerina-nutcracker/ballerina/runtime"
	"github.com/ballerina-nutcracker/ballerina/values"
)

var tlsVersionByName = map[string]uint16{
	"TLSv1":   tls.VersionTLS10,
	"TLSv1.0": tls.VersionTLS10,
	"TLSv1.1": tls.VersionTLS11,
	"TLSv1.2": tls.VersionTLS12,
	"TLSv1.3": tls.VersionTLS13,
}

// tlsConfigFromSecureSocket translates an email:SecureSocket into the TLS settings used by
// whichever TLS mode (implicit or STARTTLS) the security setting selects. An absent secureSocket
// trusts the platform's default roots, and an absent protocol or empty versions list keeps the
// platform's default versions. The listed versions enable the range from the lowest to the highest,
// as crypto/tls cannot enable a set with gaps. Every certificate in the cert file is trusted.
func tlsConfigFromSecureSocket(rt *runtime.Runtime, config *values.Map) (*pal.TLSConfig, *values.Error) {
	v, ok := config.Get("secureSocket")
	if !ok {
		return &pal.TLSConfig{}, nil
	}
	secureSocket, _ := v.(*values.Map)
	tlsCfg := &pal.TLSConfig{}
	if verify, ok := secureSocket.Get("verifyHostName"); ok && verify == false {
		tlsCfg.SkipHostnameVerification = true
	}
	certPath, _ := stringField(secureSocket, "cert")
	certPEM, err := readCertificates(rt, certPath)
	if err != nil {
		return nil, err
	}
	tlsCfg.CACertPEM = certPEM
	if p, ok := secureSocket.Get("protocol"); ok {
		protocol, _ := p.(*values.Map)
		for _, name := range stringOrList(protocol, "versions") {
			version, known := tlsVersionByName[name]
			if !known {
				return nil, emailError("Unsupported TLS protocol version: " + name)
			}
			if tlsCfg.MinVersion == 0 || version < tlsCfg.MinVersion {
				tlsCfg.MinVersion = version
			}
			tlsCfg.MaxVersion = max(tlsCfg.MaxVersion, version)
		}
	}
	for _, name := range stringOrList(secureSocket, "ciphers") {
		if !knownCipherSuite(name) {
			return nil, emailError("Unsupported cipher suite: " + name)
		}
		tlsCfg.CipherSuiteNames = append(tlsCfg.CipherSuiteNames, name)
	}
	return tlsCfg, nil
}

// readCertificates returns the certificate file as PEM, accepting PEM bundles and a single DER
// certificate as Java's CertificateFactory does.
func readCertificates(rt *runtime.Runtime, path string) ([]byte, *values.Error) {
	data, err := rt.Platform().FS.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, emailError(path + " (No such file or directory)")
		}
		return nil, emailErrorWithCause("Error while reading the certificate file "+path, err)
	}
	if block, _ := pem.Decode(data); block != nil {
		if !x509.NewCertPool().AppendCertsFromPEM(data) {
			return nil, emailError("Could not parse certificate: " + path)
		}
		return data, nil
	}
	certs, parseErr := x509.ParseCertificates(data)
	if parseErr != nil || len(certs) == 0 {
		return nil, emailError("Could not parse certificate: " + path)
	}
	var out []byte
	for _, cert := range certs {
		out = append(out, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw})...)
	}
	return out, nil
}

func knownCipherSuite(name string) bool {
	return slices.ContainsFunc(append(tls.CipherSuites(), tls.InsecureCipherSuites()...),
		func(suite *tls.CipherSuite) bool { return suite.Name == name })
}
