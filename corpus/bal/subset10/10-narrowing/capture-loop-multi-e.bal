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

// Each closure in the loop body adds its own captures to the loop's group. A
// captured variable is never narrowed anywhere in the loop, while a variable no
// closure captures still narrows.
public function main() {
    int|string a = 1;
    int|string b = 2;
    int|string c = 3;
    int i = 0;
    while i < 2 {
        if a is int {
            int y = a; // @error a is captured by the first closure
            io:println(y);
        }
        if b is int {
            int y = b; // @error b is captured by the second closure
            io:println(y);
        }
        if c is int {
            int y = c;
            io:println(y);
        }
        function () setA = function () {
            a = "a";
        };
        function () setB = function () {
            b = "b";
        };
        setA();
        setB();
        i += 1;
    }
}
