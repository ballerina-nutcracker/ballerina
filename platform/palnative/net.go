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

// Native-CLI implementation of the pal.Net contract: a raw TCP(+TLS) dialer
// for non-HTTP wire protocols (e.g. ldap). NewPlatform (in pal.go) wires
// Dial into pal.Net.Dial.

package palnative

import (
	"context"
	"crypto/tls"
	"net"

	"ballerina-lang-go/platform/pal"
)

// Dial is the pal.Net.Dial factory for the native-CLI platform. It opens a
// plain TCP connection, then upgrades it to TLS in-place when tlsCfg is
// non-nil, performing the handshake before returning.
func Dial(ctx context.Context, network, address string, tlsCfg *pal.TLSConfig) (net.Conn, error) {
	var dialer net.Dialer
	conn, err := dialer.DialContext(ctx, network, address)
	if err != nil {
		return nil, err
	}
	if tlsCfg == nil {
		return conn, nil
	}
	tlsConn := tls.Client(conn, buildTLSConfig(*tlsCfg))
	if err := tlsConn.HandshakeContext(ctx); err != nil {
		_ = conn.Close()
		return nil, err
	}
	return tlsConn, nil
}
