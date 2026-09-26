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

function whileBody() {
    worker w returns int {
        return 1;
    }
    int i = 0;
    while i < 2 {
        int result = wait w; // @error reference inside a loop body
        io:println(result);
        i += 1;
    }
}

function whileCondition() {
    worker w returns int {
        return 1;
    }
    int i = 0;
    while observe(w) && i < 2 { // @error reference inside a loop condition
        i += 1;
    }
}

// A foreach collection, and each side of a range, is evaluated once before the
// loop, so a single reference there is not a repeated one.
function foreachRange() {
    worker w returns int {
        return 1;
    }
    foreach int i in 0 ..< (observe(w) ? 1 : 2) {
        io:println(i);
    }
}

function foreachBody() {
    worker w returns int {
        return 1;
    }
    foreach int _ in 0 ..< 2 {
        int result = wait w; // @error reference inside a foreach body
        io:println(result);
    }
}

function insideClosure() {
    worker w returns int {
        return 1;
    }
    var f = function() returns int {
        return wait w; // @error reference inside an anonymous function
    };
    io:println(f());
}

function observe(future<int> f) returns boolean {
    return f is future<int>;
}

function insideQuerySelect() {
    worker w returns int {
        return 1;
    }
    future<int>[] futures = from int _ in [0, 1]
        select w; // @error reference inside a query select clause
    io:println(futures.length());
}

function insideQueryWhere() {
    worker w returns int {
        return 1;
    }
    int[] results = from int i in [0, 1]
        where observe(w) // @error reference inside a query where clause
        select i;
    io:println(results);
}

function recordFieldDefault() {
    worker w returns int {
        return 1;
    }
    // A record field default is lowered to a closure run at each construction
    // of the record, so a reference in one is a repeated reference.
    record {|future<int> f = w;|} r = {}; // @error reference inside a record field default
    io:println(r);
}
