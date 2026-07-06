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

// The second `while` loop declares a mutable loop-local `x` and an
// isolated lambda that captures it. Capturing a mutable variable from
// an enclosing scope inside an isolated lambda is not allowed, so the
// reference to `x` inside the lambda must be rejected. The first loop
// (with `final int x`) is included to show that the same-named final
// capture in a sibling scope remains valid.
isolated function foo(int n) returns int {
    int sum = 0;
    int i = 0;
    while i < n {
        final int x = i * 2;
        i = i + 1;

        var f = isolated function() returns int {
            return x + 1;
        };
        sum += f();
    }
    while i < n {
        int x = i * 2;
        i = i + 1;

        var f = isolated function() returns int {
            return x + 1; // @error
        };
        sum += f();
    }

    return sum;
}

public function main() {
    io:println(foo(3));
}
