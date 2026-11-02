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
	"fmt"
	"io/fs"
	"math"
	"net"
	"net/url"
	"strings"
	"time"

	"github.com/ballerina-nutcracker/ballerina/platform/pal"
	"github.com/ballerina-nutcracker/ballerina/values"
)

// Defaults of Paho Java's MqttConnectionOptions, which jBallerina relies on when a
// ConnectionConfiguration field is omitted.
const (
	defaultKeepAlive         = 60
	defaultConnectTimeout    = 30 * time.Second
	defaultMaxReconnectDelay = 128000 * time.Millisecond
	defaultTCPPort           = "1883"
	defaultSSLPort           = "8883"
)

type serverURI struct {
	raw     string
	address string
	secure  bool
}

type outgoingMessage struct {
	payload         []byte
	qos             byte
	retained        bool
	responseTopic   string
	correlationData []byte
}

type willDetails struct {
	topic   string
	message outgoingMessage
}

type connectionOptions struct {
	clientID           string
	serverURIs         []serverURI
	username           *string
	password           []byte
	keepAlive          uint16
	connectTimeout     time.Duration
	cleanStart         bool
	automaticReconnect bool
	maxReconnectDelay  time.Duration
	tlsConfig          *pal.TLSConfig
	will               *willDetails
	manualAcks         bool
}

// parseServerURI mirrors Paho Java's NetworkModuleService URI validation: only tcp:// and
// ssl:// are accepted, with the same default ports.
func parseServerURI(raw string) (serverURI, *mqttError) {
	u, err := url.Parse(raw)
	if err != nil {
		return serverURI{}, newError(fmt.Sprintf("Can't parse string to URI \"%s\"", raw))
	}
	if u.Scheme == "" {
		return serverURI{}, newError(fmt.Sprintf("missing scheme in broker URI: %s", raw))
	}
	var secure bool
	var port string
	switch u.Scheme {
	case "tcp":
		port = defaultTCPPort
	case "ssl":
		secure = true
		port = defaultSSLPort
	case "ws", "wss":
		return serverURI{}, newError(fmt.Sprintf("WebSocket broker URIs are not supported: %s", raw))
	default:
		return serverURI{}, newError(fmt.Sprintf("no NetworkModule installed for scheme \"%s\" of URI \"%s\"", u.Scheme, raw))
	}
	if u.Path != "" {
		return serverURI{}, newError(pathNotEmptyMessage(raw, secure))
	}
	if u.Port() != "" {
		port = u.Port()
	}
	return serverURI{raw: raw, address: net.JoinHostPort(u.Hostname(), port), secure: secure}, nil
}

// pathNotEmptyMessage matches Paho Java, whose SSL module reports only the URI.
func pathNotEmptyMessage(raw string, secure bool) string {
	if secure {
		return raw
	}
	return fmt.Sprintf("URI path must be empty \"%s\"", raw)
}

func parseConnectionOptions(platform pal.Platform, serverURIValue, clientID string, config *values.Map) (*connectionOptions, *mqttError) {
	uri, mErr := parseServerURI(serverURIValue)
	if mErr != nil {
		return nil, mErr
	}
	opts := &connectionOptions{
		clientID:          clientID,
		serverURIs:        []serverURI{uri},
		keepAlive:         defaultKeepAlive,
		connectTimeout:    defaultConnectTimeout,
		cleanStart:        true,
		maxReconnectDelay: defaultMaxReconnectDelay,
	}
	if manualAcks, ok := mapBool(config, "manualAcks"); ok {
		opts.manualAcks = manualAcks
	}
	if connConfig, ok := mapGetMap(config, "connectionConfig"); ok {
		if mErr := opts.applyConnectionConfig(platform, connConfig); mErr != nil {
			return nil, mErr
		}
	}
	if willConfig, ok := mapGetMap(config, "willDetails"); ok {
		will, mErr := parseWillDetails(willConfig)
		if mErr != nil {
			return nil, mErr
		}
		opts.will = will
	}
	return opts, nil
}

func (opts *connectionOptions) applyConnectionConfig(platform pal.Platform, connConfig *values.Map) *mqttError {
	if username, ok := mapString(connConfig, "username"); ok {
		opts.username = &username
	}
	if password, ok := mapString(connConfig, "password"); ok {
		opts.password = []byte(password)
	}
	if delay, ok := mapInt(connConfig, "maxReconnectDelay"); ok {
		opts.maxReconnectDelay = time.Duration(delay) * time.Millisecond
	}
	if keepAlive, ok := mapInt(connConfig, "keepAliveInterval"); ok {
		if keepAlive < 0 || keepAlive > math.MaxUint16 {
			return newError(fmt.Sprintf("Invalid keep alive interval: %d. Allowed values are 0 to %d", keepAlive, math.MaxUint16))
		}
		opts.keepAlive = uint16(keepAlive)
	}
	if timeout, ok := mapInt(connConfig, "connectionTimeout"); ok {
		if timeout < 0 {
			return newError(fmt.Sprintf("Invalid connection timeout: %d. The value must not be negative", timeout))
		}
		opts.connectTimeout = time.Duration(timeout) * time.Second
	}
	if cleanStart, ok := mapBool(connConfig, "cleanStart"); ok {
		opts.cleanStart = cleanStart
	}
	if uris, ok := mapGetList(connConfig, "serverUris"); ok && uris.Len() > 0 {
		parsed := make([]serverURI, 0, uris.Len())
		for i := 0; i < uris.Len(); i++ {
			raw, _ := uris.Get(i).(string)
			uri, mErr := parseServerURI(raw)
			if mErr != nil {
				return mErr
			}
			parsed = append(parsed, uri)
		}
		opts.serverURIs = parsed
	}
	if automaticReconnect, ok := mapBool(connConfig, "automaticReconnect"); ok {
		opts.automaticReconnect = automaticReconnect
	}
	if secureSocket, ok := mapGetMap(connConfig, "secureSocket"); ok {
		tlsConfig, mErr := parseSecureSocket(platform, secureSocket)
		if mErr != nil {
			return mErr
		}
		opts.tlsConfig = tlsConfig
	}
	return nil
}

func parseWillDetails(willConfig *values.Map) (*willDetails, *mqttError) {
	topic, _ := mapString(willConfig, "destinationTopic")
	messageMap, _ := mapGetMap(willConfig, "willMessage")
	message, mErr := parseOutgoingMessage(messageMap)
	if mErr != nil {
		return nil, mErr
	}
	if mErr := validateTopic(topic, false); mErr != nil {
		return nil, mErr
	}
	return &willDetails{topic: topic, message: message}, nil
}

func parseOutgoingMessage(message *values.Map) (outgoingMessage, *mqttError) {
	qos, _ := mapInt(message, "qos")
	if mErr := validateQoS(qos); mErr != nil {
		return outgoingMessage{}, mErr
	}
	out := outgoingMessage{qos: byte(qos)}
	if payload, ok := mapGetList(message, "payload"); ok {
		out.payload = payload.ToByteSlice()
	}
	out.retained, _ = mapBool(message, "retained")
	if properties, ok := mapGetMap(message, "properties"); ok {
		out.responseTopic, _ = mapString(properties, "responseTopic")
		if correlationData, ok := mapGetList(properties, "correlationData"); ok {
			out.correlationData = correlationData.ToByteSlice()
		}
	}
	return out, nil
}

// parseSecureSocket builds the PAL TLS settings. The context protocol defaults to TLSv1.2,
// matching the SSLContext jBallerina creates when no protocol is configured.
func parseSecureSocket(platform pal.Platform, secureSocket *values.Map) (*pal.TLSConfig, *mqttError) {
	tlsConfig := &pal.TLSConfig{MaxVersion: tls.VersionTLS12}
	if cert, ok := secureSocket.Get("cert"); ok {
		certPath, isPath := cert.(string)
		if !isPath {
			return nil, newError("crypto:TrustStore is not supported as the mqtt:SecureSocket cert; provide a PEM certificate file path")
		}
		certPEM, mErr := readCertificate(platform, certPath)
		if mErr != nil {
			return nil, mErr
		}
		tlsConfig.CACertPEM = certPEM
	}
	if key, ok := mapGetMap(secureSocket, "key"); ok {
		certFile, isCertKey := mapString(key, "certFile")
		if !isCertKey {
			return nil, newError("crypto:KeyStore is not supported as the mqtt:SecureSocket key; provide an mqtt:CertKey")
		}
		certPEM, mErr := readCertificate(platform, certFile)
		if mErr != nil {
			return nil, mErr
		}
		keyFile, _ := mapString(key, "keyFile")
		keyPEM, mErr := readPrivateKey(platform, keyFile)
		if mErr != nil {
			return nil, mErr
		}
		tlsConfig.ClientCertPEM = certPEM
		tlsConfig.ClientKeyPEM = keyPEM
	}
	if protocol, ok := mapGetMap(secureSocket, "protocol"); ok {
		version, _ := mapString(protocol, "version")
		if v, known := tlsVersions[version]; known {
			tlsConfig.MaxVersion = v
			tlsConfig.MinVersion = min(v, tls.VersionTLS12)
		}
	}
	return tlsConfig, nil
}

var tlsVersions = map[string]uint16{
	"1.0": tls.VersionTLS10,
	"1.1": tls.VersionTLS11,
	"1.2": tls.VersionTLS12,
	"1.3": tls.VersionTLS13,
}

func readCertificate(platform pal.Platform, path string) ([]byte, *mqttError) {
	const prefix = "Failed to get the public key from Crypto API. "
	data, err := platform.FS.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, newError(prefix + "Certificate file not found at: " + path)
		}
		return nil, newError(prefix + err.Error())
	}
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, newError(prefix + "Unable to do public key operations: no PEM certificate found in " + path)
	}
	if _, err := x509.ParseCertificate(block.Bytes); err != nil {
		return nil, newError(prefix + "Unable to do public key operations: " + err.Error())
	}
	return data, nil
}

func readPrivateKey(platform pal.Platform, path string) ([]byte, *mqttError) {
	const prefix = "Failed to get the private key from Crypto API. "
	data, err := platform.FS.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, newError(prefix + "Key file not found at: " + path)
		}
		return nil, newError(prefix + err.Error())
	}
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, newError(prefix + "Failed to parse private key information from the given input")
	}
	if block.Type == "ENCRYPTED PRIVATE KEY" || strings.Contains(block.Headers["Proc-Type"], "ENCRYPTED") {
		return nil, newError(prefix + "Encrypted private keys are not supported: " + path)
	}
	if !isParsablePrivateKey(block.Bytes) {
		return nil, newError(prefix + "Failed to parse private key information from the given input")
	}
	return data, nil
}

func isParsablePrivateKey(der []byte) bool {
	if _, err := x509.ParsePKCS8PrivateKey(der); err == nil {
		return true
	}
	if _, err := x509.ParsePKCS1PrivateKey(der); err == nil {
		return true
	}
	_, err := x509.ParseECPrivateKey(der)
	return err == nil
}

func mapGetMap(m *values.Map, key string) (*values.Map, bool) {
	if m == nil {
		return nil, false
	}
	v, ok := m.Get(key)
	if !ok {
		return nil, false
	}
	mv, ok := v.(*values.Map)
	return mv, ok
}

func mapGetList(m *values.Map, key string) (*values.List, bool) {
	if m == nil {
		return nil, false
	}
	v, ok := m.Get(key)
	if !ok {
		return nil, false
	}
	lv, ok := v.(*values.List)
	return lv, ok
}

func mapString(m *values.Map, key string) (string, bool) {
	if m == nil {
		return "", false
	}
	v, ok := m.Get(key)
	if !ok {
		return "", false
	}
	s, ok := v.(string)
	return s, ok
}

func mapInt(m *values.Map, key string) (int64, bool) {
	if m == nil {
		return 0, false
	}
	v, ok := m.Get(key)
	if !ok {
		return 0, false
	}
	i, ok := v.(int64)
	return i, ok
}

func mapBool(m *values.Map, key string) (bool, bool) {
	if m == nil {
		return false, false
	}
	v, ok := m.Get(key)
	if !ok {
		return false, false
	}
	b, ok := v.(bool)
	return b, ok
}
