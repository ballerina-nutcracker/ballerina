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
    int[] values = [3, 1, 2];
    values.unshift(0);
    io:println(values); // @output [0,3,1,2]
    io:println(values.pop()); // @output 2
    io:println(values.shift()); // @output 0
    io:println(values); // @output [3,1]

    values.unshift();
    values.unshift(7, 8);
    io:println(values); // @output [7,8,3,1]

    string[] queue = [];
    queue.push("a", "b");
    queue.unshift("z");
    io:println(queue.shift(), queue.shift(), queue.pop()); // @output zab
    io:println(queue.length()); // @output 0

    [int, string...] t = [1, "a", "b"];
    int|string last = t.pop();
    io:println(last); // @output b
    io:println(t); // @output [1,"a"]

    [int, int...] u = [1, 2];
    io:println(u.shift()); // @output 1
    u.unshift(5);
    io:println(u); // @output [5,2]
}
