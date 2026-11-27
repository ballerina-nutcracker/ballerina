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

// An onConnect that returns () keeps the connection tracked and read, so
// immediateStop() still closes it.
import ballerina/io;
import ballerina/lang.runtime;
import ballerina/tcp;

service class NoConnServiceServer {
    *tcp:Service;

    remote function onConnect(tcp:Caller caller) {
        _ = caller.id;
        io:println("onConnect"); // @output onConnect
    }
}

listener tcp:Listener noConnSvcListener = new (19413, localHost = "127.0.0.1");

function init() returns error? {
    check noConnSvcListener.attach(new NoConnServiceServer());
}

public function testMain() returns error? {
    tcp:Client c = check new ("127.0.0.1", 19413, timeout = 3);
    check c->writeBytes("dropped".toBytes());
    runtime:sleep(0.5);
    check noConnSvcListener.immediateStop();
    (readonly & byte[])|tcp:Error r = c->readBytes();
    io:println(r is tcp:Error ? r.message() : "data"); // @output Connection closed by the server.
    check c->close();
}
