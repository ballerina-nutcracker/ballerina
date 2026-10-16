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

// Client and ConnectClient against a udp:Listener: non-positive timeouts, open config records, use after close and closing twice.
import ballerina/io;
import ballerina/udp;

service class EchoService {
    *udp:Service;

    remote function onBytes(readonly & byte[] data) returns byte[] {
        return data;
    }
}

listener udp:Listener clientEchoListener = new (19497);

function init() returns error? {
    check clientEchoListener.attach(new EchoService());
}

public function testMain() returns error? {
    udp:Client c = check new ({timeout: 0, "foo": 1});
    check c->sendDatagram({remoteHost: "127.0.0.1", remotePort: 19497, data: "hello".toBytes()});
    readonly & udp:Datagram echoed = check c->receiveDatagram();
    io:println(check string:fromBytes(echoed.data)); // @output hello
    check c->close();
    udp:Error? afterClose = c->sendDatagram({remoteHost: "127.0.0.1", remotePort: 19497, data: "x".toBytes()});
    io:println(afterClose is udp:Error); // @output true
    udp:Error? closeTwice = c->close();
    io:println(closeTwice is ()); // @output true

    udp:ConnectClient cc = check new ("127.0.0.1", 19497, {timeout: -1d, "foo": 1});
    check cc->writeBytes("world".toBytes());
    readonly & byte[] echoed2 = check cc->readBytes();
    io:println(check string:fromBytes(echoed2)); // @output world
    check cc->close();
    udp:Error? writeAfterClose = cc->writeBytes("x".toBytes());
    io:println(writeAfterClose is udp:Error); // @output true
    udp:Error? ccCloseTwice = cc->close();
    io:println(ccCloseTwice is ()); // @output true
}
