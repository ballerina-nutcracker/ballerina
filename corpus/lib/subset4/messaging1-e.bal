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

import ballerina/messaging;

isolated service class NoRemote {
    isolated function onMessage(anydata payload) returns error? {
        _ = payload;
    }
}

isolated client class PartialStore {
    isolated remote function store(anydata payload) returns error? {
        _ = payload;
    }
}

function attachNonRemote(messaging:StoreListener l) returns error? {
    check l.attach(new NoRemote()); // @error
}

function attachWithPath(messaging:StoreListener l, messaging:StoreService svc) returns error? {
    check l.attach(svc, "/orders"); // @error
}

function partialStore() returns messaging:StoreListener|error {
    return new (new PartialStore()); // @error
}

function readonlyId(messaging:Message msg) {
    msg.id = "2"; // @error
}

function ackNonStringId(messaging:InMemoryMessageStore store) returns error? {
    check store->acknowledge(1); // @error
}

function unknownOption(messaging:Store store) returns messaging:StoreListener|error {
    return new (store, batchSize = 10); // @error
}

public function main() {
}
