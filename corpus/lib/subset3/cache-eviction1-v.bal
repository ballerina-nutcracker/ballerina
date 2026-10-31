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

function fill(cache:Cache c, int n) returns error? {
    foreach int i in 0 ..< n {
        check c.put("k" + i.toString(), i);
    }
}

function sizeAfterOverflow(int capacity, float evictionFactor) returns int|error {
    cache:Cache c = new (capacity = capacity, evictionFactor = evictionFactor);
    check fill(c, capacity);
    check c.put("new", 1);
    return c.size();
}

public function main() returns error? {
    // Evicts ceil(capacity * evictionFactor) entries, computed exactly from the decimal factor and clamped to [1, capacity].
    io:println(check sizeAfterOverflow(10, 0.3)); // @output 8
    io:println(check sizeAfterOverflow(50, 0.3)); // @output 36
    io:println(check sizeAfterOverflow(100, 0.6)); // @output 41
    io:println(check sizeAfterOverflow(100, 0.15)); // @output 86
    io:println(check sizeAfterOverflow(10, 0.25)); // @output 8
    io:println(check sizeAfterOverflow(3, 0.1)); // @output 3
    io:println(check sizeAfterOverflow(10, 0.1)); // @output 10
    io:println(check sizeAfterOverflow(3, 0.7)); // @output 1
    io:println(check sizeAfterOverflow(7, 0.2)); // @output 6
    io:println(check sizeAfterOverflow(100, 0.07)); // @output 94
    io:println(check sizeAfterOverflow(10, 1.0)); // @output 1
    io:println(check sizeAfterOverflow(5, 0.01)); // @output 5
    io:println(check sizeAfterOverflow(1, 0.25)); // @output 1

    cache:Cache defaults = new (capacity = 8);
    check fill(defaults, 8);
    check defaults.put("new", 1);
    io:println(defaults.size(), " ", defaults.keys()); // @output 7 ["k2","k3","k4","k5","k6","k7","new"]

    cache:Cache c = new (capacity = 3, evictionFactor = 0.34);
    check fill(c, 3);
    _ = c.hasKey("k0");
    check c.put("k0", 99);
    io:println(c.size(), " ", c.hasKey("k0"), " ", c.hasKey("k1"), " ", c.hasKey("k2")); // @output 3 true true true
    io:println(check c.get("k0")); // @output 99

    cache:Cache d = new (capacity = 3, evictionFactor = 0.34);
    check fill(d, 3);
    _ = check d.get("k0");
    check d.put("k1", 11);
    check d.put("x", 1);
    io:println(d.keys()); // @output ["k1","x"]

    cache:Cache e = new (capacity = 3, evictionFactor = 0.34);
    check fill(e, 3);
    check e.invalidate("k1");
    check e.put("y", 1);
    io:println(e.size(), " ", e.keys()); // @output 3 ["k0","k2","y"]
    check e.invalidateAll();
    check e.put("z", 1);
    io:println(e.size(), " ", e.keys()); // @output 1 ["z"]

    cache:Cache tiny = new (capacity = 2, evictionFactor = 1.0E-300);
    check fill(tiny, 10);
    io:println(tiny.size(), " ", tiny.keys()); // @output 2 ["k8","k9"]
}
