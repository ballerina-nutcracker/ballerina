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

// A handler that panics or returns an error is logged to stderr and the listener keeps serving.
import ballerina/io;
import ballerina/udp;

service class FaultyService {
    *udp:Service;

    remote function onBytes(readonly & byte[] data) returns byte[]|udp:Error {
        if data.length() == 1 {
            panic error("boom");
        }
        if data.length() == 2 {
            return error("handler failed");
        }
        return data;
    }
}

listener udp:Listener faultyListener = new (19498);

function init() returns error? {
    check faultyListener.attach(new FaultyService());
}

public function testMain() returns error? {
    udp:ConnectClient c = check new ("127.0.0.1", 19498, {timeout: 1});
    check c->writeBytes([1]);
    byte[]|udp:Error r1 = c->readBytes();
    io:println(r1 is udp:Error ? r1.message() : "ok"); // @output Read timed out
    check c->writeBytes([1, 2]);
    byte[]|udp:Error r2 = c->readBytes();
    io:println(r2 is udp:Error ? r2.message() : "ok"); // @output Read timed out
    check c->writeBytes("hello".toBytes());
    byte[] r3 = check c->readBytes();
    io:println(check string:fromBytes(r3)); // @output hello
    check c->close();
}
