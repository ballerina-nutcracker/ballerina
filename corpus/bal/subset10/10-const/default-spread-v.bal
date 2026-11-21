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

isolated function next() returns int {
    return 42;
}

type Const record {|
    int x;
    int y = 5;
|};

type Runtime record {|
    int x;
    int y = next();
|};

type Open record {
    int x;
    int y = next();
};

type MaybeY record {|
    int x;
    int y?;
|};

const map<int> X = {x: 1};
const map<int> XY = {x: 1, y: 9};

const Const A = {...X};
const Const B = {...XY};
const Runtime C = {...XY};
const Runtime D = {...X, y: 3};
const Open E = {...<MaybeY>XY};

public function main() {
    io:println(A); // @output {"x":1,"y":5}
    io:println(B); // @output {"x":1,"y":9}
    io:println(C); // @output {"x":1,"y":9}
    io:println(D); // @output {"x":1,"y":3}
    io:println(E); // @output {"x":1,"y":9}
    io:println(B is readonly & record {|1 x; 9 y;|}); // @output true
}
