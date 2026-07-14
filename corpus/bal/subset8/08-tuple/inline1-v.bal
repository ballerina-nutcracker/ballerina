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

type T 1|2|3;

public function main() {
    [int, int] p = [17, 42];
    io:println(p); // @output [17,42]
    [int, string, boolean] t1 = [2, "x", true];
    io:println(t1); // @output [2,"x",true]
    [T, string] t2 = [2, "test"];
    io:println(t2); // @output [2,"test"]
    [int...] t3 = [2, 4, 6];
    io:println(t3); // @output [2,4,6]
    [string, int...] t4 = ["test", 2, 4];
    io:println(t4); // @output ["test",2,4]
}
