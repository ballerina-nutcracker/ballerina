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
// jBallerina's reflection-driven parameter binding. See onBytesArgs in
// native/dispatch.go — it inspects the resolved method's declared parameter
// types (via semtypes.ObjectMemberType/FunctionParamListType) rather than
// assuming a fixed two-parameter signature.
import ballerina/io;
import ballerina/tcp;

service class BareEchoService {
    *tcp:ConnectionService;

    remote function onBytes(readonly & byte[] data) returns byte[] {
        return data;
    }
}

service class BareEchoServer {
    *tcp:Service;

    remote function onConnect(tcp:Caller caller) returns tcp:ConnectionService {
        _ = caller.id;
        return new BareEchoService();
    }
}

listener tcp:Listener bareEchoListener = new (19399);

function init() returns error? {
    check bareEchoListener.attach(new BareEchoServer());
}

public function testMain() returns error? {
    tcp:Client c = check new ("localhost", 19399, {});
    check c->writeBytes("hello".toBytes());
    readonly & byte[] echoed = check c->readBytes();
    io:println('string:fromBytes(echoed)); // @output hello
    check c->close();
}
