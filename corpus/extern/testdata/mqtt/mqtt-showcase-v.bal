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

final string BROKER_URL = brokerUrl("anon");

const REQ_TOPIC = "showcase/request";
const RESP_TOPIC = "showcase/response";

isolated service class Responder {
    *mqtt:Service;

    remote function onMessage(mqtt:Message message, mqtt:Caller caller) returns error? {
        string reply = string `pong to ${check string:fromBytes(message.payload)} on ${message.topic ?: "?"}`;
        check caller->respond({payload: reply.toBytes()});
        check caller->complete();
    }

    remote function onError(mqtt:Error err) {
        io:println("server error: ", err.message());
    }
}

public function main() returns error? {
    mqtt:Listener lsnr = check new (BROKER_URL, "showcase-listener", {topic: REQ_TOPIC, qos: 1}, manualAcks = true);
    check lsnr.attach(new Responder());
    check lsnr.'start();

    mqtt:Client c = check new (BROKER_URL, "showcase-client", connectionConfig = {keepAliveInterval: 30, cleanStart: true});
    boolean connected = check c->isConnected();
    io:println("connected: ", connected); // @output connected: true

    check c->subscribe([RESP_TOPIC]);
    stream<mqtt:Message, error?> responses = check c->receive();

    mqtt:DeliveryToken token = check c->publish(REQ_TOPIC, {
        payload: "ping".toBytes(),
        qos: 1,
        properties: {responseTopic: RESP_TOPIC, correlationData: "id-1".toBytes()}
    });
    io:println("delivered to: ", token.topic); // @output delivered to: showcase/request

    record {|mqtt:Message value;|}? resp = check responses.next();
    if resp is record {|mqtt:Message value;|} {
        mqtt:Message m = resp.value;
        // @output client got: pong to ping on showcase/request qos=1 retained=false
        io:println("client got: ", check string:fromBytes(m.payload), " qos=", m.qos, " retained=", m.retained);
        byte[]? corr = m.properties?.correlationData;
        io:println("correlation: ", corr is byte[] ? check string:fromBytes(corr) : "none"); // @output correlation: id-1
    }
    check responses.close();
    record {|mqtt:Message value;|}|error? closed = responses.next();
    // @output after close: Stream is closed. Therefore, no operations are allowed further on the stream.
    io:println("after close: ", closed is error ? closed.message() : "no error");

    check c->disconnect();
    connected = check c->isConnected();
    io:println("connected: ", connected); // @output connected: false
    check c->reconnect();
    // reconnect() only starts the reconnect; poll until it completes.
    foreach int _ in 0 ..< 50 {
        connected = check c->isConnected();
        if connected {
            break;
        }
        runtime:sleep(0.1);
    }
    io:println("reconnected: ", connected); // @output reconnected: true
    mqtt:Error? closeErr = c->close();
    io:println("close while connected: ", closeErr is mqtt:Error ? closeErr.message() : "ok"); // @output close while connected: Client is connected
    check c->disconnect();
    check c->close();

    check lsnr.gracefulStop();

    mqtt:Client|mqtt:Error bad = new ("tcp://localhost:1", "showcase-bad");
    if bad is mqtt:Error {
        io:println("connect failed: ", bad.message()); // @output connect failed: Unable to connect to server
    }
}
