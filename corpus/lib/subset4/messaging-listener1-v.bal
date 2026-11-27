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

import ballerina/io;
import ballerina/lang.runtime;
import ballerina/messaging;

isolated anydata[] received = [];

isolated function receivedCount() returns int {
    lock {
        return received.length();
    }
}

isolated function awaitReceived(int expected) returns int {
    foreach int _ in 0 ..< 250 {
        if receivedCount() >= expected {
            break;
        }
        runtime:sleep(0.02);
    }
    return receivedCount();
}

isolated service class Recorder {
    *messaging:StoreService;

    isolated remote function onMessage(anydata payload) returns error? {
        if payload is map<anydata> {
            payload["seen"] = true;
        }
        lock {
            received.push(payload.clone());
        }
    }
}

public function main() returns error? {
    messaging:InMemoryMessageStore store = new;
    messaging:StoreListener l = check new (store, pollingInterval = 0.02);
    Recorder svc = new;

    io:println(l.detach(svc)); // @output error Error ("no service is attached")
    check l.'start();
    store->store("before attach");
    runtime:sleep(0.1);
    io:println(receivedCount()); // @output 0

    check l.attach(svc);
    io:println(l.attach(new Recorder())); // @output error Error ("service is already attached. Only one service can be attached to the message store listener")
    check l.'start();
    check l.'start();
    map<anydata> event = {kind: "created"};
    store->store(event);
    store->store(1);
    io:println(awaitReceived(3)); // @output 3
    lock {
        // Overlapping poll runs can finish in any order.
        io:println(received.indexOf("before attach") is int, " ", received.indexOf({kind: "created", seen: true}) is int,
            " ", received.indexOf(1) is int); // @output true true true
    }
    messaging:Message? drained = store->retrieve();
    io:println(drained is ()); // @output true
    io:println(event); // @output {"kind":"created"}

    check l.immediateStop();
    check l.immediateStop();
    // No message is delivered once immediateStop returns (ballerina-platform/ballerina-library#9901).
    store->store("while stopped");
    runtime:sleep(0.1);
    io:println(receivedCount()); // @output 3

    check l.'start();
    io:println(awaitReceived(4)); // @output 4

    // A rejected detach must not stop polling: https://github.com/ballerina-platform/ballerina-library/issues/9704
    io:println(l.detach(new Recorder())); // @output error Error ("the provided service is not attached to the listener")
    store->store("after rejected detach");
    io:println(awaitReceived(5)); // @output 5
    check l.gracefulStop();

    check l.detach(svc);
    check l.attach(new Recorder());
}
