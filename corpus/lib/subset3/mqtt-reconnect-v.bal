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
// The broker drops a connection when another client takes over its client ID or when the
// client sends an empty SUBSCRIBE.
import ballerina/io;
import ballerina/lang.runtime;
import ballerina/mqtt;

function waitFor(mqtt:Client c, boolean expected) returns boolean|error {
    foreach int _ in 0 ..< 100 {
        boolean connected = check c->isConnected();
        if connected == expected {
            return connected;
        }
        runtime:sleep(0.1);
    }
    return !expected;
}

public function main() returns error? {
    // The taken-over client reconnects automatically and takes the client ID back.
    mqtt:Client resilient = check new (mqtt:DEFAULT_URL, "reconnect-v-shared",
        connectionConfig = {automaticReconnect: true, maxReconnectDelay: 1000});
    mqtt:Client intruder = check new (mqtt:DEFAULT_URL, "reconnect-v-shared");
    io:println("intruder dropped: ", !check waitFor(intruder, false)); // @output intruder dropped: true
    io:println("resilient connected: ", check waitFor(resilient, true)); // @output resilient connected: true
    mqtt:Error? r = intruder->disconnect();
    io:println("intruder disconnect: ", r is mqtt:Error ? r.message() : "ok"); // @output intruder disconnect: Client is disconnected
    check intruder->close();
    check resilient->disconnect();
    check resilient->close();

    // An empty SUBSCRIBE is a protocol violation; the client sends it like Paho Java does and
    // reports the server-initiated disconnect.
    mqtt:Client c = check new (mqtt:DEFAULT_URL, "reconnect-v-empty");
    string[] none = [];
    r = c->subscribe(none);
    // @output empty subscribe: The Server Disconnected the client. Disconnect RC: 129
    io:println("empty subscribe: ", r is mqtt:Error ? r.message() : "ok");
    io:println("connected: ", check waitFor(c, false)); // @output connected: false
    r = c->reconnect();
    io:println("reconnect: ", r is mqtt:Error ? r.message() : "ok"); // @output reconnect: ok
    io:println("reconnected: ", check waitFor(c, true)); // @output reconnected: true
    check c->disconnect();
    check c->close();
}
