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

// A connection service without onBytes is accepted, as jBallerina does: bytes
// sent on the connection are dropped and onClose still fires on disconnect.
import ballerina/io;
import ballerina/lang.runtime;
import ballerina/tcp;

boolean closed = false;

service class CloseOnlyServer {
    *tcp:Service;

    remote function onConnect(tcp:Caller caller) returns tcp:ConnectionService {
        io:println("onConnect ", caller.localPort > 0); // @output onConnect true
        return new CloseOnlyConnService();
    }
}

service class CloseOnlyConnService {
    *tcp:ConnectionService;

    remote function onClose() {
        io:println("onClose"); // @output onClose
        closed = true;
    }
}

public function testMain() returns error? {
    tcp:Listener l = check new (21932, localHost = "127.0.0.1");
    check l.attach(new CloseOnlyServer());
    check l.'start();
    tcp:Client c = check new ("127.0.0.1", 21932, timeout = 1);
    check c->writeBytes("hi".toBytes());
    check c->close();
    foreach int _ in 0 ..< 100 {
        if closed {
            break;
        }
        runtime:sleep(0.02);
    }
    check l.immediateStop();
    io:println(closed); // @output true
}
