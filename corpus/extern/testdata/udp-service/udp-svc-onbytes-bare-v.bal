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

// Regression test: onBytes may be declared with the bare
// onBytes(readonly & byte[] data) form (no Caller parameter), matching
// jBallerina's reflection-driven parameter binding. See remoteMethodArgs in
// native/dispatch.go — invoking this form with a hardcoded 3-argument call
// used to crash the interpreter's argument binding on the first datagram.
import ballerina/io;
import ballerina/udp;

service class BareEchoService {
    *udp:Service;

    remote function onBytes(readonly & byte[] data) returns byte[] {
        return data;
    }
}

listener udp:Listener bareOnBytesListener = new (19494);

function init() returns error? {
    check bareOnBytesListener.attach(new BareEchoService());
}

public function testMain() returns error? {
    udp:ConnectClient c = check new ("127.0.0.1", 19494, {timeout: 5});
    check c->writeBytes("hello".toBytes());
    readonly & byte[] echoed = check c->readBytes();
    io:println('string:fromBytes(echoed)); // @output hello
    check c->close();
}
