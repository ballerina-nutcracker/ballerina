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
// Module-level listeners are attached, started and gracefully stopped by the program
// lifecycle; the output does not depend on when the listeners start relative to main.
import ballerina/io;
import ballerina/mqtt;

listener mqtt:Listener moduleListener = new (mqtt:DEFAULT_URL, "module-listener-v-1", "module-listener-v/a");

service on moduleListener {
    remote function onMessage(mqtt:Message message) returns error? {
        io:println("unexpected message on ", message.topic);
    }
}

// The cast avoids an ambiguous module-level list constructor (#1319).
service on new mqtt:Listener(mqtt:DEFAULT_URL, "module-listener-v-2", <mqtt:Subscription[]>[{topic: "module-listener-v/b", qos: 0}]) {
    remote function onMessage(mqtt:Message message, mqtt:Caller caller) returns error? {
        check caller->complete();
        io:println("unexpected message on ", message.topic);
    }

    remote function onError(mqtt:Error err) {
        io:println("unexpected error ", err.message());
    }
}

public function main() {
    io:println("main done"); // @output main done
}
