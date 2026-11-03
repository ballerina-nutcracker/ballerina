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

// Exercises Listener.detach (identity-checked, unlike jBallerina's
// any-detach-clears-whatever's-attached bug) and Listener.immediateStop
// (a real force-stop in this port, unlike jBallerina's unimplemented stub —
// see the README's Notable Behavioural Changes).
import ballerina/io;
import ballerina/udp;

service class NoopService {
    *udp:Service;

    remote function onBytes(readonly & byte[] data) {
        _ = data;
    }
}

service class EchoService {
    *udp:Service;

    remote function onBytes(readonly & byte[] data) returns byte[] {
        return data;
    }
}

listener udp:Listener lifecycleListener = new (19492);

function init() returns error? {
    NoopService first = new NoopService();
    check lifecycleListener.attach(first);
    check lifecycleListener.detach(first);
    check lifecycleListener.attach(new NoopService());
}

public function testMain() returns error? {
    check lifecycleListener.immediateStop();

    // Binding a fresh listener to the same port only succeeds if
    // immediateStop() truly released the previous socket (unlike
    // jBallerina's unimplemented no-op stub).
    udp:Listener second = check new (19492);
    check second.attach(new NoopService());
    udp:Error? startResult = second.'start();
    io:println(startResult is ()); // @output true
    check second.gracefulStop();

    udp:Listener noService = check new (19492);
    io:println(noService.'start() is ()); // @output true
    check noService.gracefulStop();

    udp:Listener restarted = check new (19492);
    check restarted.attach(new EchoService());
    check restarted.'start();
    check restarted.gracefulStop();
    check restarted.'start();
    udp:ConnectClient c = check new ("127.0.0.1", 19492, {timeout: 5});
    check c->writeBytes("hi".toBytes());
    readonly & byte[] echoed = check c->readBytes();
    io:println(check string:fromBytes(echoed)); // @output hi
    check c->close();
    check restarted.gracefulStop();
}
