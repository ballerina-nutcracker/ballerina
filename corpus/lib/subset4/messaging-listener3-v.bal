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

// Stands in for a remote broker whose retrieve is slow to answer.
isolated client class SlowStore {
    *messaging:Store;

    isolated remote function store(anydata payload) returns error? {
        _ = payload;
    }

    isolated remote function retrieve() returns messaging:Message|error? {
        runtime:sleep(5);
        return;
    }

    isolated remote function acknowledge(string id, boolean success = true) returns error? {
        _ = id;
        _ = success;
    }
}

isolated int received = 0;

isolated service class Counter {
    *messaging:StoreService;

    isolated remote function onMessage(anydata payload) returns error? {
        _ = payload;
        lock {
            received += 1;
        }
    }
}

isolated function awaitReceived(int expected) returns int {
    foreach int _ in 0 ..< 40 {
        lock {
            if received >= expected {
                return received;
            }
        }
        runtime:sleep(0.02);
    }
    lock {
        return received;
    }
}

public function main() returns error? {
    // A slow store's retrieve holds up neither another listener's polls nor its stop.
    messaging:StoreListener slow = check new (new SlowStore(), pollingInterval = 0.1);
    check slow.attach(new Counter());
    check slow.'start();

    messaging:InMemoryMessageStore orders = new;
    messaging:StoreListener fast = check new (orders, pollingInterval = 0.05);
    check fast.attach(new Counter());
    check fast.'start();
    runtime:sleep(0.3);
    foreach int i in 0 ..< 3 {
        orders->store(i);
    }
    io:println(awaitReceived(3)); // @output 3
    check fast.immediateStop();
    check slow.immediateStop();
    io:println("stopped"); // @output stopped
}
