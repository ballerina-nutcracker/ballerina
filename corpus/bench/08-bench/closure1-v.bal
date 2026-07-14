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
    return x + 1;
}

function genFs(int base) returns F[] {
    F[] array = [];
    foreach int i in 0 ..< 10000000 {
        final int v = i;
        if i % 2 == 0 {
            F f = function(int x) returns int {
                return v + x + base;
            };
            array.push(f);
        }
        else {
            array.push(foo);
        }
    }
    return array;
}

function fSum(F[] funcs) returns int {
    int sum = 0;
    foreach int i in 0 ..< funcs.length() {
        F f = funcs[i];
        sum += f(i);
    }
    return sum;
}

public function main() {
    F[] funcs = genFs(10);
    io:println(fSum(funcs)); // @output 75000045000000
}

