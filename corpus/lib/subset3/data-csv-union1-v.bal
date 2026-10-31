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

type IntA record {|int a;|};

type StringA record {|string a;|};

type Union (IntA|StringA)[];

type ListUnion (int[]|string[])[];

type RowsOrLists IntA[]|string[][];

type NoMatch (record {|int b;|}|record {|boolean b;|})[];

public function main() {
    // The first member that every row converts to wins.
    io:println(csv:parseString("a\n1\nx", {}, Union));
    io:println(csv:parseString("a\n1\n2", {}, Union));
    io:println(csv:parseString("a\n1\nx", {}, ListUnion));
    io:println(csv:parseString("a\n1", {}, RowsOrLists));
    io:println(csv:parseString("a\nx", {}, RowsOrLists));
    io:println(csv:parseString("a\n1\n2\n3", {skipLines: [2]}, Union));
    Union|csv:Error noMember = csv:parseString("b\n1", {}, Union);
    string prefix = "value 'b\n1' cannot be cast into 'typedesc ";
    io:println(noMember is csv:Error && noMember.message().substring(0, prefix.length()) == prefix);
    io:println(csv:parseString("b\nx", {enableConstraintValidation: false}, NoMatch));
    io:println(csv:parseString("a\n1", {outputWithHeaders: true}, ListUnion));
    io:println(csv:transform([{"a": 1}, {"a": "x"}], {}, Union));
    io:println(csv:transform([{"a": 1}], {}, Union));
    io:println(csv:transform([{"a": 1}], {}, RowsOrLists));
    io:println(csv:parseList([["1"], ["x"]], {customHeaders: ["a"]}, Union));
}

// @output [{"a":"1"},{"a":"x"}]
// @output [{"a":1},{"a":2}]
// @output [["1"],["x"]]
// @output [{"a":1}]
// @output [["x"]]
// @output [{"a":1},{"a":3}]
// @output true
// @output error Error ("source value cannot converted in to the '({| b: int, never... |}|{| b: boolean, never... |})[]'")
// @output [["a"],["1"]]
// @output error Error ("source value cannot converted in to the '({| a: int, never... |}|{| a: string, never... |})[]'")
// @output [{"a":1}]
// @output [{"a":1}]
// @output [{"a":"1"},{"a":"x"}]
