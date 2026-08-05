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

// Exercises Client.readBytes' read-timeout error path against a listener
// that accepts a connection but never writes anything back.
import ballerina/io;
import ballerina/tcp;

service class SilentConnService {
    *tcp:ConnectionService;
}

service class SilentServer {
    *tcp:Service;

    remote function onConnect(tcp:Caller caller) returns tcp:ConnectionService {
        _ = caller.id;
        return new SilentConnService();
    }
}

listener tcp:Listener silentListener = new (19397);

function init() returns error? {
    check silentListener.attach(new SilentServer());
}

public function testMain() returns error? {
    tcp:Client c = check new ("127.0.0.1", 19397, {timeout: 0.05});
    (readonly & byte[])|tcp:Error readResult = c->readBytes();
    io:println(readResult is tcp:Error); // @output true
    check c->close();
}
