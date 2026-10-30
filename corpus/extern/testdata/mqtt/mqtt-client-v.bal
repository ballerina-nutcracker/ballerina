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
import ballerina/mqtt;

isolated function brokerUrl(string listenerId) returns string = external;

final string BROKER_URL = brokerUrl("anon");

function printNext(stream<mqtt:Message, error?> s) returns error? {
    record {|mqtt:Message value;|}? next = check s.next();
    if next is () {
        io:println("end of stream");
        return;
    }
    mqtt:Message m = next.value;
    io:println(m.topic, " ", check string:fromBytes(m.payload), " qos=", m.qos, " retained=", m.retained,
            " duplicate=", m.duplicate, " id=", m.messageId, " properties=", m.properties);
}

function readOne(stream<mqtt:Message, error?> s) returns string|error {
    record {|mqtt:Message value;|}? next = check s.next();
    if next is () {
        return "end of stream";
    }
    return string:fromBytes(next.value.payload);
}

public function main() returns error? {
    mqtt:Client c = check new (BROKER_URL, "client-v-main");

    // Every subscription shape: string, string[], Subscription and Subscription[].
    check c->subscribe("client-v/a");
    check c->subscribe(["client-v/b", "client-v/c"]);
    check c->subscribe({topic: "client-v/d", qos: 2});
    // A typed variable works around https://github.com/ballerina-nutcracker/ballerina/issues/1340.
    mqtt:Subscription[] subscriptions = [{topic: "client-v/e", qos: 0}, {topic: "client-v/wild/+/x"}];
    check c->subscribe(subscriptions);
    stream<mqtt:Message, error?> s = check c->receive();

    // Packet identifiers are shared by SUBSCRIBE and PUBLISH, so four subscribes used ids 1-4.
    mqtt:DeliveryToken t1 = check c->publish("client-v/a", {payload: "one".toBytes()});
    io:println(t1); // @output {"messageId":5,"topic":"client-v/a"}
    mqtt:DeliveryToken t2 = check c->publish("client-v/d", {payload: "two".toBytes(), qos: 2});
    io:println(t2); // @output {"messageId":6,"topic":"client-v/d"}
    _ = check c->publish("client-v/e", {payload: "three".toBytes(), qos: 0});

    // @output client-v/a one qos=1 retained=false duplicate=false id=1 properties={}
    check printNext(s);
    // @output client-v/d two qos=2 retained=false duplicate=false id=2 properties={}
    check printNext(s);
    // @output client-v/e three qos=0 retained=false duplicate=false id=0 properties={}
    check printNext(s);

    // The received QoS is capped by the subscription QoS (client-v/b defaults to 1).
    _ = check c->publish("client-v/b", {payload: "capped".toBytes(), qos: 2,
        properties: {responseTopic: "client-v/reply", correlationData: [1, 2]}});
    // @output client-v/b capped qos=1 retained=false duplicate=false id=3 properties={"responseTopic":"client-v/reply","correlationData":[1,2]}
    check printNext(s);

    _ = check c->publish("client-v/wild/1/x", {payload: "wild".toBytes()});
    // @output client-v/wild/1/x wild qos=1 retained=false duplicate=false id=4 properties={}
    check printNext(s);

    // A retained message is delivered to a later subscription with retained=true.
    _ = check c->publish("client-v/retained", {payload: "kept".toBytes(), retained: true});
    check c->subscribe("client-v/retained");
    // @output client-v/retained kept qos=1 retained=true duplicate=false id=5 properties={}
    check printNext(s);
    _ = check c->publish("client-v/retained", {payload: [], retained: true});
    // @output client-v/retained  qos=1 retained=false duplicate=false id=6 properties={}
    check printNext(s);

    // Every receive() stream of a client shares one queue.
    stream<mqtt:Message, error?> other = check c->receive();
    _ = check c->publish("client-v/c", {payload: "shared".toBytes()});
    io:println(check readOne(other)); // @output shared
    check other.close();
    error? closeAgain = other.close();
    // @output Stream is closed. Therefore, no operations are allowed further on the stream.
    io:println(closeAgain is error ? closeAgain.message() : "closed");
    _ = check c->publish("client-v/c", {payload: "still open".toBytes()});
    io:println(check readOne(s)); // @output still open

    // A stream can be consumed from another strand.
    future<string|error> reader = start readOne(s);
    _ = check c->publish("client-v/c", {payload: "from strand".toBytes()});
    string fromStrand = check wait reader;
    io:println(fromStrand); // @output from strand

    check s.close();
    check c->disconnect();
    check c->close();
}
