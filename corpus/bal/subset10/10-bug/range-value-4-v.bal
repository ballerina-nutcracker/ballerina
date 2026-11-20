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

// A range expression is always isolated (spec 6.26): it can initialize an
// isolated variable or be assigned to a field of an isolated object, and the
// iterator() and next() methods of its value can be called from an isolated
// function.
isolated object:Iterable range = 1 ..< 3;

isolated class Holder {
    private object:Iterable r;

    isolated function init() {
        self.r = 1 ... 2;
    }

    isolated function reset() {
        lock {
            self.r = 4 ... 5;
        }
    }
}

isolated function sum(int n) returns int {
    var r = 1 ... n;
    var it = r.iterator();
    int total = 0;
    var next = it.next();
    while next !is () {
        total += next.value;
        next = it.next();
    }
    return total;
}

isolated function count(int n) returns int {
    int c = 0;
    foreach int _ in 0 ..< n {
        c += 1;
    }
    return c;
}

public function main() {
    io:println(sum(4)); // @output 10
    io:println(count(3)); // @output 3
    lock {
        io:println(range is object:Iterable); // @output true
    }
    Holder h = new;
    h.reset();
}
