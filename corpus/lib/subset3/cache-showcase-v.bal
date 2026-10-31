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
import ballerina/lang.runtime;

public function main() returns error? {
    cache:Cache c = new (capacity = 4, evictionFactor = 0.5);
    check c.put("a", "A");
    check c.put("b", 2);
    check c.put("c", [1, 2]);
    _ = check c.get("a");
    check c.put("d", {x: 1});
    check c.put("e", "E");
    io:println(c.size(), " ", c.capacity()); // @output 3 4
    io:println(c.hasKey("b"), " ", c.hasKey("a")); // @output false true
    io:println(check c.get("a")); // @output A

    cache:Error? e = c.put("n", ());
    if e is cache:Error {
        io:println(e.message()); // @output Unsupported cache value '()' for the key: n.
    }
    any|cache:Error m = c.get("zz");
    if m is cache:Error {
        io:println(m.message()); // @output Cache entry from the given key: zz, is not available.
    }
    check c.invalidate("a");
    cache:AbstractCache ac = c;
    io:println(ac.size()); // @output 2
    check c.invalidateAll();
    io:println(c.keys().length()); // @output 0

    check c.put("t", "T", 1);
    runtime:sleep(2);
    io:println(c.size(), " ", c.hasKey("t"), " ", c.size()); // @output 1 false 0

    cache:Cache c2 = new (defaultMaxAge = 1, cleanupInterval = 1);
    check c2.put("x", 1);
    io:println(waitForSize(c2, 0)); // @output 0
}

function waitForSize(cache:Cache c, int expected) returns int {
    foreach int _ in 0 ..< 100 {
        if c.size() == expected {
            break;
        }
        runtime:sleep(0.1);
    }
    return c.size();
}
