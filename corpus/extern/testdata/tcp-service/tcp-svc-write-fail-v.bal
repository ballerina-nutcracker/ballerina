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

// Exercises Caller write failures once the peer has gone away: the client
// writes then closes, so the server's later writes hit a reset connection.
// The first failed caller->writeBytes closes the connection (dispatching
// onClose), as Netty's auto-close does in jBallerina, so later writes and the
// byte[] returned from onBytes fail as "Socket connection already closed."
// and the read loop reports no further onError.
import ballerina/io;
import ballerina/lang.runtime;
import ballerina/tcp;

isolated int errorCount = 0;

service class PeerGoneConnService {
    *tcp:ConnectionService;

    remote function onClose() {
        io:println("onClose"); // @output onClose
    }

    remote function onBytes(tcp:Caller caller, readonly & byte[] data) returns byte[] {
        tcp:Error? writeErr = ();
        foreach int _ in 0 ..< 500 {
            writeErr = caller->writeBytes(data);
            if writeErr is tcp:Error {
                break;
            }
            runtime:sleep(0.01);
        }
        io:println("caller write: ", writeErr is tcp:Error ? writeErr.message() : "no error"); // @output caller write: Socket connection already closed.
        tcp:Error? againErr = caller->writeBytes(data);
        io:println("caller write again: ", againErr is tcp:Error ? againErr.message() : "no error"); // @output caller write again: Socket connection already closed.
        return data;
    }

    remote function onError(readonly & tcp:Error err) {
        io:println("onError: ", err.message()); // @output onError: Socket connection already closed.
        lock {
            errorCount += 1;
        }
    }
}

service class PeerGoneServer {
    *tcp:Service;

    remote function onConnect(tcp:Caller caller) returns tcp:ConnectionService {
        _ = caller.id;
        return new PeerGoneConnService();
    }
}

listener tcp:Listener peerGoneListener = new (19408, localHost = "127.0.0.1");

function init() returns error? {
    check peerGoneListener.attach(new PeerGoneServer());
}

function errors() returns int {
    lock {
        return errorCount;
    }
}

public function testMain() returns error? {
    tcp:Client c = check new ("127.0.0.1", 19408);
    check c->writeBytes("hello".toBytes());
    check c->close();
    foreach int _ in 0 ..< 500 {
        if errors() > 0 {
            break;
        }
        runtime:sleep(0.01);
    }
    runtime:sleep(0.2);
    io:println("onError count: ", errors()); // @output onError count: 1
    check peerGoneListener.gracefulStop();
}
