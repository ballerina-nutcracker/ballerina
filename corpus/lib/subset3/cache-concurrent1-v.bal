// Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
//
// WSO2 LLC. licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except
// in compliance with the License.
//
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

import ballerina/cache;
import ballerina/io;

const WORKERS = 8;
const PER_WORKER = 50;

isolated function work(cache:Cache c, int id) returns int|error {
    int hits = 0;
    foreach int i in 0 ..< PER_WORKER {
        string key = string `w${id}-${i}`;
        check c.put(key, i);
        any v = check c.get(key);
        if v == i {
            hits += 1;
        }
        _ = c.hasKey(key);
        _ = c.keys();
    }
    return hits;
}

public function main() returns error? {
    // Run every call site once on this strand first: the interpreter's per-call-site callee cache is not yet thread-safe (#1123).
    _ = check work(new (capacity = PER_WORKER), -1);
    _ = check fillSmall(new (capacity = 10, evictionFactor = 0.5), -1);

    cache:Cache big = new (capacity = WORKERS * PER_WORKER);
    future<int|error>[] futures = [];
    foreach int id in 0 ..< WORKERS {
        future<int|error> f = start work(big, id);
        futures.push(f);
    }
    int total = 0;
    foreach future<int|error> f in futures {
        total += check wait f;
    }
    io:println(total, " ", big.size()); // @output 400 400

    cache:Cache small = new (capacity = 10, evictionFactor = 0.5);
    future<int|error>[] evicting = [];
    foreach int id in 0 ..< WORKERS {
        future<int|error> f = start fillSmall(small, id);
        evicting.push(f);
    }
    foreach future<int|error> f in evicting {
        _ = check wait f;
    }
    io:println(small.size() <= small.capacity(), " ", small.keys().length() == small.size()); // @output true true
}

isolated function fillSmall(cache:Cache c, int id) returns int|error {
    foreach int i in 0 ..< PER_WORKER {
        check c.put(string `s${id}-${i}`, i);
    }
    return 0;
}
