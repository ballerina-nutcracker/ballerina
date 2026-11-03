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

// Runs against the in-process broker started by corpus/extern/mqtt_test.go.
import ballerina/io;
import ballerina/lang.runtime;
import ballerina/mqtt;

isolated function brokerUrl(string listenerId) returns string = external;

final mqtt:Client c = check new (brokerUrl("anon"), "stream-close-v");

// Isolated so the started strand blocks on a thread of its own instead of the one main runs on.
isolated function readFromOwnStream() returns string|error {
    stream<mqtt:Message, error?> own = check c->receive();
    record {|mqtt:Message value;|}? next = check own.next();
    check own.close();
    if next is () {
        return "end of stream";
    }
    return string:fromBytes(next.value.payload);
}

public function main() returns error? {
    check c->subscribe("stream-close-v/a");
    stream<mqtt:Message, error?> closed = check c->receive();
    future<string|error> reader = start readFromOwnStream();
    runtime:sleep(0.5);

    // Closing a stream does not end the other streams of the client: the strand blocked on its
    // own stream keeps waiting and gets a later message.
    check closed.close();
    runtime:sleep(0.2);
    _ = check c->publish("stream-close-v/a", {payload: "after".toBytes()});
    string got = check wait reader;
    io:println(got); // @output after

    check c->disconnect();
    check c->close();
}
