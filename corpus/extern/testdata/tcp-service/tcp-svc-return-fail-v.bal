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

// Exercises a failed write of the byte[] returned from onBytes: onBytes waits
// for the client to close, writes once (making the peer reset the connection),
// then returns the data. That write fails, so onError gets "Socket connection
// already closed." and the connection is closed with onClose, without a second
// onError for the read side. jBallerina reports the same onError and onClose,
// but closes on the peer's FIN before the first write, so its first write
// already fails (see the tcp README behavioural notes).
import ballerina/io;
import ballerina/lang.runtime;
import ballerina/tcp;

isolated boolean clientClosed = false;
isolated boolean serverClosed = false;
isolated int errorCount = 0;

service class ReturnFailConnService {
    *tcp:ConnectionService;

    remote function onBytes(tcp:Caller caller, readonly & byte[] data) returns byte[] {
        foreach int _ in 0 ..< 500 {
            if isClientClosed() {
                break;
            }
            runtime:sleep(0.01);
        }
        runtime:sleep(0.1);
        tcp:Error? firstWrite = caller->writeBytes(data);
        if firstWrite is tcp:Error {
            io:println("first write failed early");
        }
        runtime:sleep(0.2);
        return data;
    }

    remote function onError(readonly & tcp:Error err) {
        io:println("onError: ", err.message()); // @output onError: Socket connection already closed.
        lock {
            errorCount += 1;
        }
    }

    remote function onClose() {
        io:println("onClose"); // @output onClose
        lock {
            serverClosed = true;
        }
    }
}

service class ReturnFailServer {
    *tcp:Service;

    remote function onConnect(tcp:Caller caller) returns tcp:ConnectionService {
        _ = caller.id;
        return new ReturnFailConnService();
    }
}

listener tcp:Listener returnFailListener = new (19410, localHost = "127.0.0.1");

function init() returns error? {
    check returnFailListener.attach(new ReturnFailServer());
}

function isClientClosed() returns boolean {
    lock {
        return clientClosed;
    }
}

function isServerClosed() returns boolean {
    lock {
        return serverClosed;
    }
}

function errors() returns int {
    lock {
        return errorCount;
    }
}

public function testMain() returns error? {
    tcp:Client c = check new ("127.0.0.1", 19410);
    check c->writeBytes("hello".toBytes());
    check c->close();
    lock {
        clientClosed = true;
    }
    foreach int _ in 0 ..< 500 {
        if isServerClosed() {
            break;
        }
        runtime:sleep(0.01);
    }
    runtime:sleep(0.2);
    io:println("onError count: ", errors()); // @output onError count: 1
    check returnFailListener.gracefulStop();
}
