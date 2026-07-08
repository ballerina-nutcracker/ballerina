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

// Anonymous `service on new tcp:Listener(...) { ... }` bodies cannot target
// a `distinct service object` type in this interpreter yet — see the
// README's Notable Behavioural Changes. Named service classes with explicit
// `*tcp:Service;`/`*tcp:ConnectionService;` inclusion work fine and are used
// throughout this stdlib's tests instead.
import ballerina/io;
import ballerina/tcp;

service class EchoService {
    *tcp:ConnectionService;

    remote function onBytes(tcp:Caller caller, readonly & byte[] data) returns tcp:Error? {
        check caller->writeBytes(data);
    }

    remote function onClose() returns tcp:Error? {
        // Fires asynchronously relative to the client's own close() call, so
        // it is not asserted on here to keep this fixture's output
        // deterministic — see tcp-svc-close-once-v.bal for that assertion.
    }
}

service class EchoServer {
    *tcp:Service;

    remote function onConnect(tcp:Caller caller) returns tcp:ConnectionService {
        _ = caller.id;
        io:println("onConnect"); // @output onConnect
        return new EchoService();
    }
}

listener tcp:Listener echoListener = new (19390);

function init() returns error? {
    check echoListener.attach(new EchoServer());
}

// testMain is invoked by the harness while the runtime is parked in the
// listening state. It drives the live service over a real tcp:Client.
public function testMain() returns error? {
    tcp:Client c = check new ("localhost", 19390, {});
    check c->writeBytes("hello".toBytes());
    readonly & byte[] echoed = check c->readBytes();
    io:println('string:fromBytes(echoed)); // @output hello
    check c->close();
}
