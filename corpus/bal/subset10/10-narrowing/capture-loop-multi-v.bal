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

// Several closures in a loop capture a and b, but c is never captured, so it
// still narrows inside the loop and after it.
public function main() {
    int|string a = 1;
    int|string b = 2;
    int|string c = 3;
    int total = 0;
    foreach int i in 0 ..< 3 {
        function () setA = function () {
            a = i;
        };
        function () returns int|string getB = function () returns int|string {
            return b;
        };
        setA();
        io:println(getB()); // @output 2
                            // @output 2
                            // @output 2
        if c is int {
            total += c;
        }
    }
    if c is int {
        total += c;
    }
    io:println(total); // @output 12
    io:println(a); // @output 2
}
