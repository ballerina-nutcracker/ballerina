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

final messaging:InMemoryMessageStore orders = new;
final messaging:InMemoryMessageStore failedOrders = new;

listener messaging:StoreListener orderListener = new (orders, {
    pollingInterval: 0.02,
    maxRetries: 1,
    retryInterval: 0.02,
    deadLetterStore: failedOrders
});

isolated string[] shipped = [];

isolated function shippedCount() returns int {
    lock {
        return shipped.length();
    }
}

isolated service on orderListener {
    isolated remote function onMessage(anydata payload) returns error? {
        if payload !is map<anydata> {
            return error("unexpected payload");
        }
        lock {
            shipped.push(payload["item"].toString());
        }
    }
}

public function testMain() returns error? {
    orders->store({item: "book"});
    orders->store({item: "pen"});
    foreach int _ in 0 ..< 250 {
        if shippedCount() >= 2 {
            break;
        }
        runtime:sleep(0.02);
    }
    lock {
        // Overlapping poll runs can finish in any order.
        string[] sorted = from string item in shipped order by item select item;
        io:println("shipped: ", sorted.toString()); // @output shipped: ["book","pen"]
    }

    messaging:StoreListener|messaging:Error invalid = new (orders, maxRetries = -1);
    io:println(invalid); // @output error Error ("maxRetries must be greater than or equal to zero")
    invalid = new (orders, pollingInterval = 0);
    io:println(invalid); // @output error Error ("pollingInterval must be greater than zero")
    invalid = new (orders, retryInterval = -1);
    io:println(invalid); // @output error Error ("retryInterval must be greater than zero")
}
