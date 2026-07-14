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

// Regression test: onDatagram may be declared with the bare
// onDatagram(readonly & udp:Datagram datagram) form (no Caller parameter) —
// this is the exact shape used by this module's own official examples. See
// remoteMethodArgs in native/dispatch.go — invoking this form with a
// hardcoded 3-argument call used to crash the interpreter's argument
// binding on the first datagram, taking down the whole listener process.
import ballerina/io;
import ballerina/udp;

service class BareEchoService {
    *udp:Service;

    remote function onDatagram(readonly & udp:Datagram datagram) returns udp:Datagram {
        return datagram;
    }
}

listener udp:Listener bareOnDatagramListener = new (19495);

function init() returns error? {
    check bareOnDatagramListener.attach(new BareEchoService());
}

public function testMain() returns error? {
    udp:Client c = check new ({timeout: 5});
    check c->sendDatagram({remoteHost: "127.0.0.1", remotePort: 19495, data: "hello".toBytes()});
    readonly & udp:Datagram echoed = check c->receiveDatagram();
    io:println('string:fromBytes(echoed.data)); // @output hello
    check c->close();
}
