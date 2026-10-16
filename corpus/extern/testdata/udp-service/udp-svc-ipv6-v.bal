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

// A listener replies to a ConnectClient dialing the IPv6 literal "::1".
import ballerina/io;
import ballerina/udp;

service class EchoService {
    *udp:Service;

    remote function onDatagram(readonly & udp:Datagram datagram, udp:Caller caller) returns error? {
        check caller->sendDatagram({remoteHost: datagram.remoteHost, remotePort: datagram.remotePort, data: datagram.data});
    }
}

listener udp:Listener ipv6Listener = new (19496);

function init() returns error? {
    check ipv6Listener.attach(new EchoService());
}

public function testMain() returns error? {
    udp:ConnectClient c = check new ("::1", 19496, {timeout: 5});
    check c->writeBytes("hello".toBytes());
    readonly & byte[] echoed = check c->readBytes();
    io:println('string:fromBytes(echoed)); // @output hello
    check c->close();
}
