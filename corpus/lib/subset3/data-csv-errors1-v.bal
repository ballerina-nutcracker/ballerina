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

type StringRow record {|string a;|};

type StringRows StringRow[];

type MixedUnion record {|int[]|string a;|}[];

type Union (record {|int a;|}|record {|string a;|})[];

public function main() {
    // Malformed input is reported with its line and column.
    io:println(csv:parseString("a,b\n1,C:\\x", {}, Rows));
    io:println(csv:parseString("a,b\n1,\\u00zz", {}, Rows));
    io:println(csv:parseString("\\q,b\n1,2", {}, Strings));
    io:println(csv:parseString("a,b\n", {}, Rows));
    // A missing or empty header row is an error.
    io:println(csv:parseString("", {}, Rows));
    io:println(csv:parseString("\na,b\n1,x", {}, Rows));
    io:println(csv:parseString("# c\na,b\n1,x", {}, Rows));
    io:println(csv:parseString("a,,b\n1,2,3", {}, Strings));
    io:println(csv:parseString("a,b\n1,x", {header: 5}, Rows));
    io:println(csv:parseString("a,b", {header: ()}, Rows));
    // A quoted header row that ends the input leaves a row that was never created.
    io:println(csv:parseString("\"a\",\"b\"", {}, Strings));
    Strings|error headerOnly = csv:parseString("\"a\",\"b\"", {}, Strings);
    if headerOnly is error {
        io:println(headerOnly.message());
    }
    // Values that do not fit the target are reported with their value.
    io:println(csv:transform([{"a": 2.5}], {}, StringRows));
    io:println(csv:transform([{"a": 1234.5678d}], {}, StringRows));
    io:println(csv:transform([{"a": 1234567}], {}, StringRows));
    // Floats are grouped in full and rounded half-even on their exact binary value.
    io:println(csv:transform([{"a": 1.0e20}], {}, StringRows));
    io:println(csv:transform([{"a": -float:Infinity}], {}, StringRows));
    io:println(csv:transform([{"a": 0.0005}], {}, StringRows));
    io:println(csv:transform([{"a": 0.1235}], {}, StringRows));
    io:println(csv:transform([{"a": true}], {}, MixedUnion));
    io:println(csv:transform([{"a": "x"}], {}, MixedUnion));
    // With failSafe, a union member that fails on a row is logged to the log file and the next member is tried.
    io:println(csv:transform([{"a": 1}, {"a": "x"}], {failSafe: {enableConsoleLogs: false}}, Union));
}

// @output error Error ("error while reading the csv data 'expected 'escaped characters'' in line: '2', column: '6'")
// @output error Error ("error while reading the csv data 'expected 'hexadecimal value of an unicode character'' in line: '2', column: '7'")
// @output error Error ("error while reading the csv data 'expected 'escaped characters'' in line: '1', column: '2'")
// @output error Error ("error while reading the csv data 'Invalid token found' in line: '2', column: '0'")
// @output error Error ("provided header row is empty")
// @output error Error ("provided header row is empty")
// @output error Error ("provided header row is empty")
// @output error Error ("provided header row is empty")
// @output error Error ("provided header row is empty")
// @output error Error ("no matching header value is found for the required field 'a'")
// @output error("{ballerina/lang.array}InherentTypeViolation",message="incompatible types: expected 'string[]', found '()'")
// @output {ballerina/lang.array}InherentTypeViolation
// @output error Error ("no mapping type found for value '2.5' in key 'a'")
// @output error Error ("no mapping type found for value '1234.5678' in key 'a'")
// @output error Error ("no mapping type found for value '1,234,567' in key 'a'")
// @output error Error ("no mapping type found for value '100,000,000,000,000,000,000' in key 'a'")
// @output error Error ("no mapping type found for value '-∞' in key 'a'")
// @output error Error ("no mapping type found for value '0.001' in key 'a'")
// @output error Error ("no mapping type found for value '0.123' in key 'a'")
// @output error Error ("expected type cannot contains types other than basic types, 'int[]' not allowed")
// @output [{"a":"x"}]
// @output [{"a":1}]
