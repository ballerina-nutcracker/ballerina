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

function one() returns int {
    return 1;
}

// A worker and a future created by a start action can be waited on together.
// The two field types differ: a worker is referenced exactly once, so its
// value cannot have been claimed already, while a start future can have been,
// and carries the error possibility.
function multiple() {
    worker w returns int {
        return 2;
    }
    future<int> f = start one();
    record {|int w; int|error f;|} both = wait {w, f};
    io:println(both.w); // @output 2
    io:println(both.f); // @output 1
}

// Both alternates produce the same value, so the result does not depend on
// which one the scheduler completes first.
function alternate() {
    worker w returns int {
        return 1;
    }
    future<int> f = start one();
    int|error first = wait w | f;
    io:println(first); // @output 1
}

// The start future is claimed before the wait, so only the worker arm of the
// multiple wait can still produce a value.
function claimed() {
    worker w returns int {
        return 2;
    }
    future<int> f = start one();
    int|error claim = wait f;
    io:println(claim); // @output 1
    record {|int w; int|error f;|} both = wait {w, f};
    io:println(both.w); // @output 2
    io:println(both.f); // @output error("multiple waits on the same future is not allowed")
}

public function main() {
    multiple();
    alternate();
    claimed();
}
