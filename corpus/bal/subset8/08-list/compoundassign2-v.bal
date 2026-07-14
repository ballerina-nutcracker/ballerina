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

public function main() {
    int[] x = [2, 100];
    x[1] >>= x[1] >> 4;
    io:println(x[1]); // @output 1
    x[1] += 33 * 3;
    io:println(x[1]); // @output 100
    x[1] -= x[1] * x[1];
    io:println(x[1]); // @output -9900
    string[] s = ["hello"];
    s[0] += " world";
    io:println(s[0]); // @output hello world
    x[1] = 2;
    x[func1()] += 3; // @output func1
    io:println(x[1]); // @output 5
    x[(4 - 3)] -= 2;
    io:println(x[1]); // @output 3
    x[func1()] += func2(); // @output func1
    // @output func2
    io:println(x[1]); // @output 5
}

function func1() returns int {
    io:println("func1");
    return 1;
}

function func2() returns int {
    io:println("func2");
    return 2;
}
