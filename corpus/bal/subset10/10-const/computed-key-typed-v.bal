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

const S = "s";

type R record {|
    int a;
    string...;
|};

type A record {|
    int a;
|};

type B record {|
    int b;
    int...;
|};

type AB A|B;

type D record {
    int a = 5;
    int b = 6;
};

// The computed value is checked against the rest type of R.
const R X = {a: 1, [S]: "t"};
// A computed key that supplies a defaulted field replaces the default.
const D Y = {["a"]: 1};
// Only the specific field b selects the member of the union, so B is the
// inherent type and the computed field a goes to its rest type.
const AB Z = {b: 1, ["a"]: 2};

public function main() {
    io:println(X); // @output {"a":1,"s":"t"}
    io:println(Y); // @output {"a":1,"b":6}
    io:println(Z); // @output {"b":1,"a":2}
}
