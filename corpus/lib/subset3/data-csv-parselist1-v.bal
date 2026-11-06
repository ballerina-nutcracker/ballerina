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

type Row record {|int a; string b;|};

type Rows Row[];

type Strings string[][];

type Anydata anydata[][];

type Pairs [int, string][];

type Maps map<int>[];

type Open record {int a;}[];

public function main() {
    string[][] lst = [["a", "b"], ["1", "x"], ["2", "y"]];
    io:println(csv:parseList(lst, {}, Strings));
    io:println(csv:parseList(lst, {headerRows: 1}, Rows));
    io:println(csv:parseList(lst, {headerRows: 1, outputWithHeaders: true}, Strings));
    io:println(csv:parseList(lst, {headerRows: 1}, Anydata));
    io:println(csv:parseList(lst, {headerRows: 1, outputWithHeaders: true}, Pairs));
    io:println(csv:parseList([["1", "x"]], {customHeaders: ["a", "b"]}, Rows));
    io:println(csv:parseList([["1", "x"]], {customHeaders: ["a", "b"], outputWithHeaders: true}, Strings));
    io:println(csv:parseList(lst, {customHeaders: ["a", "b"]}, Rows));
    io:println(csv:parseList(lst, {headerRows: 2}, Rows));
    io:println(csv:parseList([["h"], ["a", "b"], ["1", "x"]], {headerRows: 2, customHeaders: ["a", "b"]}, Rows));
    io:println(csv:parseList([["1", "x"], ["2"]], {customHeaders: ["a", "b"]}, Rows));
    io:println(csv:parseList([["1", "x", "3"]], {customHeaders: ["a", "b", "c"]}, Rows));
    io:println(csv:parseList([["1", "x"]], {customHeaders: ["p", "q"]}, Rows));
    io:println(csv:parseList([["1", "x"]], {customHeaders: ["a"]}, Rows));
    io:println(csv:parseList([["1", "zz"]], {customHeaders: ["a", "b"]}, Maps));
    io:println(csv:parseList([["1", "x"]], {}, Open));
    io:println(csv:parseList([[" 1", "x"]], {}, Pairs));
    io:println(csv:parseList([["1", "x", "y"]], {allowDataProjection: false}, Pairs));
    io:println(csv:parseList(lst, {headerRows: 1, skipLines: [1]}, Rows));
}

// @output [["a","b"],["1","x"],["2","y"]]
// @output [{"a":1,"b":"x"},{"a":2,"b":"y"}]
// @output [["a","b"],["1","x"],["2","y"]]
// @output [[1,"x"],[2,"y"]]
// @output error Error ("value 'a' in index '0' is not compatible with array type 'int'")
// @output [{"a":1,"b":"x"}]
// @output [["a","b"],["1","x"]]
// @output error Error ("no mapping type found for value 'a' in key 'a'")
// @output error Error ("custom headers should be provided")
// @output [{"a":1,"b":"x"}]
// @output error Error ("CSV data rows with varying headers are not yet supported")
// @output [{"a":1,"b":"x"}]
// @output error Error ("value '[\"1\",\"x\"]' cannot be cast into '{| a: int, b: string, never... |}', because fields in '{| a: int, b: string, never... |}' or the provided expected headers are not matching with the '[\"1\",\"x\"]'")
// @output error Error ("invalid number of headers")
// @output [{"a":1}]
// @output error Error ("no matching header value is found for the required field 'a'")
// @output error Error ("value ' 1' in index '0' is not compatible with array type 'int'")
// @output error Error ("invalid array size for expected tuple type, cannot be greater than '2'")
// @output [{"a":2,"b":"y"}]
