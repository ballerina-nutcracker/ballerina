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

type P record {|
    int x = 1;
    int y = 2;
|};

type Q record {|
    string s = "q";
    boolean f = true;
|};

const (P|Q) A = {x: 3};
const (P|Q) B = {y: 4};
const (P|Q) C = {s: "z"};

public function main() {
    io:println(A); // @output {"x":3,"y":2}
    io:println(A is P); // @output true
    io:println(B); // @output {"y":4,"x":1}
    io:println(C); // @output {"s":"z","f":true}
    io:println(C is Q); // @output true
}
