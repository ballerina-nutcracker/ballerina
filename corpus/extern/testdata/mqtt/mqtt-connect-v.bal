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

// Runs against the in-process broker started by corpus/extern/mqtt_test.go, which has an
// anonymous and a password-protected listener.
import ballerina/io;
import ballerina/mqtt;

isolated function brokerUrl(string listenerId) returns string = external;

final string BROKER_URL = brokerUrl("anon");
final string PASSWORD_URL = brokerUrl("password");

function connect(string label, string url, string clientId, *mqtt:ClientConfiguration config) returns error? {
    mqtt:Client|mqtt:Error c = new (url, clientId, config);
    if c is mqtt:Error {
        io:println(label, ": ", c.message(), " (mqtt:Error=", c is mqtt:Error, ")");
        return;
    }
    io:println(label, ": connected");
    check c->disconnect();
    check c->close();
}

// The CONNACK reason code of a rejected login depends on the broker (Mosquitto sends 135, Not
// authorized; mochi-mqtt sends 134, Bad User Name or Password), so either message is accepted.
function login(string label, string clientId, *mqtt:ConnectionConfiguration connectionConfig) returns error? {
    mqtt:Client|mqtt:Error c = new (PASSWORD_URL, clientId, connectionConfig = connectionConfig);
    if c is mqtt:Error {
        boolean loginRejected = c.message() == "Not authorized." || c.message() == "Bad User Name or Password.";
        io:println(label, ": login rejected=", loginRejected, " (mqtt:Error=", c is mqtt:Error, ")");
        return;
    }
    io:println(label, ": connected");
    check c->disconnect();
    check c->close();
}

function listen(string label, string url, string clientId) returns error? {
    mqtt:Listener|mqtt:Error l = new (url, clientId, "connect-v/t");
    if l is mqtt:Error {
        io:println(label, ": ", l.message());
        return;
    }
    io:println(label, ": connected");
    check l.gracefulStop();
}

public function main() returns error? {
    check connect("refused", "tcp://localhost:1", "connect-v-1"); // @output refused: Unable to connect to server (mqtt:Error=true)
    // @output http scheme: no NetworkModule installed for scheme "http" of URI "http://localhost:1883" (mqtt:Error=true)
    check connect("http scheme", "http://localhost:1883", "connect-v-2");
    // @output no scheme: no NetworkModule installed for scheme "localhost" of URI "localhost:1883" (mqtt:Error=true)
    check connect("no scheme", "localhost:1883", "connect-v-3");
    // @output ws scheme: WebSocket broker URIs are not supported: ws://localhost:1883 (mqtt:Error=true)
    check connect("ws scheme", "ws://localhost:1883", "connect-v-4");
    // @output unparsable: Can't parse string to URI "127.0.0.1:1883" (mqtt:Error=true)
    check connect("unparsable", "127.0.0.1:1883", "connect-v-4a");
    // @output missing scheme: missing scheme in broker URI: //localhost:1883 (mqtt:Error=true)
    check connect("missing scheme", "//localhost:1883", "connect-v-4b");

    check login("no credentials", "connect-v-6"); // @output no credentials: login rejected=true (mqtt:Error=true)
    // @output wrong credentials: login rejected=true (mqtt:Error=true)
    check login("wrong credentials", "connect-v-7", username = "mqttuser", password = "password");
    check login("credentials", "connect-v-8", username = "ballerina", password = "ballerinamqtt"); // @output credentials: connected

    // serverUris replace the constructor URI and are tried in order.
    // @output serverUris: connected
    check connect("serverUris", "tcp://localhost:1", "connect-v-9", connectionConfig = {serverUris: ["tcp://localhost:2", BROKER_URL]});
    // @output serverUris all down: Unable to connect to server (mqtt:Error=true)
    check connect("serverUris all down", BROKER_URL, "connect-v-10", connectionConfig = {serverUris: ["tcp://localhost:1", "tcp://localhost:2"]});
    // @output invalid serverUris: no NetworkModule installed for scheme "http" of URI "http://localhost:1883" (mqtt:Error=true)
    check connect("invalid serverUris", BROKER_URL, "connect-v-11", connectionConfig = {serverUris: ["http://localhost:1883"]});
    // @output empty serverUris: connected
    check connect("empty serverUris", BROKER_URL, "connect-v-17", connectionConfig = {serverUris: []});

    check connect("empty client id", BROKER_URL, ""); // @output empty client id: connected
    // @output options: connected
    check connect("options", BROKER_URL, "connect-v-12", connectionConfig = {
        keepAliveInterval: 10,
        connectionTimeout: 5,
        cleanStart: false,
        automaticReconnect: true,
        maxReconnectDelay: 2000
    });
    // @output will: connected
    check connect("will", BROKER_URL, "connect-v-13", willDetails = {
        destinationTopic: "connect-v/will",
        willMessage: {payload: "bye".toBytes(), qos: 1, retained: false}
    });

    check listen("listener refused", "tcp://localhost:1", "connect-v-14"); // @output listener refused: Unable to connect to server
    // @output listener http scheme: no NetworkModule installed for scheme "http" of URI "http://localhost:1"
    check listen("listener http scheme", "http://localhost:1", "connect-v-15");
    check listen("listener", BROKER_URL, "connect-v-16"); // @output listener: connected
}
