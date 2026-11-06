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

// The expressions of limit and collect clauses are evaluated once per query,
// so a single reference to a worker in one is not a repeated reference.
function observe(future<int> f) returns int {
    return f is future<int> ? 1 : 2;
}

function limitAfterFrom() {
    worker w returns int {
        return 1;
    }
    int[] r = from int i in [10, 20, 30]
        limit observe(w)
        select i;
    io:println(r); // @output [10]
}

function limitAfterJoin() {
    worker w returns int {
        return 1;
    }
    int[] r = from int i in [1, 2, 3]
        join int j in [1, 2, 3] on i equals j
        where i > 1
        limit observe(w)
        select i;
    io:println(r); // @output [2]
}

function inCollect() {
    worker w returns int {
        return 1;
    }
    int total = from int i in [1, 2, 3]
        collect [i].length() + observe(w);
    io:println(total); // @output 4
}

public function main() {
    limitAfterFrom();
    limitAfterJoin();
    inCollect();
}
