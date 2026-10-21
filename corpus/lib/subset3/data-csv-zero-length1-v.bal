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

type NoRows int[0][];

type EmptyRows int[][0];

type NoRecords record {}[0];

type NoRequired record {int a;}[0];

type NoChoices (record {int a;}|map<string>)[0];

type NoReadonlyRecords (readonly & record {int a;})[0];

type NoStrictChoices (record {int a;}|record {|int b;|})[0];

type NoIntChoices (record {int a;}|map<int>)[0];

public function main() {
    string csv = "a,b\n1,2\n3,4";
    // A closed result array of length 0 holds no rows.
    io:println(csv:parseString(csv, {}, NoRows));
    io:println(csv:parseString(csv, {allowDataProjection: false}, NoRows));
    io:println(csv:parseList([["1", "2"]], {}, NoRows));
    io:println(csv:transform([{"a": 1}], {}, NoRecords));
    io:println(csv:parseString(csv, {}, NoRecords));
    // The input is still parsed, so its errors are still reported.
    io:println(csv:parseString(csv, {skipLines: "2-1"}, NoRows));
    // Every row is still converted to the element type, so a missing column or a bad value is an error.
    io:println(csv:parseString("b\n1", {}, NoRequired));
    io:println(csv:parseString("a\n1\nx", {}, NoRequired));
    io:println(csv:parseBytes("a\nx".toBytes(), {}, NoRequired));
    io:println(csv:parseString("a\nx", {}, NoRows));
    io:println(csv:parseString("a\n1", {}, NoRequired));
    // transform and parseList do not look at the rows.
    io:println(csv:transform([{"b": 1}], {}, NoRequired));
    io:println(csv:parseList([["x"]], {customHeaders: ["a"]}, NoRequired));
    // A union or read-only element type holds no rows either (ballerina-platform/ballerina-library#9809).
    io:println(csv:parseString("a\n1", {}, NoChoices));
    io:println(csv:parseString("a\n1", {enableConstraintValidation: false}, NoChoices));
    io:println(csv:transform([{"a": 1}], {}, NoChoices));
    io:println(csv:parseList([["1"]], {customHeaders: ["a"]}, NoChoices));
    io:println(csv:parseString("a\n1", {}, NoReadonlyRecords));
    io:println(csv:transform([{"a": 1}], {}, NoReadonlyRecords));
    // A row that no member of a union element type fits is still an error.
    io:println(csv:parseString("a\nx", {}, NoStrictChoices));
    io:println(csv:parseString("a\nx", {enableConstraintValidation: false}, NoStrictChoices));
    io:println(csv:parseBytes("a\nx".toBytes(), {}, NoStrictChoices));
    // A row that some member fits gives no rows (ballerina-platform/ballerina-library#9809).
    io:println(csv:parseString("a\n1", {}, NoStrictChoices));
    io:println(csv:parseString("a\nx", {}, NoIntChoices));
    // A row of length 0 takes no values: projection drops them, and without projection they are an error.
    io:println(csv:parseList([["1", "2"], ["3"]], {}, EmptyRows));
    [][]|error emptyTuples = csv:parseList([["1", "2"]]);
    io:println(emptyTuples);
    io:println(csv:transform([{"a": 1}], {}, EmptyRows));
    io:println(csv:parseString(csv, {allowDataProjection: false}, EmptyRows));
    io:println(csv:transform([{"a": 1}], {allowDataProjection: false}, EmptyRows));
}

// @output []
// @output []
// @output []
// @output []
// @output []
// @output error Error ("invalid range for the skipLines field. Start value must be less than or equal to end value.")
// @output error Error ("no matching header value is found for the required field 'a'")
// @output error Error ("value 'x' cannot be cast into 'int'")
// @output error Error ("value 'x' cannot be cast into 'int'")
// @output error Error ("value 'x' cannot be cast into 'int'")
// @output []
// @output []
// @output []
// @output []
// @output []
// @output []
// @output []
// @output []
// @output []
// @output error Error ("value 'a\nx' cannot be cast into 'typedesc ({| a: int, anydata... |}|{| b: int, never... |})[0]'")
// @output error Error ("source value cannot converted in to the '({| a: int, anydata... |}|{| b: int, never... |})[0]'")
// @output error Error ("value '[97,10,120]' cannot be cast into 'typedesc ({| a: int, anydata... |}|{| b: int, never... |})[0]'")
// @output []
// @output []
// @output [[],[]]
// @output [[]]
// @output [[]]
// @output error Error ("invalid array size for expected array type, cannot be greater than '0'")
// @output error Error ("invalid array size for expected array type, cannot be greater than '0'")
