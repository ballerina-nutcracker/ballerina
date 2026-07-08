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

// Exercises the connection-service onError dispatch path: a raw TCP reset
// (RST) on an accepted connection surfaces as a non-EOF read error, which
// the read loop routes to onError rather than treating as a normal close.
// signalConnected/resetRawConnection/signalOnError/waitForOnError are
// test-only externs registered by the driving Go test (see
// tcp_onerror_test.go). signalConnected fires from onConnect so the Go side
// only resets the raw connection once the interpreter has actually accepted
// it and dispatched onConnect — otherwise the reset can race the kernel's
// accept queue and never reach the interpreter's read loop at all.
import ballerina/io;
import ballerina/tcp;

service class ErrorReportingConnService {
    *tcp:ConnectionService;

    remote function onError(readonly & tcp:Error err) returns tcp:Error? {
        _ = err.message();
        signalOnError();
    }
}

service class ErrorReportingServer {
    *tcp:Service;

    remote function onConnect(tcp:Caller caller) returns tcp:ConnectionService {
        _ = caller.id;
        signalConnected();
        return new ErrorReportingConnService();
    }
}

listener tcp:Listener errorListener = new (19398);

function init() returns error? {
    check errorListener.attach(new ErrorReportingServer());
}

isolated function signalConnected() = external;
isolated function resetRawConnection(int port) returns error? = external;
isolated function signalOnError() = external;
isolated function waitForOnError() returns error? = external;

public function testMain() returns error? {
    check resetRawConnection(19398);
    error? waitResult = waitForOnError();
    io:println(waitResult is ()); // @output true
}
