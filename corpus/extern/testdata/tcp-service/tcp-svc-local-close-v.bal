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

// A connection service that closes its own Caller from onBytes gets onClose
// but no onError, matching jBallerina. onClose runs before the socket closes,
// so it prints before the client's read observes the close.
import ballerina/io;
import ballerina/tcp;

service class ClosingConnService {
    *tcp:ConnectionService;

    remote function onBytes(tcp:Caller caller, readonly & byte[] data) returns tcp:Error? {
        io:println("onBytes ", check 'string:fromBytes(data)); // @output onBytes hello
        check caller->close();
        check caller->close();
    }

    remote function onError(readonly & tcp:Error err) {
        io:println("onError ", err.message());
    }

    remote function onClose() {
        io:println("onClose"); // @output onClose
    }
}

service class ClosingServer {
    *tcp:Service;

    remote function onConnect(tcp:Caller caller) returns tcp:ConnectionService {
        _ = caller.id;
        return new ClosingConnService();
    }
}

listener tcp:Listener closingListener = new (19400, localHost = "127.0.0.1");

function init() returns error? {
    check closingListener.attach(new ClosingServer());
}

public function testMain() returns error? {
    tcp:Client c = check new ("127.0.0.1", 19400, timeout = 2);
    check c->writeBytes("hello".toBytes());
    readonly & byte[]|tcp:Error response = c->readBytes();
    check c->close();
    check closingListener.gracefulStop();
    io:println("stopped"); // @output stopped
    io:println(response is tcp:Error); // @output true
}
