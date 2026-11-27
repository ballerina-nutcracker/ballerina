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
	"errors"
	"io/fs"
	"slices"
	"strings"

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
	if err := restrictCipherSuites(tlsCfg, stringOrList(secureSocket, "ciphers")); err != nil {
		return nil, err
	}
	return tlsCfg, nil
}

// tls13Suites are the TLS 1.3 suites, which crypto/tls always enables together.
var tls13Suites = []string{"TLS_AES_128_GCM_SHA256", "TLS_AES_256_GCM_SHA384", "TLS_CHACHA20_POLY1305_SHA256"}

// javaOnlyCipherSuites are the suites Java 21 supports and crypto/tls lacks; they are skipped.
var javaOnlyCipherSuites = []string{
	"TLS_DHE_DSS_WITH_AES_128_CBC_SHA", "TLS_DHE_DSS_WITH_AES_128_CBC_SHA256", "TLS_DHE_DSS_WITH_AES_128_GCM_SHA256",
	"TLS_DHE_DSS_WITH_AES_256_CBC_SHA", "TLS_DHE_DSS_WITH_AES_256_CBC_SHA256", "TLS_DHE_DSS_WITH_AES_256_GCM_SHA384",
	"TLS_DHE_RSA_WITH_AES_128_CBC_SHA", "TLS_DHE_RSA_WITH_AES_128_CBC_SHA256", "TLS_DHE_RSA_WITH_AES_128_GCM_SHA256",
	"TLS_DHE_RSA_WITH_AES_256_CBC_SHA", "TLS_DHE_RSA_WITH_AES_256_CBC_SHA256", "TLS_DHE_RSA_WITH_AES_256_GCM_SHA384",
	"TLS_DHE_RSA_WITH_CHACHA20_POLY1305_SHA256", "TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA384",
	"TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA384", "TLS_EMPTY_RENEGOTIATION_INFO_SCSV", "TLS_RSA_WITH_AES_256_CBC_SHA256",
}

// restrictCipherSuites enables the listed suites that crypto/tls supports, skipping the ones only
// Java supports as long as one is usable; any other name is an error. Without a TLS 1.3 suite in
// the list, TLS 1.3 is not negotiated.
func restrictCipherSuites(tlsCfg *pal.TLSConfig, names []string) *values.Error {
	if len(names) == 0 {
		return nil
	}
	tls13 := 0
	for _, name := range names {
		switch {
		case knownCipherSuite(name):
			if slices.Contains(tlsCfg.CipherSuiteNames, name) {
				continue
			}
			tlsCfg.CipherSuiteNames = append(tlsCfg.CipherSuiteNames, name)
			if slices.Contains(tls13Suites, name) {
				tls13++
			}
		case !slices.Contains(javaOnlyCipherSuites, name):
			return emailError("Unsupported cipher suite: " + name)
		}
	}
	tls13Enabled := tlsCfg.MaxVersion == 0 || tlsCfg.MaxVersion >= tls.VersionTLS13
	switch {
	case len(tlsCfg.CipherSuiteNames) == 0:
		return emailError("Unsupported cipher suite: " + names[0])
	case tls13 == 0 && tls13Enabled:
		tlsCfg.MaxVersion = tls.VersionTLS12
	case tls13 < len(tls13Suites) && tls13Enabled:
		return emailError("TLS 1.3 cipher suites cannot be restricted; list all of " + strings.Join(tls13Suites, ", ") + " or none")
	}
	if !slices.ContainsFunc(tlsCfg.CipherSuiteNames, func(name string) bool { return runsWithin(name, tlsCfg) }) {
		return emailError("The cipher suites cannot be used with the TLS protocol versions")
	}
	return nil
}

// runsWithin reports whether the suite supports a version in the enabled range, which defaults to
// TLS 1.2 through TLS 1.3 as in crypto/tls.
func runsWithin(name string, tlsCfg *pal.TLSConfig) bool {
	lo, hi := tlsCfg.MinVersion, tlsCfg.MaxVersion
	if lo == 0 {
		lo = tls.VersionTLS12
	}
	if hi == 0 {
		hi = tls.VersionTLS13
	}
	for _, suite := range append(tls.CipherSuites(), tls.InsecureCipherSuites()...) {
		if suite.Name == name {
			return slices.ContainsFunc(suite.SupportedVersions, func(v uint16) bool { return lo <= v && v <= hi })
		}
	}
	return false
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
