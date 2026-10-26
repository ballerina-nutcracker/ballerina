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

package extern_test

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	mochi "github.com/mochi-mqtt/server/v2"
	"github.com/mochi-mqtt/server/v2/listeners"
	"github.com/mochi-mqtt/server/v2/packets"
)

const (
	mqttCertsDir     = "testdata/mqtt/certs"
	mqttUsername     = "ballerina"
	mqttUserPassword = "ballerinamqtt"
)

// mqttListenerSpec mirrors one listener of module-ballerina-mqtt's Mosquitto test broker.
type mqttListenerSpec struct {
	id           string
	scheme       string
	clientAuth   tls.ClientAuthType
	requireLogin bool
}

var mqttListenerSpecs = []mqttListenerSpec{
	{id: "anon", scheme: "tcp"},
	{id: "password", scheme: "tcp", requireLogin: true},
	{id: "tls", scheme: "ssl", clientAuth: tls.VerifyClientCertIfGiven},
	{id: "mtls", scheme: "ssl", clientAuth: tls.RequireAndVerifyClientCert},
	{id: "mtls-password", scheme: "ssl", clientAuth: tls.RequireAndVerifyClientCert, requireLogin: true},
}

// startMqttBroker starts an in-process MQTT v5 broker with one loopback listener per
// mqttListenerSpecs entry on an ephemeral port, and returns each listener's broker URI by id.
func startMqttBroker(t *testing.T) map[string]string {
	t.Helper()
	server := mochi.New(&mochi.Options{Logger: slog.New(slog.NewTextHandler(io.Discard, nil))})
	t.Cleanup(func() { closeMqttBroker(server) })
	loginRequired := map[string]bool{}
	for _, spec := range mqttListenerSpecs {
		loginRequired[spec.id] = spec.requireLogin
	}
	if err := server.AddHook(&mqttBrokerHook{loginRequired: loginRequired}, nil); err != nil {
		t.Fatalf("adding mqtt broker hook: %v", err)
	}

	urls := map[string]string{}
	for _, spec := range mqttListenerSpecs {
		config := listeners.Config{ID: spec.id, Address: "127.0.0.1:0"}
		if spec.scheme == "ssl" {
			config.TLSConfig = mqttServerTLSConfig(t, spec.clientAuth)
		}
		listener := listeners.NewTCP(config)
		if err := server.AddListener(listener); err != nil {
			t.Fatalf("adding mqtt listener %s: %v", spec.id, err)
		}
		urls[spec.id] = spec.scheme + "://" + listener.Address()
	}
	if err := server.Serve(); err != nil {
		t.Fatalf("starting mqtt broker: %v", err)
	}
	return urls
}

// closeMqttBroker bounds the broker shutdown: mochi v2.7.9 can deadlock in Close when a client
// attaches while it closes, as GetByListener re-takes a read lock a waiting writer blocks.
func closeMqttBroker(server *mochi.Server) {
	done := make(chan struct{})
	go func() {
		_ = server.Close()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
	}
}

func mqttServerTLSConfig(t *testing.T, clientAuth tls.ClientAuthType) *tls.Config {
	t.Helper()
	serverCert, err := tls.LoadX509KeyPair(filepath.Join(mqttCertsDir, "server.crt"), filepath.Join(mqttCertsDir, "server.key"))
	if err != nil {
		t.Fatalf("loading mqtt server certificate: %v", err)
	}
	clientCA, err := os.ReadFile(filepath.Join(mqttCertsDir, "client.crt"))
	if err != nil {
		t.Fatalf("reading mqtt client certificate: %v", err)
	}
	clientCAs := x509.NewCertPool()
	if !clientCAs.AppendCertsFromPEM(clientCA) {
		t.Fatalf("parsing mqtt client certificate")
	}
	return &tls.Config{
		Certificates: []tls.Certificate{serverCert},
		ClientAuth:   clientAuth,
		ClientCAs:    clientCAs,
		MinVersion:   tls.VersionTLS12,
	}
}

// mqttBrokerHook admits every client on anonymous listeners and only the test user on
// listeners that require a login, like Mosquitto's per_listener_settings password_file.
type mqttBrokerHook struct {
	mochi.HookBase
	loginRequired map[string]bool
}

func (h *mqttBrokerHook) ID() string {
	return "mqtt-test-broker"
}

func (h *mqttBrokerHook) Provides(b byte) bool {
	return b == mochi.OnConnectAuthenticate || b == mochi.OnACLCheck || b == mochi.OnDisconnect
}

func (h *mqttBrokerHook) OnConnectAuthenticate(cl *mochi.Client, pk packets.Packet) bool {
	if !h.loginRequired[cl.Net.Listener] {
		return true
	}
	return string(pk.Connect.Username) == mqttUsername && bytes.Equal(pk.Connect.Password, []byte(mqttUserPassword))
}

func (h *mqttBrokerHook) OnACLCheck(*mochi.Client, string, bool) bool {
	return true
}

// OnDisconnect holds back a taken-over client until mochi marks it as taken over. Otherwise
// mochi can remove the client ID from its registry after the new client registered it, and a
// later take-over of that client ID no longer disconnects the new client.
func (h *mqttBrokerHook) OnDisconnect(cl *mochi.Client, _ error, _ bool) {
	if !errors.Is(cl.StopCause(), packets.ErrSessionTakenOver) {
		return
	}
	for deadline := time.Now().Add(5 * time.Second); !cl.IsTakenOver() && time.Now().Before(deadline); {
		time.Sleep(time.Millisecond)
	}
}
