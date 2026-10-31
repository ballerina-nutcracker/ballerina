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
	"strings"
	"time"

	"github.com/ballerina-nutcracker/ballerina/platform/pal"
	"github.com/ballerina-nutcracker/ballerina/values"
)

const (
	protocolFTPS = "ftps"
	protocolSFTP = "sftp"
)

func defaultPort(protocol string) int64 {
	if protocol == protocolSFTP {
		return 22
	}
	return 21
}

// endpoint is the connection part of a client or listener configuration.
type endpoint struct {
	protocol      string
	host          string
	port          int64
	username      string
	password      string
	hasPassword   bool
	userDirIsRoot bool

	connectTimeout   time.Duration
	ftpDataTimeout   time.Duration
	ftpSocketTimeout time.Duration
	asciiMode        bool

	tlsConfig   *pal.TLSConfig
	implicitTLS bool

	privateKeyPath     string
	privateKeyPassword string
	preferredMethods   []string
	knownHostsPath     string
}

type retrySettings struct {
	count           int64
	interval        float64
	backOffFactor   float64
	maxWaitInterval float64
}

func getMap(m *values.Map, key string) *values.Map {
	v, _ := m.Get(key)
	sub, _ := v.(*values.Map)
	return sub
}

func getString(m *values.Map, key string) (string, bool) {
	v, _ := m.Get(key)
	s, ok := v.(string)
	return s, ok
}

func getInt(m *values.Map, key string) int64 {
	v, _ := m.Get(key)
	i, _ := v.(int64)
	return i
}

func getBool(m *values.Map, key string) bool {
	v, _ := m.Get(key)
	b, _ := v.(bool)
	return b
}

func getFloat(m *values.Map, key string) float64 {
	v, _ := m.Get(key)
	f, _ := values.NumericConvertToFloat(v)
	return f
}

func getStrings(m *values.Map, key string) []string {
	v, _ := m.Get(key)
	list, _ := v.(*values.List)
	if list == nil {
		return nil
	}
	out := make([]string, list.Len())
	for i := range out {
		out[i], _ = list.Get(i).(string)
	}
	return out
}

func seconds(f float64) time.Duration {
	return time.Duration(f * float64(time.Second))
}

func validateRetryConfig(config *values.Map) (*retrySettings, error) {
	retry := getMap(config, "retryConfig")
	if retry == nil {
		return nil, nil
	}
	r := &retrySettings{
		count:           getInt(retry, "count"),
		interval:        getFloat(retry, "interval"),
		backOffFactor:   getFloat(retry, "backOffFactor"),
		maxWaitInterval: getFloat(retry, "maxWaitInterval"),
	}
	switch {
	case r.count <= 0:
		return nil, newError(kindInvalidConfig, "retryConfig.count must be greater than 0.")
	case r.interval <= 0:
		return nil, newError(kindInvalidConfig, "retryConfig.interval must be greater than 0.")
	case r.backOffFactor < 1.0:
		return nil, newError(kindInvalidConfig, "retryConfig.backOffFactor must be greater than or equal to 1.0.")
	case r.maxWaitInterval <= 0:
		return nil, newError(kindInvalidConfig, "retryConfig.maxWaitInterval must be greater than 0.")
	case r.maxWaitInterval < r.interval:
		return nil, newError(kindInvalidConfig,
			"retryConfig.maxWaitInterval must be greater than or equal to retryConfig.interval.")
	}
	return r, nil
}

// parseCredentials applies the anonymous defaults: the username is "anonymous" and the
// password is empty on FTP/FTPS and absent on SFTP.
func parseCredentials(config *values.Map, ep *endpoint) {
	ep.username = "anonymous"
	ep.hasPassword = ep.protocol != protocolSFTP
	auth := getMap(config, "auth")
	if auth == nil {
		return
	}
	credentials := getMap(auth, "credentials")
	if credentials == nil {
		return
	}
	ep.username, _ = getString(credentials, "username")
	if password, ok := getString(credentials, "password"); ok {
		ep.password = password
		ep.hasPassword = true
	}
}

func parsePort(config *values.Map) (int64, error) {
	port := getInt(config, "port")
	if port < 1 || port > 65535 {
		return 0, newError(kindInvalidConfig, "Port must be between 1 and 65535 (got: "+formatInt(port)+")")
	}
	return port, nil
}

func validateAuthProtocol(auth *values.Map, protocol string) error {
	if getMap(auth, "privateKey") != nil && protocol != protocolSFTP {
		return newError(kindError, "privateKey can only be used with SFTP protocol.")
	}
	if getMap(auth, "secureSocket") != nil && protocol != protocolFTPS {
		return newError(kindError, "secureSocket can only be used with FTPS protocol.")
	}
	return nil
}

func validateTimeout(value float64, field string) error {
	if value < 0 {
		return newError(kindInvalidConfig, field+" must be positive or zero (got: "+values.FormatFloat(value)+")")
	}
	if value > 600 {
		return newError(kindInvalidConfig, field+" must not exceed 600 seconds (got: "+values.FormatFloat(value)+")")
	}
	return nil
}

func parseTimeouts(config *values.Map, ep *endpoint) error {
	connectTimeout := getFloat(config, "connectTimeout")
	if err := validateTimeout(connectTimeout, "connectTimeout"); err != nil {
		return err
	}
	ep.connectTimeout = seconds(connectTimeout)
	socketConfig := getMap(config, "socketConfig")
	if socketConfig == nil {
		return nil
	}
	for _, field := range []string{"ftpDataTimeout", "ftpSocketTimeout", "sftpSessionTimeout"} {
		if err := validateTimeout(getFloat(socketConfig, field), field); err != nil {
			return err
		}
	}
	ep.ftpDataTimeout = seconds(getFloat(socketConfig, "ftpDataTimeout"))
	ep.ftpSocketTimeout = seconds(getFloat(socketConfig, "ftpSocketTimeout"))
	return nil
}

func validateProxy(config *values.Map) error {
	proxy := getMap(config, "proxy")
	if proxy == nil {
		return nil
	}
	if host, _ := getString(proxy, "host"); strings.TrimSpace(host) == "" {
		return newError(kindInvalidConfig, "Proxy host cannot be empty")
	}
	if port := getInt(proxy, "port"); port < 1 || port > 65535 {
		return newError(kindInvalidConfig, "Proxy port must be between 1 and 65535 (got: "+formatInt(port)+")")
	}
	return nil
}

func parseSecureSocket(ep *endpoint, secureSocket *values.Map, platform pal.Platform, msgs endpointMessages) error {
	cfg := &pal.TLSConfig{}
	if secureSocket != nil {
		mode, _ := getString(secureSocket, "mode")
		ep.implicitTLS = mode == "IMPLICIT"
		cfg.SkipHostnameVerification = !getBool(secureSocket, "verifyHostName")
		if key := getMap(secureSocket, "key"); key != nil {
			keyPath, _ := getString(key, "path")
			if strings.TrimSpace(keyPath) == "" {
				return newError(msgs.privateKeyKind, msgs.keyStorePath)
			}
			password, _ := getString(key, "password")
			certPEM, keyPEM, err := keyStoreToPEM(platform, keyPath, password)
			if err != nil {
				return newError(kindError, "Failed to configure SSL/TLS certificates for FTPS: "+err.Error())
			}
			cfg.ClientCertPEM, cfg.ClientKeyPEM = certPEM, keyPEM
		}
		certValue, _ := secureSocket.Get("cert")
		var certPath, certPassword string
		hasCert := false
		switch cert := certValue.(type) {
		case string:
			certPath, hasCert = cert, true
		case *values.Map:
			certPath, _ = getString(cert, "path")
			certPassword, _ = getString(cert, "password")
			hasCert = true
		}
		if hasCert {
			if strings.TrimSpace(certPath) == "" {
				return newError(msgs.privateKeyKind, msgs.trustStorePath)
			}
			caPEM, err := trustedCertsPEM(platform, certPath, certPassword, certValue)
			if err != nil {
				return newError(kindError, "Failed to configure SSL/TLS certificates for FTPS: "+err.Error())
			}
			cfg.CACertPEM = caPEM
		}
	}
	ep.tlsConfig = cfg
	return nil
}

// endpointMessages holds the wording that differs between the client and the listener, which
// validate the same auth settings in different jBallerina classes.
type endpointMessages struct {
	privateKeyKind errorKind
	privateKeyPath string
	keyStorePath   string
	trustStorePath string
}

var clientMessages = endpointMessages{
	privateKeyKind: kindInvalidConfig,
	privateKeyPath: "privateKey path cannot be empty.",
	keyStorePath:   "secureSocket.key.path cannot be empty.",
	trustStorePath: "secureSocket.cert.path cannot be empty.",
}

var listenerMessages = endpointMessages{
	privateKeyKind: kindError,
	privateKeyPath: "Private key path cannot be empty",
	keyStorePath:   "Failed to load FTPS Server Keystore: Path cannot be empty",
	trustStorePath: "Failed to load FTPS Server Truststore: Path cannot be empty",
}

const noPreferredMethod = "No preferred auth method is specified in the SFTP client configurations"

// parseAuth extracts the server identity and validates the auth settings.
func parseAuth(config *values.Map, platform pal.Platform, msgs endpointMessages) (*endpoint, error) {
	protocol, _ := getString(config, "protocol")
	ep := &endpoint{protocol: protocol, userDirIsRoot: getBool(config, "userDirIsRoot")}
	ep.host, _ = getString(config, "host")
	parseCredentials(config, ep)
	port, err := parsePort(config)
	if err != nil {
		return nil, err
	}
	ep.port = port
	ep.preferredMethods = []string{"PUBLICKEY", "PASSWORD"}
	if protocol == protocolFTPS {
		ep.tlsConfig = &pal.TLSConfig{}
	}
	auth := getMap(config, "auth")
	if auth == nil {
		return ep, nil
	}
	if err := validateAuthProtocol(auth, protocol); err != nil {
		return nil, err
	}
	if privateKey := getMap(auth, "privateKey"); privateKey != nil {
		keyPath, _ := getString(privateKey, "path")
		if strings.TrimSpace(keyPath) == "" {
			return nil, newError(msgs.privateKeyKind, msgs.privateKeyPath)
		}
		ep.privateKeyPath = keyPath
		ep.privateKeyPassword, _ = getString(privateKey, "password")
	}
	if protocol == protocolFTPS {
		if err := parseSecureSocket(ep, getMap(auth, "secureSocket"), platform, msgs); err != nil {
			return nil, err
		}
	}
	if protocol == protocolSFTP {
		methods := getStrings(auth, "preferredMethods")
		if len(methods) == 0 {
			return nil, newError(kindError, noPreferredMethod)
		}
		ep.preferredMethods = methods
	}
	return ep, nil
}

// parseTransport validates the timeouts and proxy and extracts the remaining transport options.
func parseTransport(config *values.Map, ep *endpoint) error {
	if err := parseTimeouts(config, ep); err != nil {
		return err
	}
	if err := validateProxy(config); err != nil {
		return err
	}
	if getMap(config, "proxy") != nil {
		return newError(kindError, "Proxy configuration is not yet supported")
	}
	mode, _ := getString(config, "fileTransferMode")
	ep.asciiMode = mode == "ASCII"
	ep.knownHostsPath, _ = getString(config, "sftpSshKnownHosts")
	return nil
}

func parseEndpoint(config *values.Map, platform pal.Platform) (*endpoint, error) {
	ep, err := parseAuth(config, platform, clientMessages)
	if err != nil {
		return nil, err
	}
	if err := parseTransport(config, ep); err != nil {
		return nil, err
	}
	return ep, nil
}
