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
import ballerina/messaging;

public function main() returns error? {
    messaging:InMemoryMessageStore store = new;
    messaging:Message? empty = store->retrieve();
    io:println(empty is ()); // @output true

    map<anydata> cart = {id: 1, items: ["a", "b"]};
    store->store(cart);
    store->store("second");
    cart["id"] = 99;

    messaging:Message? first = store->retrieve();
    if first !is messaging:Message {
        return error("expected a message");
    }
    io:println(first.payload); // @output {"id":1,"items":["a","b"]}

    messaging:Message? next = store->retrieve();
    if next !is messaging:Message {
        return error("expected a message");
    }
    io:println(next.payload); // @output second
    messaging:Message? allInFlight = store->retrieve();
    io:println(allInFlight is ()); // @output true

    map<anydata> retrieved = <map<anydata>>first.payload;
    retrieved["id"] = 42;
    check store->acknowledge(first.id, false);
    messaging:Message? redelivered = store->retrieve();
    if redelivered !is messaging:Message {
        return error("expected a message");
    }
    io:println(redelivered.id == first.id, " ", redelivered.payload); // @output true {"id":1,"items":["a","b"]}

    check store->acknowledge(redelivered.id);
    check store->acknowledge(next.id);
    messaging:Message? drained = store->retrieve();
    io:println(drained is ()); // @output true

    error? unknown = store->acknowledge("missing");
    if unknown is error {
        io:println(unknown); // @output error("Message with the given ID not found or not in flight",id="missing")
    }
    error? twice = store->acknowledge(next.id);
    io:println(twice is error); // @output true

    store->store(());
    messaging:Message? nilPayload = store->retrieve();
    if nilPayload is messaging:Message {
        io:println(nilPayload.payload is ()); // @output true
        error? released = store->acknowledge(nilPayload.id, false);
        io:println(released is ()); // @output true
        error? notInFlight = store->acknowledge(nilPayload.id, false);
        io:println(notInFlight is error); // @output true
    }

    messaging:Store abstractStore = store;
    check abstractStore->store(3.5d);
    messaging:Message|error? viaInterface = abstractStore->retrieve();
    io:println(viaInterface is messaging:Message); // @output true
}
