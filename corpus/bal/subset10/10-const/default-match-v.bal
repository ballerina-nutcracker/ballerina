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

type Leaf record {|
    int n = 1;
|};

type R record {|
    int x = 5;
    Leaf leaf = {};
|};

const R A = {};
const R B = {x: 6};

type DefaultR A;

function describe(R r) returns string {
    match r {
        A => {
            return "default";
        }
        _ => {
            return "other";
        }
    }
}

function isDefault(any v) returns boolean {
    return v is DefaultR;
}

public function main() {
    io:println(describe({})); // @output default
    io:println(describe({x: 5, leaf: {n: 1}})); // @output default
    io:println(describe({leaf: {n: 2}})); // @output other
    io:println(isDefault(A)); // @output true
    io:println(isDefault(B)); // @output false
}
