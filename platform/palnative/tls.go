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

// Shared TLS config assembly for native-CLI PAL implementations that dial
// outbound connections (http.go's HTTP client, net.go's raw socket dialer).

package palnative

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"

	"github.com/ballerina-nutcracker/ballerina/platform/pal"
)

// buildTLSConfig assembles a *tls.Config from a pal.TLSConfig, resolving CA
// pools, client certificates, SNI, cipher suites, and protocol version
// bounds. Shared by every native-CLI PAL category that opens a TLS
// connection. Fails closed on a malformed CA or client certificate: jBallerina's
// own TLS setup (netty's SslContextBuilder.trustManager/keyManager) throws
// immediately on invalid PEM material rather than silently falling back to
// the system trust store or an unauthenticated connection.
func buildTLSConfig(cfg pal.TLSConfig) (*tls.Config, error) {
	tlsConfig := &tls.Config{InsecureSkipVerify: cfg.InsecureSkipVerify} //nolint:gosec
	if len(cfg.CACertPEM) > 0 {
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(cfg.CACertPEM) {
			return nil, fmt.Errorf("failed to parse CA certificate PEM: no valid certificates found")
		}
		tlsConfig.RootCAs = pool
		if !cfg.InsecureSkipVerify {
			// Go 1.15+ requires SANs for hostname verification; many self-signed and
			// Java-issued certs only set the CN field. When a custom CA is provided
			// we do our own verification so CN-only certs are accepted as a fallback.
			tlsConfig.InsecureSkipVerify = true //nolint:gosec
			tlsConfig.VerifyConnection = tlsVerifyConnectionWithCNFallback(pool, cfg.ServerName)
		}
	}
	if len(cfg.ClientCertPEM) > 0 && len(cfg.ClientKeyPEM) > 0 {
		cert, err := tls.X509KeyPair(cfg.ClientCertPEM, cfg.ClientKeyPEM)
		if err != nil {
			return nil, fmt.Errorf("failed to load client certificate: %w", err)
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}
	tlsConfig.ServerName = cfg.ServerName
	tlsConfig.SessionTicketsDisabled = cfg.DisableSessionTickets
	tlsConfig.MinVersion = tls.VersionTLS12 // secure default; overridden below if configured
	if cfg.MinVersion != 0 {
		tlsConfig.MinVersion = cfg.MinVersion
	}
	if cfg.MaxVersion != 0 {
		tlsConfig.MaxVersion = cfg.MaxVersion
	}
	if len(cfg.CipherSuiteNames) > 0 {
		if resolved := resolveCipherSuites(cfg.CipherSuiteNames); len(resolved) > 0 {
			tlsConfig.CipherSuites = resolved
		} else {
			_, _ = fmt.Fprintf(os.Stderr, "warning: no valid cipher suites resolved from cfg.CipherSuiteNames %v; keeping secure defaults\n", cfg.CipherSuiteNames)
		}
	}
	return tlsConfig, nil
}
