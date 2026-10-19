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

// Requires an MQTT v5 broker on mqtt:DEFAULT_URL (see corpus/lib/resources/mqtt/compose.yaml).
import ballerina/io;
import ballerina/lang.runtime;
import ballerina/mqtt;

function show(string label, any|error result) {
    if result is error {
        io:println(label, ": ", result.message(), " (mqtt:Error=", result is mqtt:Error, ")");
    } else {
        io:println(label, ": ", result is () ? "ok" : result.toString());
    }
}

function waitUntilConnected(mqtt:Client c) returns boolean|error {
    foreach int _ in 0 ..< 50 {
        boolean connected = check c->isConnected();
        if connected {
            return true;
        }
        runtime:sleep(0.1);
    }
    return false;
}

public function main() returns error? {
    mqtt:Client c = check new (mqtt:DEFAULT_URL, "lifecycle-v-client");

    mqtt:Error? r = c->reconnect();
    show("reconnect while connected", r); // @output reconnect while connected: Client is connected (mqtt:Error=true)
    r = c->close();
    show("close while connected", r); // @output close while connected: Client is connected (mqtt:Error=true)

    r = c->disconnect();
    show("disconnect", r); // @output disconnect: ok
    boolean|mqtt:Error connected = c->isConnected();
    show("isConnected", connected); // @output isConnected: false
    r = c->disconnect();
    show("disconnect again", r); // @output disconnect again: Client is disconnected (mqtt:Error=true)
    mqtt:DeliveryToken|mqtt:Error token = c->publish("lifecycle-v/t", {payload: "x".toBytes()});
    show("publish while disconnected", token); // @output publish while disconnected: Client is not connected (mqtt:Error=true)
    r = c->subscribe("lifecycle-v/t");
    show("subscribe while disconnected", r); // @output subscribe while disconnected: Client is not connected (mqtt:Error=true)

    r = c->reconnect();
    show("reconnect", r); // @output reconnect: ok
    show("reconnected", waitUntilConnected(c)); // @output reconnected: true
    token = c->publish("lifecycle-v/t", {payload: "x".toBytes()});
    show("publish after reconnect", token is mqtt:DeliveryToken); // @output publish after reconnect: true

    r = c->disconnect();
    show("disconnect", r); // @output disconnect: ok
    r = c->close();
    show("close", r); // @output close: ok
    r = c->close();
    show("close again", r); // @output close again: ok
    connected = c->isConnected();
    show("isConnected after close", connected); // @output isConnected after close: false
    r = c->reconnect();
    show("reconnect after close", r); // @output reconnect after close: Client is closed (mqtt:Error=true)
    r = c->disconnect();
    show("disconnect after close", r); // @output disconnect after close: Client is closed (mqtt:Error=true)
    token = c->publish("lifecycle-v/t", {payload: "x".toBytes()});
    show("publish after close", token); // @output publish after close: Client is not connected (mqtt:Error=true)
    stream<mqtt:Message, error?>|mqtt:Error s = c->receive();
    show("receive after close", s is stream<mqtt:Message, error?>); // @output receive after close: true
}
