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

// Exercises the listener -> onBytes -> auto-reply path: returning byte[]
// from onBytes sends it back to the datagram's sender without an explicit
// Caller call.
import ballerina/io;
import ballerina/udp;

service class EchoService {
    *udp:Service;

    remote function onBytes(readonly & byte[] data, udp:Caller caller) returns byte[]|udp:Error? {
        _ = caller.remoteHost;
        return data;
    }
}

listener udp:Listener echoListener = new (19490);

function init() returns error? {
    check echoListener.attach(new EchoService());
}

// testMain is invoked by the harness while the runtime is parked in the
// listening state. It drives the live service over a real udp:ConnectClient.
public function testMain() returns error? {
    udp:ConnectClient c = check new ("127.0.0.1", 19490, {timeout: 5});
    check c->writeBytes("hello".toBytes());
    readonly & byte[] echoed = check c->readBytes();
    io:println('string:fromBytes(echoed)); // @output hello
    check c->close();
}
