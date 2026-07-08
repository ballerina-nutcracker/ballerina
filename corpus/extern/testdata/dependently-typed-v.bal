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

type Point record {| int x; int y; |};

public function main() {
    int a = inferred(0);
    io:println(a); // @output 1
    string b = inferred(1);
    io:println(b); // @output foo

    int x = inferredSubType(0);
    io:println(x); // @output 1

    1 y = inferredSubType(0);
    io:println(y); // @output 1

    int|error d = inferredPartially(0);
    io:println(d); // @output 0

    string|error e = inferredPartially(0);
    io:println(e); // @output bar

    Point p1 = {x: 1, y: 2};
    Point p2 = shiftBy(p1, 10, 20);
    io:println(p2.x); // @output 11
    io:println(p2.y); // @output 22

    // Call using named arguments.
    int aNamed = inferred(val = 7);
    io:println(aNamed); // @output 1
    Point p3 = shiftBy(dy = 100, p = p1, dx = 5);
    io:println(p3.x); // @output 6
    io:println(p3.y); // @output 102

    // Required param has a default value, so it can be omitted or supplied by name.
    int defA = inferredWithDefault();
    io:println(defA); // @output 42
    int defB = inferredWithDefault(val = 100);
    io:println(defB); // @output 100
    string defC = inferredWithDefault(val = 7);
    io:println(defC); // @output 7
}

function inferred(int val, typedesc retTy = <>) returns retTy = external;

function inferredSubType(int val, typedesc<int> retTy = <>) returns retTy = external;

function inferredPartially(int val, typedesc<anydata> retTy = <>) returns retTy|error = external;

function shiftBy(Point p, int dx, int dy, typedesc retTy = <>) returns retTy = external;

function inferredWithDefault(int val = 42, typedesc retTy = <>) returns retTy = external;
