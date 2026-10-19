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
// Invalid QoS values and topics are returned as mqtt:Error (jBallerina panics instead; see
// https://github.com/ballerina-platform/ballerina-library/issues/9328).
import ballerina/io;
import ballerina/mqtt;

function show(string label, any|error result) {
    if result is error {
        io:println(label, ": ", result.message(), " (mqtt:Error=", result is mqtt:Error, ")");
    } else {
        io:println(label, ": ok");
    }
}

isolated service class Svc {
    *mqtt:Service;

    remote function onMessage(mqtt:Message message) {
        io:println(message.topic);
    }
}

public function main() returns error? {
    mqtt:Client c = check new (mqtt:DEFAULT_URL, "validation-v-client");
    byte[] payload = "x".toBytes();

    mqtt:DeliveryToken|mqtt:Error token = c->publish("validation-v/t", {payload, qos: 3});
    show("qos 3", token); // @output qos 3: Invalid QoS value: 3. Allowed values are 0, 1 and 2 (mqtt:Error=true)
    token = c->publish("validation-v/t", {payload, qos: -1});
    show("qos -1", token); // @output qos -1: Invalid QoS value: -1. Allowed values are 0, 1 and 2 (mqtt:Error=true)
    token = c->publish("validation-v/#", {payload});
    show("wildcard topic", token); // @output wildcard topic: The topic name MUST NOT contain any wildcard characters (#+) (mqtt:Error=true)
    token = c->publish("validation-v/+", {payload});
    show("single-level wildcard topic", token); // @output single-level wildcard topic: The topic name MUST NOT contain any wildcard characters (#+) (mqtt:Error=true)
    token = c->publish("", {payload});
    show("empty topic", token); // @output empty topic: Invalid topic length, should be in range[1, 65535]! (mqtt:Error=true)

    mqtt:Error? r = c->subscribe("validation-v/#/x");
    show("misplaced #", r); // @output misplaced #: Invalid usage of multi-level wildcard in topic string: validation-v/#/x (mqtt:Error=true)
    r = c->subscribe("validation-v#");
    show("# not a level", r); // @output # not a level: Invalid usage of multi-level wildcard in topic string: validation-v# (mqtt:Error=true)
    r = c->subscribe("validation-v/a+");
    show("+ not a level", r); // @output + not a level: Invalid usage of single-level wildcard in topic string 'validation-v/a+'! (mqtt:Error=true)
    r = c->subscribe({topic: "validation-v/t", qos: 3});
    show("subscription qos 3", r); // @output subscription qos 3: Invalid QoS value: 3. Allowed values are 0, 1 and 2 (mqtt:Error=true)
    r = c->subscribe(["#", "+", "validation-v/+/x/#"]);
    show("valid filters", r); // @output valid filters: ok

    mqtt:Client|mqtt:Error willClient = new (mqtt:DEFAULT_URL, "validation-v-will", willDetails = {
        destinationTopic: "validation-v/will/#",
        willMessage: {payload}
    });
    show("will topic", willClient); // @output will topic: The topic name MUST NOT contain any wildcard characters (#+) (mqtt:Error=true)
    willClient = new (mqtt:DEFAULT_URL, "validation-v-will", willDetails = {
        destinationTopic: "validation-v/will",
        willMessage: {payload, qos: 5}
    });
    show("will qos", willClient); // @output will qos: Invalid QoS value: 5. Allowed values are 0, 1 and 2 (mqtt:Error=true)

    mqtt:Listener|mqtt:Error lsnr = new (mqtt:DEFAULT_URL, "validation-v-listener", "validation-v/a/#/b");
    if lsnr is mqtt:Listener {
        check lsnr.attach(new Svc());
        r = lsnr.'start();
        show("listener start", r); // @output listener start: Invalid usage of multi-level wildcard in topic string: validation-v/a/#/b (mqtt:Error=true)
        check lsnr.gracefulStop();
    }

    check c->disconnect();
    check c->close();
}
