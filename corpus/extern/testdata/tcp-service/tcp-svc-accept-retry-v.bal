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

// The driving Go test injects a listener whose first Accept fails with a
// transient error; the accept loop must retry instead of exiting.
import ballerina/io;
import ballerina/tcp;

service class EchoService {
    *tcp:ConnectionService;

    remote function onBytes(readonly & byte[] data) returns byte[] {
        return data;
    }
}

service class EchoServer {
    *tcp:Service;

    remote function onConnect(tcp:Caller caller) returns tcp:ConnectionService {
        _ = caller.id;
        return new EchoService();
    }
}

listener tcp:Listener retryListener = new (19411, localHost = "127.0.0.1");

function init() returns error? {
    check retryListener.attach(new EchoServer());
}

public function testMain() returns error? {
    tcp:Client c = check new ("127.0.0.1", 19411, timeout = 5);
    check c->writeBytes("after retry".toBytes());
    readonly & byte[] echoed = check c->readBytes();
    io:println('string:fromBytes(echoed)); // @output after retry
    check c->close();
}
