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

type A record {|
    int x;
|};

type B record {|
    string x;
|};

type R record {|
    A|B f = {x: "s"};
    A|B g = {x: name()};
    A|B h = {x: S};
    A|B i = {x: n};
|};

isolated function name() returns string {
    return "s";
}

final int n = 1;

const S = "c";

public function main() {
    R r = {};
    io:println(r.f is B); // @output true
    io:println(r.g is B); // @output true
    io:println(r.h is B); // @output true
    io:println(r.i is A); // @output true
}
