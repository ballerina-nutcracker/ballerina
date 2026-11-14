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

import ballerina/data.csv;
import ballerina/io;

type Ints int[][];

type Strings string[][];

type Anydata anydata[][];

type Pairs [int, string][];

type WithRest [int, string...][];

type Pair2 int[2][];

type Pair3 int[3][];

type Maps map<int>[];

type ReadonlyLists readonly & string[][];

type JsonRows json[][];

public function main() {
    io:println(csv:parseString("a\n1\n-1", {}, Ints));
    io:println(csv:parseString("a\n1.5", {}, Ints));
    // Tuples drop extra columns unless projection is disabled, and pad missing ones.
    io:println(csv:parseString("a,b\n1,x\n2,y", {}, Pairs));
    io:println(csv:parseString("a,b,c\n1,x,9", {}, Pairs));
    io:println(csv:parseString("a,b,c\n1,x,9", {allowDataProjection: false}, Pairs));
    io:println(csv:parseString("a,b,c\n1,x,9", {}, WithRest));
    io:println(csv:parseString("a\n1", {}, WithRest));
    io:println(csv:parseString("a,b\n1", {}, Pairs));
    // `int[2][]` is a closed outer array of two open `int[]` rows; missing rows are filled with `[]`.
    io:println(csv:parseString("a,b,c\n1,2,3", {}, Pair2));
    io:println(csv:parseString("a,b,c\n1,2,3", {allowDataProjection: false}, Pair2));
    io:println(csv:parseString("a,b\n1,2", {}, Pair3));
    // Map rows silently drop values that do not convert.
    io:println(csv:parseString("a,b\n1,zz\n2,3", {}, Maps));
    // outputWithHeaders adds the header row, converted to the element type.
    io:println(csv:parseString("a,b\n1,x", {outputWithHeaders: true}, Strings));
    io:println(csv:parseString("1,b\n1,x", {outputWithHeaders: true}, Anydata));
    io:println(csv:parseString("a,b\n1,x", {outputWithHeaders: true}, Ints));
    // As in jBallerina, a header row with no data rows after it gives no rows.
    io:println(csv:parseString("a,b", {outputWithHeaders: true}, Strings));
    io:println(csv:parseString("a,b\n1,x\n2,y", {outputWithHeaders: true, skipLines: [1]}, Strings));
    io:println(csv:parseString("1,x", {header: (), outputWithHeaders: true, customHeadersIfHeadersAbsent: ["p", "q"]}, Strings));
    io:println(csv:parseString("1,x", {header: (), outputWithHeaders: true}, Strings));
    io:println(csv:parseString("a,b\n1,x", {}, ReadonlyLists));
    io:println(csv:parseString("a,b\n1,true", {}, JsonRows));
}

// @output [[1],[-1]]
// @output error Error ("value '1.5' cannot be cast into 'int'")
// @output [[1,"x"],[2,"y"]]
// @output [[1,"x"]]
// @output error Error ("invalid array size for expected tuple type, cannot be greater than '2'")
// @output [[1,"x","9"]]
// @output [[1]]
// @output [[1,""]]
// @output [[1,2,3],[]]
// @output [[1,2,3],[]]
// @output [[1,2],[],[]]
// @output [{"a":1},{"a":2,"b":3}]
// @output [["a","b"],["1","x"]]
// @output [[1,"b"],[1,"x"]]
// @output error Error ("value 'a' cannot be cast into 'int'")
// @output []
// @output [["a","b"],["2","y"]]
// @output [["p","q"],["1","x"]]
// @output [["1","x"]]
// @output [["1","x"]]
// @output [[1,true]]
