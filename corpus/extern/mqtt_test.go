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
	"fmt"
	"testing"

	"github.com/ballerina-nutcracker/ballerina/platform/pal"
	"github.com/ballerina-nutcracker/ballerina/platform/palnative"
	"github.com/ballerina-nutcracker/ballerina/runtime/extern"
	"github.com/ballerina-nutcracker/ballerina/test_util/testharness"
	"github.com/ballerina-nutcracker/ballerina/values"
)

// mqttPal wraps the in-memory TestPal with real TCP dialing so the mqtt fixtures can reach the
// in-process broker started per test.
type mqttPal struct {
	testharness.TestPal
}

func newMqttPal() *mqttPal {
	return &mqttPal{TestPal: testharness.NewTestPal()}
}

func (p *mqttPal) Platform() pal.Platform {
	base := p.TestPal.Platform()
	base.Net = pal.Net{Dial: palnative.Dial}
	return base
}

// runMqtt runs testdata/mqtt/<name>.bal against a fresh broker; the fixture gets the URI of
// each broker listener from its external brokerUrl function.
func runMqtt(t *testing.T, name string) {
	t.Helper()
	t.Parallel()
	skipIfNoLoopback(t)
	urls := startMqttBroker(t)
	externs := []testharness.ExternRegistration{{
		Org: "$anon", Module: name, FuncName: "brokerUrl",
		Impl: func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
			id := values.String(args[0], nil)
			url, ok := urls[id]
			if !ok {
				return nil, fmt.Errorf("unknown mqtt broker listener %q", id)
			}
			return url, nil
		},
	}}
	runExtern(t, fileCase("mqtt/"+name), newMqttPal(), externs)
}

func TestMqttClient(t *testing.T) {
	runMqtt(t, "mqtt-client-v")
}

func TestMqttConnect(t *testing.T) {
	runMqtt(t, "mqtt-connect-v")
}

func TestMqttLifecycle(t *testing.T) {
	runMqtt(t, "mqtt-lifecycle-v")
}

func TestMqttListener(t *testing.T) {
	runMqtt(t, "mqtt-listener-v")
}

func TestMqttModuleListener(t *testing.T) {
	runMqtt(t, "mqtt-module-listener-v")
}

func TestMqttReconnect(t *testing.T) {
	runMqtt(t, "mqtt-reconnect-v")
}

func TestMqttShowcase(t *testing.T) {
	runMqtt(t, "mqtt-showcase-v")
}

func TestMqttTLS(t *testing.T) {
	runMqtt(t, "mqtt-tls-v")
}

func TestMqttValidation(t *testing.T) {
	runMqtt(t, "mqtt-validation-v")
}
