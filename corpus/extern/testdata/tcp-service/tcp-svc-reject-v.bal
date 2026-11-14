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

// jBallerina leaves a connection open with reads paused forever when onConnect
// returns an error; this port closes it instead (see the README's Notable Behavioural Changes).
import ballerina/io;
import ballerina/tcp;

service class RejectingServer {
    *tcp:Service;

    remote function onConnect(tcp:Caller caller) returns tcp:Error? {
        _ = caller.id;
        return error("rejected");
    }
}

listener tcp:Listener rejectListener = new (19391);

function init() returns error? {
    check rejectListener.attach(new RejectingServer());
}

public function testMain() returns error? {
    tcp:Client c = check new ("localhost", 19391);
    // The write's outcome is timing-dependent (the server may already have
    // closed by the time it arrives), so it is deliberately not asserted on.
    tcp:Error? writeResult = c->writeBytes("x".toBytes());
    boolean _ = writeResult is error;
    (readonly & byte[])|tcp:Error readResult = c->readBytes();
    io:println(readResult is tcp:Error); // @output true

    // Once the peer has reset the connection, a write reports it as closed. The first writes may still be buffered locally.
    tcp:Error? lastWrite = ();
    foreach int _ in 0 ..< 100 {
        lastWrite = c->writeBytes("x".toBytes());
        if lastWrite is tcp:Error {
            break;
        }
    }
    io:println(lastWrite is tcp:Error ? lastWrite.message() : "no error"); // @output Socket connection already closed.
}
