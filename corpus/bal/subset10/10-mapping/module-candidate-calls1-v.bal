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

type C record {|
    function () returns int x;
|};

type D record {|
    Counter x;
|};

type E record {|
    function () returns string x;
|};

class Counter {
    int n = 0;
}

A|B g1 = {x: name()};

A|B g2 = {x: "abc".length()};

C|E g3 = {x: function() returns int { return 1; }};

D|B g4 = {x: new};

function name() returns string {
    return "s";
}

public function main() {
    io:println(g1 is B); // @output true
    io:println(g2 is A); // @output true
    io:println(g3 is C); // @output true
    io:println(g4 is D); // @output true
}
