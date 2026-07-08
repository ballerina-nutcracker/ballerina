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

// Exercises the listener -> onDatagram -> explicit Caller.sendDatagram path,
// driven by a connectionless udp:Client.
import ballerina/io;
import ballerina/udp;

service class DatagramService {
    *udp:Service;

    remote function onDatagram(readonly & udp:Datagram datagram, udp:Caller caller) returns udp:Error? {
        check caller->sendDatagram({remoteHost: datagram.remoteHost, remotePort: datagram.remotePort, data: datagram.data});
    }
}

listener udp:Listener datagramListener = new (19491);

function init() returns error? {
    check datagramListener.attach(new DatagramService());
}

public function testMain() returns error? {
    udp:Client c = check new ({timeout: 5});
    check c->sendDatagram({remoteHost: "127.0.0.1", remotePort: 19491, data: "hello".toBytes()});
    readonly & udp:Datagram echoed = check c->receiveDatagram();
    io:println('string:fromBytes(echoed.data)); // @output hello
    check c->close();
}
