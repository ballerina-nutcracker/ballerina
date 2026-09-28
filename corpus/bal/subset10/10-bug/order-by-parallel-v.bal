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

function sorted0(int[] xs) returns int[] {
    return from int x in xs order by x descending select x;
}

function list0() returns int[] {
    int[] xs = [0, 1, 2];
    return xs;
}

function sorted1(int[] xs) returns int[] {
    return from int x in xs order by x descending select x;
}

function list1() returns int[] {
    int[] xs = [1, 1, 2];
    return xs;
}

function sorted2(int[] xs) returns int[] {
    return from int x in xs order by x descending select x;
}

function list2() returns int[] {
    int[] xs = [2, 1, 2];
    return xs;
}

function sorted3(int[] xs) returns int[] {
    return from int x in xs order by x descending select x;
}

function list3() returns int[] {
    int[] xs = [3, 1, 2];
    return xs;
}

function sorted4(int[] xs) returns int[] {
    return from int x in xs order by x descending select x;
}

function list4() returns int[] {
    int[] xs = [4, 1, 2];
    return xs;
}

function sorted5(int[] xs) returns int[] {
    return from int x in xs order by x descending select x;
}

function list5() returns int[] {
    int[] xs = [5, 1, 2];
    return xs;
}

function sorted6(int[] xs) returns int[] {
    return from int x in xs order by x descending select x;
}

function list6() returns int[] {
    int[] xs = [6, 1, 2];
    return xs;
}

function sorted7(int[] xs) returns int[] {
    return from int x in xs order by x descending select x;
}

function list7() returns int[] {
    int[] xs = [7, 1, 2];
    return xs;
}

function sorted8(int[] xs) returns int[] {
    return from int x in xs order by x descending select x;
}

function list8() returns int[] {
    int[] xs = [8, 1, 2];
    return xs;
}

function sorted9(int[] xs) returns int[] {
    return from int x in xs order by x descending select x;
}

function list9() returns int[] {
    int[] xs = [9, 1, 2];
    return xs;
}

function sorted10(int[] xs) returns int[] {
    return from int x in xs order by x descending select x;
}

function list10() returns int[] {
    int[] xs = [10, 1, 2];
    return xs;
}

function sorted11(int[] xs) returns int[] {
    return from int x in xs order by x descending select x;
}

function list11() returns int[] {
    int[] xs = [11, 1, 2];
    return xs;
}

public function main() {
    io:println(sorted0(list0())); // @output [2,1,0]
    io:println(sorted1(list1())); // @output [2,1,1]
    io:println(sorted2(list2())); // @output [2,2,1]
    io:println(sorted3(list3())); // @output [3,2,1]
    io:println(sorted4(list4())); // @output [4,2,1]
    io:println(sorted5(list5())); // @output [5,2,1]
    io:println(sorted6(list6())); // @output [6,2,1]
    io:println(sorted7(list7())); // @output [7,2,1]
    io:println(sorted8(list8())); // @output [8,2,1]
    io:println(sorted9(list9())); // @output [9,2,1]
    io:println(sorted10(list10())); // @output [10,2,1]
    io:println(sorted11(list11())); // @output [11,2,1]
}
