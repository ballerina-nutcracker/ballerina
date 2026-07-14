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

type F function (int) returns int;

function foo(int x) returns int {
    return x + 1 + 2;
}

function bench(int base) returns int {
    F f = foo;
    int sum = 0;
    foreach int i in 0 ..< 10000000 {
        final int v = i;
        if i % 2 == 0 {
            f = function(int x) returns int {
                return v + x + base;
            };
        }
        else {
            f = foo;
        }
        sum += f(base);
    }
    return sum;
}

public function main() {
    io:println(bench(10)); // @output 25000160000000
}

