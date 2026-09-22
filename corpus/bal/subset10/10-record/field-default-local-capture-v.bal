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

final int offset = 1;

function captureLocal() returns int {
    final int offset = 20;
    record {|string name; int age = offset;|} person = {name: "Pubudu"};
    return person.age;
}

function captureParam(int base) returns int {
    final int offset = 20;
    record {|int total = base + offset;|} sum = {};
    return sum.total;
}

function captureFromNestedBlock(int base) returns int {
    if base > 0 {
        final int inner = 100;
        record {|int value = inner + base;|} rec = {};
        return rec.value;
    }
    return 0;
}

function captureInLambda() returns int {
    final int outerVal = 7;
    var fn = function() returns int {
        final int inner = 5;
        record {|int value = outerVal + inner;|} rec = {};
        return rec.value;
    };
    return fn();
}

public function main() {
    io:println(offset); // @output 1
    io:println(captureLocal()); // @output 20
    io:println(captureParam(5)); // @output 25
    io:println(captureFromNestedBlock(2)); // @output 102
    io:println(captureInLambda()); // @output 12
}
