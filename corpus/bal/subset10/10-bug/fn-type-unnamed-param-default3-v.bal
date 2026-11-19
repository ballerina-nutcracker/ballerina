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

type UnnamedDefaultBeforeNamed function (int = 3, int b = 4) returns int;

type UnnamedBeforeUnnamedDefault function (int, int = 3) returns int;

type AllUnnamedDefaults function (int = 3, int = 4) returns int;

type UnnamedWithRest function (int, int b = 3, int... r) returns int;

type UnnamedInMiddle function (int a, int, int c = 3) returns int;

type DefaultUsesNamedAfterUnnamed function (int, int a, int b = a + 1) returns int;

type TwoUnnamedBeforeDefault function (int, int, int c = 5) returns int;

function add(int x, int y) returns int => x + y;

function digits(int x, int y, int z) returns int => x * 100 + y * 10 + z;

function addRest(int x, int y, int... r) returns int => x + y + r.length();

public function main() {
    UnnamedDefaultBeforeNamed f1 = add;
    io:println(f1()); // @output 7
    io:println(f1(1)); // @output 5

    UnnamedBeforeUnnamedDefault f2 = add;
    io:println(f2(1)); // @output 4

    AllUnnamedDefaults f3 = add;
    io:println(f3()); // @output 7

    UnnamedWithRest f4 = addRest;
    io:println(f4(1)); // @output 4

    UnnamedInMiddle f5 = digits;
    io:println(f5(1, 2)); // @output 123

    DefaultUsesNamedAfterUnnamed f6 = digits;
    io:println(f6(1, 2)); // @output 123

    TwoUnnamedBeforeDefault f7 = digits;
    io:println(f7(1, 2)); // @output 125
}
