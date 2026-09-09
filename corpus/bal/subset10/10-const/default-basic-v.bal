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

type One record {|
    int x = 5;
|};

type Many record {|
    int x = 5;
    string s = "a";
    boolean b = true;
|};

type Mixed record {|
    int x;
    string s = "a";
|};

const One A = {};
const Many B = {};
const Many C = {s: "z"};
const Many D = {x: 1, s: "z", b: false};
const Mixed E = {x: 7};

public function main() {
    io:println(A); // @output {"x":5}
    io:println(A.x); // @output 5
    io:println(A is One); // @output true
    io:println(A is readonly); // @output true
    io:println(B); // @output {"x":5,"s":"a","b":true}
    io:println(C); // @output {"s":"z","x":5,"b":true}
    io:println(D); // @output {"x":1,"s":"z","b":false}
    io:println(E); // @output {"x":7,"s":"a"}
    io:println(E is Mixed); // @output true
    io:println(A is readonly & record {|5 x;|}); // @output true
}
