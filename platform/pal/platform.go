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

// Package pal provides the Platform Adaptation Layer (PAL).
//
// PAL abstracts away interactions with the underlying platform such that the
// runtime can be agnostic toward the underlying platform. All library functions
// that interact with the underlying platform (e.g. io, http) should use PAL.
// Each supported platform (e.g. native-cli, web-editor) should provide an
// implementation of PAL to the runtime.
package pal

import (
	"context"
	"io"
	"net"
	"net/http"
	"time"
)

// Signal is a platform-agnostic shutdown request delivered to the runtime.
// Each platform implementation decides how to source these (OS signals,
// HTTP control plane, test injection, ...).
type Signal uint8

const (
	GracefulStop Signal = iota
	ImmediateStop
)

// SignalSource is the runtime's read-only view of incoming shutdown signals.
// Implementations close the channel only on platform shutdown; the runtime
// must treat a closed channel as "no more signals" (not as a signal value).
type SignalSource struct {
	Signals <-chan Signal
}

type (
	Platform struct {
		IO      IO
		FS      FS
		OS      OS
		Time    Time
		HTTP    HTTP
		Net     Net
		Signals SignalSource
	}
	IO struct {
		Stdout func(p []byte) (n int, err error)
		Stderr func(p []byte) (n int, err error)
	}
	FS struct {
		ReadFile      func(path string) ([]byte, error)
		WriteFile     func(path string, data []byte) error
		AppendFile    func(path string, data []byte) error
		Getwd         func() (string, error)
		Mkdir         func(path string) error
		MkdirAll      func(path string) error
		Remove        func(path string) error
		RemoveAll     func(path string) error
		Rename        func(oldPath, newPath string) error
		CreateFile    func(path string) error
		Stat          func(path string) (*FileInfo, error)
		Lstat         func(path string) (*FileInfo, error)
		ReadDir       func(path string) ([]FileInfo, error)
		Copy          func(src, dst string, opts CopyOptions) error
		CreateTemp    func(prefix, suffix, dir string) (string, error)
		CreateTempDir func(prefix, suffix, dir string) (string, error)
		Readlink      func(path string) (string, error)
		// Watch starts monitoring path for filesystem changes, optionally
		// descending into subdirectories, delivering each change to handler on
		// a platform-owned goroutine. The returned handle's Close stops
		// watching and releases OS resources (e.g. inotify/kqueue handles).
		Watch func(path string, recursive bool, handler WatchHandler) (WatchHandle, error)
	}
	OS struct {
		GetEnv      func(name string) string
		GetUsername func() string
		GetUserHome func() string
		SetEnv      func(key, val string) error
		UnsetEnv    func(key string) error
		ListEnv     func() map[string]string
		Exec        func(command string, args []string, envOverride map[string]string) (ProcessHandle, error)
	}
	Time struct {
		Now          func() time.Time
		MonotonicNow func() time.Duration
		Sleep        func(d time.Duration)
	}
	// Net abstracts raw TCP(+TLS) socket dialing and listening for non-HTTP
	// wire protocols (e.g. ldap, tcp). HTTP-based stdlibs must keep using
	// pal.HTTP instead.
	Net struct {
		// Dial opens a TCP connection to address, optionally binding to
		// localAddr first (empty = OS-chosen) and optionally upgrading to
		// TLS (tlsCfg != nil) before returning. Reuses TLSConfig from HTTP.
		Dial func(ctx context.Context, network, address, localAddr string, tlsCfg *TLSConfig) (net.Conn, error)
		// Listen binds a TCP listener on address, optionally wrapping it in
		// TLS (tlsCfg != nil). Reuses ServerTLSConfig from HTTP. The caller
		// owns the accept loop and per-connection dispatch.
		Listen func(network, address string, tlsCfg *ServerTLSConfig) (net.Listener, error)
		// DialPacket opens a connected datagram socket (e.g. UDP) to address,
		// optionally binding to localAddr first (empty = OS-chosen). The
		// returned net.Conn's Read/Write only exchange data with address.
		DialPacket func(ctx context.Context, network, address, localAddr string) (net.Conn, error)
		// ListenPacket binds an unconnected datagram socket (e.g. UDP) on
		// address. The returned net.PacketConn's ReadFrom/WriteTo carry an
		// explicit per-datagram peer address, unlike DialPacket's net.Conn.
		ListenPacket func(network, address string) (net.PacketConn, error)
	}
	HTTP struct {
		// NewClient builds an outbound HTTP client (native: net/http; WASM: fetch).
		NewClient func(cfg ClientConfig) HTTPClient
		// Listen starts serving inbound requests to handler. On native it binds a
		// TCP socket and runs http.Server.Serve; a WASM/web platform registers the
		// handler with its JS host instead (no socket). Returns a handle whose
		// Shutdown/Close control the listener lifecycle.
		Listen func(cfg ServerConfig, handler http.Handler) (ServerHandle, error)
	}
)

// ProcessHandle is an opaque handle to a running subprocess.
type ProcessHandle interface {
	WaitForExit() (int, error)
	ReadStdout() ([]byte, error)
	ReadStderr() ([]byte, error)
	Kill()
}

// HTTP
type (
	// TLSConfig carries TLS settings derived from Ballerina's secureSocket config.
	TLSConfig struct {
		InsecureSkipVerify    bool          // secureSocket.enable=false OR verifyHostName=false
		CACertPEM             []byte        // secureSocket.cert (string PEM file path) → file contents
		ClientCertPEM         []byte        // secureSocket.key.certFile → file contents
		ClientKeyPEM          []byte        // secureSocket.key.keyFile  → file contents
		ServerName            string        // secureSocket.serverName → tls.Config.ServerName (SNI)
		CipherSuiteNames      []string      // secureSocket.ciphers → IANA names; platform resolves IDs
		MinVersion            uint16        // secureSocket.protocol.versions min → tls.Config.MinVersion
		MaxVersion            uint16        // secureSocket.protocol.versions max → tls.Config.MaxVersion
		HandshakeTimeout      time.Duration // secureSocket.handshakeTimeout → transport.TLSHandshakeTimeout
		DisableSessionTickets bool          // secureSocket.shareSession=false → tls.Config.SessionTicketsDisabled
	}
	// FollowRedirects controls HTTP redirect behaviour, matching Ballerina's http:FollowRedirects.
	FollowRedirects struct {
		Enabled          bool // default false — no redirects by default (Ballerina spec)
		MaxCount         int  // 0 uses Ballerina default of 5; only used when Enabled=true
		AllowAuthHeaders bool // if true, forward Authorization/Proxy-Authorization on redirect
	}
	// PoolConfig carries connection pool settings derived from Ballerina's http:PoolConfiguration.
	// Zero values trigger platform-chosen defaults that mirror jBallerina's pool config.
	PoolConfig struct {
		// MaxIdleConnsPerHost is the per-host idle connection pool size.
		// 0 → default (100); matches jBallerina maxIdleConnections=100.
		MaxIdleConnsPerHost int
		// MaxIdleConns is the global idle pool size across all hosts.
		// 0 → default (512).
		MaxIdleConns int
		// MaxConnsPerHost is the maximum total (active+idle) connections per host.
		// 0 = unlimited; matches jBallerina maxActiveConnections=-1.
		MaxConnsPerHost int
		// IdleConnTimeout is how long idle connections are kept in the pool before eviction.
		// 0 → default (300s); matches jBallerina minEvictableIdleTime=300.
		IdleConnTimeout time.Duration
		// DialTimeout is the maximum time to establish a new TCP connection.
		// 0 → default (15s); matches jBallerina socketConfig.connectTimeOut=15.
		DialTimeout time.Duration
		// ResponseHeaderTimeout limits the time waiting for the first response byte.
		// 0 = no limit (disabled by default; opt-in via poolConfig).
		ResponseHeaderTimeout time.Duration
		// WriteBufferSize and ReadBufferSize size the per-connection user-space I/O buffers.
		// 0 → default (32 KB each).
		WriteBufferSize int
		ReadBufferSize  int
		// DisableCompression prevents the transport from injecting Accept-Encoding: gzip.
		DisableCompression bool
	}
	// ResponseLimitConfig carries response size limits derived from Ballerina's http:ResponseLimitConfigs.
	// Zero values mean "use defaults" (set explicitly in Client.initNative to match jBallerina defaults).
	ResponseLimitConfig struct {
		// MaxStatusLineLength is the maximum byte length of the HTTP response status line.
		// Accepted for config compatibility; not enforced at runtime (no Go transport equivalent).
		// jBallerina default: 4096.
		MaxStatusLineLength int
		// MaxHeaderSize maps to http.Transport.MaxResponseHeaderBytes.
		// jBallerina default: 8192. 0 = Go transport default (10 MB).
		MaxHeaderSize int64
		// MaxEntityBodySize is the maximum byte length of the response body.
		// -1 = no limit (jBallerina default). ≥0 enforced per-response via a counting reader.
		MaxEntityBodySize int64
	}
	// ProxyConfig carries HTTP proxy settings derived from Ballerina's http:ProxyConfig.
	ProxyConfig struct {
		Host     string // proxy hostname; empty = no proxy
		Port     int    // proxy port
		UserName string // proxy auth username; empty = no auth
		Password string // proxy auth password
	}
	// ClientConfig bundles all static options for a new HTTP client instance.
	ClientConfig struct {
		Timeout         time.Duration
		FollowRedirects FollowRedirects
		HTTPVersion     string // "1.1" or "2.0"; defaults to "2.0"
		TLS             TLSConfig
		Pool            PoolConfig
		ResponseLimits  ResponseLimitConfig
		Proxy           ProxyConfig
	}
	// HTTPClient is an opaque handle to an HTTP client created by the platform.
	// It is created once per Ballerina http:Client init and reused across requests.
	HTTPClient interface {
		Execute(ctx context.Context, method, url string, body io.Reader, contentLength int64, contentType string, reqHeaders map[string][]string) (statusCode int, respHeaders map[string][]string, respBody io.ReadCloser, err error)
	}

	// ServerTLSConfig carries TLS settings for an HTTP listener, derived from
	// Ballerina's http:ListenerSecureSocket. The caller pre-reads all PEM material
	// (via FS) so the platform never touches the filesystem; the platform assembles
	// the concrete TLS configuration. WASM/web platforms may ignore this entirely.
	ServerTLSConfig struct {
		CertPEM               []byte   // key.certFile contents — server certificate
		KeyPEM                []byte   // key.keyFile contents  — server private key
		ClientCACertPEM       []byte   // mutualSsl CA pool; non-empty → require+verify client cert
		MinVersion            uint16   // TLS min version (e.g. 0x0303 for TLSv1.2); 0 = platform default
		MaxVersion            uint16   // TLS max version; 0 = unset
		CipherSuiteNames      []string // IANA cipher names; platform resolves to IDs
		DisableSessionTickets bool     // shareSession=false → disable TLS session tickets
	}
	// ServerConfig bundles the static options for an HTTP listener.
	ServerConfig struct {
		Host         string
		Port         int
		HTTPVersion  string           // "1.1" or "2.0"
		WriteTimeout time.Duration    // 0 → platform default
		TLS          *ServerTLSConfig // nil = plaintext
	}
	// ServerHandle is an opaque handle to a started HTTP listener, returned by
	// HTTP.Listen. It owns the listener lifecycle.
	ServerHandle interface {
		Shutdown(ctx context.Context) error // graceful: drain in-flight requests, then close
		Close() error                       // immediate: close all connections now
	}
)

// FileInfo carries metadata for a single filesystem entry.
type FileInfo struct {
	AbsPath    string
	Size       int64
	ModifiedAt time.Time
	IsDir      bool
	IsSymlink  bool
	IsReadable bool
	IsWritable bool
}

// CopyOptions controls the behavior of FS.Copy.
type CopyOptions struct {
	ReplaceExisting bool
	CopyAttributes  bool
	NoFollowLinks   bool
}

// WatchOp identifies the kind of filesystem change reported by FS.Watch.
type WatchOp uint8

const (
	WatchCreate WatchOp = iota
	WatchModify
	WatchDelete
)

// WatchEvent carries a single filesystem change notification from FS.Watch.
type WatchEvent struct {
	Path string
	Op   WatchOp
}

// WatchHandler receives filesystem change notifications from FS.Watch. It is
// invoked on a platform-owned goroutine, independent of the strand that
// called FS.Watch.
type WatchHandler func(WatchEvent)

// WatchHandle is an opaque handle to an active filesystem watch, returned by
// FS.Watch. Close stops watching and releases OS resources.
type WatchHandle interface {
	Close() error
}
