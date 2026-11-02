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

const int MIN = 1;
const int MAX = 10;

const true ORDERED = MIN < MAX;
const false REVERSED = MIN >= MAX;

enum Color {
    RED,
    GREEN
}

function ordered() returns true {
    return MIN < MAX;
}

function one() returns 1 {
    return 1;
}

function pickIf() returns int {
    if MIN < MAX {
        return 1;
    }
}

function pickWhile() returns int {
    while MIN <= MAX {
        return 2;
    }
}

function param(1 x) returns true {
    return x < 2;
}

function narrowed(int v) returns boolean {
    if v is 1 {
        true t = v < 2;
        return t;
    }
    return false;
}

public function main() {
    io:println(ORDERED, REVERSED); // @output truefalse
    io:println(ordered()); // @output true
    io:println(pickIf()); // @output 1
    io:println(pickWhile()); // @output 2
    io:println(param(1)); // @output true
    io:println(narrowed(1)); // @output true

    boolean b = narrowed(2);
    true a = MIN < MAX && MAX > MIN;
    io:println(a); // @output true
    false c = MIN > MAX && b;
    io:println(c); // @output false
    true d = MIN < MAX || b;
    io:println(d); // @output true

    true fromCall = one() <= 1;
    io:println(fromCall); // @output true
    true enumEq = RED == "RED";
    io:println(enumEq); // @output true
    false enumLt = RED < "GREEN";
    io:println(enumLt); // @output false

    true nested = (1 < 2) == (3 < 4);
    io:println(nested); // @output true
    true nestedOrder = (1 > 2) < (3 < 4);
    io:println(nestedOrder); // @output true
    true negative = -1 < 0;
    io:println(negative); // @output true
    true sum = 1 + 1 == 2;
    io:println(sum); // @output true
    true concat = "a" + "b" >= "ab";
    io:println(concat); // @output true

    any v = MIN < MAX;
    io:println(v is true); // @output true
}
