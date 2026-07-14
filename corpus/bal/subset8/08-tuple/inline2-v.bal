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

type T1 1|2|3;

type T2 string|true;

public function main() {
    [T1] p = [1];
    io:println(p); // @output [1]
    [int] t1 = [21];
    io:println(t1); // @output [21]
    [T2...] t2 = [true, "test", true];
    io:println(t2); // @output [true,"test",true]
    [(int|float)] t3 = [2.5];
    io:println(t3); // @output [2.5]
    [(int|float)...] t4 = [4.5, 2, 4];
    io:println(t4); // @output [4.5,2,4]
    [(T1|float)] t5 = [2.5];
    io:println(t5); // @output [2.5]
    [(T1|float)...] t6 = [4.5, 2, 1];
    io:println(t6); // @output [4.5,2,1]
    [(T1|float)] t7 = [2];
    io:println(t7); // @output [2]
    [(T1|float), string] t8 = [2, "test"];
    io:println(t8); // @output [2,"test"]
    [(T1|float), string...] t9 = [2, "test1", "test2"];
    io:println(t9); // @output [2,"test1","test2"]
    [(T1|T2)] t10 = ["test"];
    io:println(t10); // @output ["test"]
    [(T1|T2)...] t11 = ["test", 2];
    io:println(t11); // @output ["test",2]
}
