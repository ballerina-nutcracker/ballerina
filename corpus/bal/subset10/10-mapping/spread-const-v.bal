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

const map<int> C = {"a": 1};
const map<int> D = {...C, "b": 2};

type Point record {|
    int x;
|};

const Point ORIGIN = {x: 0};
const map<int> SPREAD = {...ORIGIN};

public function main() {
    io:println(D); // @output {"a":1,"b":2}
    io:println(SPREAD); // @output {"x":0}
    io:println(D.a); // @output 1
    io:println(D["b"]); // @output 2
    io:println(D is record {|1 a; 2 b;|}); // @output true
    io:println(D is readonly); // @output true
}
