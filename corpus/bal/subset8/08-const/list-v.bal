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

const BASE = 10;
const A = [1, 2, 3];
const B = ["hello", "world"];
const C = [1, "two", 3];
const D = [[1, 2], [3, 4]];
const E = [BASE, ...[BASE + 1, -3]];

public function main() {
    io:println(A); // @output [1,2,3]
    io:println(B); // @output ["hello","world"]
    io:println(C); // @output [1,"two",3]
    io:println(D); // @output [[1,2],[3,4]]
    io:println(E); // @output [10,11,-3]
}
