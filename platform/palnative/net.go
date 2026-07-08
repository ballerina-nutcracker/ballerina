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
// and listener for non-HTTP wire protocols (e.g. ldap, tcp). NewPlatform (in
// pal.go) wires Dial/Listen into pal.Net.

package palnative

import (
	"context"
	"crypto/tls"
	"net"

	"ballerina-lang-go/platform/pal"
)

// Dial is the pal.Net.Dial factory for the native-CLI platform. It opens a
// plain TCP connection, optionally bound to localAddr, then upgrades it to
// TLS in-place when tlsCfg is non-nil, performing the handshake before
// returning.
func Dial(ctx context.Context, network, address, localAddr string, tlsCfg *pal.TLSConfig) (net.Conn, error) {
	dialer := net.Dialer{}
	if localAddr != "" {
		addr, err := net.ResolveTCPAddr(network, localAddr)
		if err != nil {
			return nil, err
		}
		dialer.LocalAddr = addr
	}
	conn, err := dialer.DialContext(ctx, network, address)
	if err != nil {
		return nil, err
	}
	if tlsCfg == nil {
		return conn, nil
	}
	effectiveTLSCfg := *tlsCfg
	if effectiveTLSCfg.ServerName == "" {
		// tls.Client (unlike tls.Dial) never derives ServerName from the
		// dial address on its own, so hostname verification would otherwise
		// always fail against an empty expected name. Must be set before
		// buildTLSConfig runs: it's captured by value into the
		// VerifyConnection closure when a custom CA is configured.
		if host, _, splitErr := net.SplitHostPort(address); splitErr == nil {
			effectiveTLSCfg.ServerName = host
		} else {
			effectiveTLSCfg.ServerName = address
		}
	}
	tlsConn := tls.Client(conn, buildTLSConfig(effectiveTLSCfg))
	if err := tlsConn.HandshakeContext(ctx); err != nil {
		_ = conn.Close()
		return nil, err
	}
	return tlsConn, nil
}

// ListenTCP is the pal.Net.Listen factory for the native-CLI platform. It
// binds a plain TCP listener, optionally wrapping it in TLS when tlsCfg is
// non-nil. The caller owns the accept loop and per-connection dispatch.
// Named distinctly from httpserver.go's Listen (pal.HTTP.Listen), which owns
// its own accept loop internally via http.Server.
func ListenTCP(network, address string, tlsCfg *pal.ServerTLSConfig) (net.Listener, error) {
	ln, err := net.Listen(network, address)
	if err != nil {
		return nil, err
	}
	if tlsCfg == nil {
		return ln, nil
	}
	cfg, err := buildServerTLSConfig(tlsCfg)
	if err != nil {
		_ = ln.Close()
		return nil, err
	}
	return tls.NewListener(ln, cfg), nil
}

// DialPacket is the pal.Net.DialPacket factory for the native-CLI platform.
// It opens a connected datagram socket (e.g. UDP) to address, optionally
// bound to localAddr first. There is no TLS analogue for datagram sockets.
func DialPacket(ctx context.Context, network, address, localAddr string) (net.Conn, error) {
	dialer := net.Dialer{}
	if localAddr != "" {
		addr, err := net.ResolveUDPAddr(network, localAddr)
		if err != nil {
			return nil, err
		}
		dialer.LocalAddr = addr
	}
	return dialer.DialContext(ctx, network, address)
}

// ListenPacket is the pal.Net.ListenPacket factory for the native-CLI
// platform. It binds an unconnected datagram socket (e.g. UDP) on address;
// the caller owns the read loop and per-datagram dispatch.
func ListenPacket(network, address string) (net.PacketConn, error) {
	return net.ListenPacket(network, address)
}
