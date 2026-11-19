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
import ballerina/test;

type Point record {|
    int x;
    int y;
|};

function eqPasses(anydata actual, anydata expected) returns boolean {
    error? e = trap test:assertEquals(actual, expected);
    return e is ();
}

function notEqPasses(anydata actual, anydata expected) returns boolean {
    error? e = trap test:assertNotEquals(actual, expected);
    return e is ();
}

function exactPasses(any|error actual, any|error expected) returns boolean {
    error? e = trap test:assertExactEquals(actual, expected);
    return e is ();
}

function notExactPasses(any|error actual, any|error expected) returns boolean {
    error? e = trap test:assertNotExactEquals(actual, expected);
    return e is ();
}

public function main() {
    map<int> m1 = {a: 1, b: 2};
    map<int> m2 = {a: 1, b: 2};
    io:println("distinct maps: ", eqPasses(m1, m2), " ", notEqPasses(m1, m2), " ", exactPasses(m1, m2), " ", notExactPasses(m1, m2));
    // @output distinct maps: true false false true

    int[] a1 = [1, 2, 3];
    int[] a2 = [1, 2, 3];
    io:println("distinct arrays: ", eqPasses(a1, a2), " ", exactPasses(a1, a2));
    // @output distinct arrays: true false

    map<int> keyOrder = {b: 2, a: 1};
    io:println("map key order: ", eqPasses(m1, keyOrder), " ", notEqPasses(m1, keyOrder));
    // @output map key order: true false

    int[] reversed = [3, 2, 1];
    io:println("array order: ", eqPasses(a1, reversed), " ", notEqPasses(a1, reversed));
    // @output array order: false true

    Point p = {x: 1, y: 2};
    map<int> pm = {x: 1, y: 2};
    io:println("record vs map: ", eqPasses(p, pm));
    // @output record vs map: true

    float f = 1.0;
    decimal d = 1.0d;
    io:println("float vs decimal: ", eqPasses(f, d), " ", notEqPasses(f, d));
    // @output float vs decimal: false true

    decimal d1 = 1.0d;
    decimal d2 = 1.00d;
    io:println("1.0d vs 1.00d: ", eqPasses(d1, d2), " ", exactPasses(d1, d2), " ", notExactPasses(d1, d2));
    // @output 1.0d vs 1.00d: true false true

    float nan = float:NaN;
    io:println("NaN: ", eqPasses(nan, nan), " ", exactPasses(nan, nan), " ", notEqPasses(nan, nan));
    // @output NaN: true true false

    float negZero = -0.0;
    float posZero = 0.0;
    io:println("-0.0 vs 0.0: ", eqPasses(negZero, posZero), " ", exactPasses(negZero, posZero), " ", notExactPasses(negZero, posZero));
    // @output -0.0 vs 0.0: true false true
}
