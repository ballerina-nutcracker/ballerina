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
// KIND, either express or implied. See the License for the
// specific language governing permissions and limitations
// under the License.

import ballerina/io;

string trace = "";
function mark(string label, int value) returns int {
    trace += label;
    return value;
}
function add(int a, int b) returns int { return a + b; }

public function main() {
    boolean flag = false;
    int arithmetic = 10 + (flag ? mark("x", 1) : mark("y", 2));
    int[] values = [flag ? 3 : 4, 5];
    int indexed = [flag ? 6 : 7][0];
    map<int> mapped = {value: flag ? 8 : 9};
    int associated = flag ? 1 : true ? 2 : 3;
    int called = add(mark("a", 10), flag ? mark("b", 20) : mark("c", 30));
    io:println(arithmetic); // @output 12
    io:println(values[0]); // @output 4
    io:println(indexed); // @output 7
    io:println(mapped["value"]); // @output 9
    io:println(associated); // @output 2
    io:println(called); // @output 40
    io:println(trace); // @output yac
}
