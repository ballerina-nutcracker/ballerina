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

type Ints int[][];

type Pairs [int, string][];

type Maps map<anydata>[];

type Finite ("x"|"y")[][];

type CharRows record {|string:Char a;|}[];

type ByteRows record {|byte a;|}[];

type IntOrString record {|int|string a;|}[];

type NestedField record {|int[] a;|}[];

type BadUnion record {|int|int[] a;|}[];

type Optional record {|int a; int b?;|}[];

type Nilable record {|int a; int? b;|}[];

type Open record {|int a; int...;|}[];

type Closed2 Row[2];

type Pair2 int[2][];

public function main() {
    record {}[] src = [{"a": 1, "b": "x"}, {"a": 2, "b": "y"}];
    io:println(csv:transform(src, {}, Rows));
    io:println(csv:transform(src, {}, Strings));
    io:println(csv:transform(src, {}, Anydata));
    io:println(csv:transform(src, {outputWithHeaders: true}, Anydata));
    io:println(csv:transform(src, {outputWithHeaders: true, headerOrder: ["b", "a"]}, Anydata));
    io:println(csv:transform(src, {headerOrder: ["b"]}, Anydata));
    io:println(csv:transform(src, {}, Pairs));
    io:println(csv:transform(src, {}, Maps));
    io:println(csv:transform(src, {}, Closed2));
    io:println(csv:transform([{"a": 1, "b": "x"}], {}, Closed2));
    io:println(csv:transform([{"a": 1, "b": 2, "c": 3}], {}, Pair2));
    io:println(csv:transform([{"a": 1, "b": 2, "c": 3}], {allowDataProjection: false}, Pair2));
    io:println(csv:transform([{"a": 1, "b": "x", "c": 3}], {allowDataProjection: false}, Pairs));
    // Strings are never converted to numbers, and numbers convert between numeric types.
    io:println(csv:transform([{"a": "1"}], {}, Ints));
    io:println(csv:transform([{"a": 1, "b": 2.6, "c": 3.5d}], {}, Ints));
    io:println(csv:transform([{"a": 2.5}, {"a": 3.5}], {}, Ints));
    io:println(csv:transform([{"a": 300}], {}, ByteRows));
    io:println(csv:transform([{"a": "xy"}], {}, CharRows));
    io:println(csv:transform([{"a": "x"}], {}, CharRows));
    io:println(csv:transform([{"a": "x"}], {}, Finite));
    io:println(csv:transform([{"a": "1"}], {}, IntOrString));
    io:println(csv:transform([{"a": 1}], {}, NestedField));
    io:println(csv:transform([{"a": 1}], {}, BadUnion));
    io:println(csv:transform([{"a": [1]}], {}, Anydata));
    io:println(csv:transform([{"a": true, "b": ()}], {}, Anydata));
    io:println(csv:transform([{"a": 1}, {"a": 1, "c": 2}], {}, Anydata));
    io:println(csv:transform([{"a": 1}, {"c": 2}], {}, Anydata));
    io:println(csv:transform([{"a": 1, "b": "x", "c": 3}], {allowDataProjection: false}, Rows));
    io:println(csv:transform([{"a": 1, "b": "x", "c": 3}], {}, Open));
    io:println(csv:transform([{"a": 1}], {}, Rows));
    io:println(csv:transform([{"a": 1, "b": "null"}], {allowDataProjection: {nilAsOptionalField: true}}, Optional));
    io:println(csv:transform([{"a": 1}], {allowDataProjection: {absentAsNilableType: true}}, Nilable));
}

// @output [{"a":1,"b":"x"},{"a":2,"b":"y"}]
// @output error Error ("value '1' in index '0' is not compatible with array type 'string'")
// @output [[1,"x"],[2,"y"]]
// @output [["a","b"],[1,"x"],[2,"y"]]
// @output [["b","a"],["x",1],["y",2]]
// @output error Error ("invalid number of headers")
// @output [[1,"x"],[2,"y"]]
// @output [{"a":1,"b":"x"},{"a":2,"b":"y"}]
// @output [{"a":1,"b":"x"},{"a":2,"b":"y"}]
// @output error Error ("invalid array size for expected array type, cannot be greater than '1'")
// @output error Error ("invalid array size for expected array type, cannot be greater than '1'")
// @output error Error ("invalid array size for expected array type, cannot be greater than '1'")
// @output error Error ("invalid array size for expected tuple type, cannot be greater than '2'")
// @output error Error ("value '1' in index '0' is not compatible with array type 'int'")
// @output [[1,3,4]]
// @output [[2],[4]]
// @output error {ballerina/lang.value}ConversionError ("'300' value cannot be converted to 'int:Unsigned8'",message="'300' value cannot be converted to 'int:Unsigned8'")
// @output error {ballerina/lang.value}ConversionError ("'\"xy\"' value cannot be converted to 'string:Char'",message="'\"xy\"' value cannot be converted to 'string:Char'")
// @output [{"a":"x"}]
// @output error Error ("value 'x' in index '0' is not compatible with array type '(\"x\"|\"y\")'")
// @output [{"a":"1"}]
// @output error Error ("expected a array type, found 'int[]' ")
// @output [{"a":1}]
// @output error Error ("value '[1]' in index '0' is not compatible with array type 'anydata'")
// @output [[true,null]]
// @output [[1],[1]]
// @output error Error ("CSV data rows with varying headers are not yet supported")
// @output error Error ("no mapping field in the expected type for header 'c'")
// @output [{"a":1,"c":3}]
// @output error Error ("no matching header value is found for the required field 'b'")
// @output [{"a":1}]
// @output [{"a":1,"b":null}]
