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

import ballerina/cache;
import ballerina/io;

const WORKERS = 8;
const PER_WORKER = 50;
const ROUNDS = 5000;

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
    cache:Cache warm = new;
    check warm.put("k", 0);
    _ = drop(warm);
    _ = drop(warm);
    check fillPrefixed(new, "w");

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

    // Two strands invalidating the same key never both succeed (ballerina-platform/ballerina-library#9693).
    cache:Cache shared = new;
    int both = 0;
    foreach int i in 0 ..< ROUNDS {
        check shared.put("k", i);
        future<boolean> a = start drop(shared);
        future<boolean> b = start drop(shared);
        boolean droppedA = check wait a;
        boolean droppedB = check wait b;
        if droppedA && droppedB {
            both += 1;
        }
    }
    io:println(both); // @output 0

    // Strands filling two caches never put an entry into the other one (ballerina-platform/ballerina-library#9405).
    cache:Cache first = new (capacity = 100000);
    cache:Cache second = new (capacity = 100000);
    future<error?> fillFirst = start fillPrefixed(first, "a");
    future<error?> fillSecond = start fillPrefixed(second, "b");
    check wait fillFirst;
    check wait fillSecond;
    io:println(first.size(), " ", prefixed(first, "b"), " ", second.size(), " ", prefixed(second, "a")); // @output 20000 0 20000 0
}

isolated function fillPrefixed(cache:Cache c, string prefix) returns error? {
    foreach int i in 0 ..< 20000 {
        check c.put(prefix + i.toString(), i);
    }
}

function prefixed(cache:Cache c, string prefix) returns int {
    int count = 0;
    foreach string key in c.keys() {
        if key.substring(0, 1) == prefix {
            count += 1;
        }
    }
    return count;
}

isolated function drop(cache:Cache c) returns boolean {
    return c.invalidate("k") is ();
}

isolated function fillSmall(cache:Cache c, int id) returns int|error {
    foreach int i in 0 ..< PER_WORKER {
        check c.put(string `s${id}-${i}`, i);
    }
    return 0;
}
