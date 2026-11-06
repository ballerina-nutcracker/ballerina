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

// A panic or a returned error in a service method is logged to stderr, as in
// jBallerina, without crashing the interpreter. Each onBytes case acks before
// failing, so the client never sends the next message before it is handled.
import ballerina/io;
import ballerina/tcp;

service class FlakyConnService {
    *tcp:ConnectionService;

    remote function onBytes(tcp:Caller caller, readonly & byte[] data) returns tcp:Error? {
        string msg = check 'string:fromBytes(data);
        if msg == "panic" {
            check caller->writeBytes("ack".toBytes());
            panic error("boom in onBytes");
        }
        if msg == "error" {
            check caller->writeBytes("ack".toBytes());
            return error("onBytes failed");
        }
        check caller->writeBytes(data);
    }
}

service class FlakyServer {
    *tcp:Service;

    remote function onConnect(tcp:Caller caller) returns tcp:ConnectionService {
        _ = caller.id;
        return new FlakyConnService();
    }
}

service class PanickingServer {
    *tcp:Service;

    remote function onConnect(tcp:Caller caller) returns tcp:ConnectionService {
        _ = caller.id;
        panic error("boom in onConnect");
    }
}

listener tcp:Listener flakyListener = new (19403, localHost = "127.0.0.1");
listener tcp:Listener panickingListener = new (19404, localHost = "127.0.0.1");

function init() returns error? {
    check flakyListener.attach(new FlakyServer());
    check panickingListener.attach(new PanickingServer());
}

public function testMain() returns error? {
    tcp:Client c = check new ("127.0.0.1", 19403, timeout = 2);
    io:println(check roundTrip(c, "panic")); // @output ack
    io:println(check roundTrip(c, "error")); // @output ack
    io:println(check roundTrip(c, "hello")); // @output hello
    check c->close();

    tcp:Client c2 = check new ("127.0.0.1", 19404, timeout = 2);
    (readonly & byte[])|tcp:Error r = c2->readBytes();
    io:println(r is tcp:Error ? r.message() : "data"); // @output Connection closed by the server.
    check c2->close();
}

function roundTrip(tcp:Client c, string msg) returns string|error {
    check c->writeBytes(msg.toBytes());
    readonly & byte[] response = check c->readBytes();
    return 'string:fromBytes(response);
}
