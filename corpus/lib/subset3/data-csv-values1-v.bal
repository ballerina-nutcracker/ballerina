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

type Anydata anydata[][];

type Floats float[][];

type Decimals decimal[][];

type Booleans boolean[][];

type Ints int[][];

type IntOrString (int|string)[][];

type StringOrNil (string?)[][];

type Finite ("A"|"B"|1|2.5|true|())[][];

type Bytes byte[][];

type Signed8 int:Signed8[][];

type Unsigned32 int:Unsigned32[][];

type Chars string:Char[][];

type Nils ()[][];

type Nested int[][][];

public function main() {
    io:println(csv:parseString("a\n1\n+1\n-1\n-0\n01\n1.0\n1.\n.5\n1E5\n1e5\n1E-5\n1,000\n9223372036854775807\n9223372036854775808\nnull\nTrue\nx", {}, Anydata));
    io:println(csv:parseString("a\n1\n-0\n1.0\n1.\n1E5\n1E+5\n1.5E+3\n1E-5\n9223372036854775808", {}, Floats));
    io:println(csv:parseString("a\n1E400", {}, Floats));
    io:println(csv:parseString("a\n1\n100\n0.001\n0.0001\n12345678\n12345678901234567890.123456789\n1E-5\n-0", {}, Decimals));
    io:println(csv:parseString("a\n1E400", {}, Decimals));
    io:println(csv:parseString("a\n1E5x", {}, Floats));
    io:println(csv:parseString("a\n1E+-5", {}, Floats));
    io:println(csv:parseString("a\ntrue\nTRUE\nFalse", {}, Booleans));
    io:println(csv:parseString("a\nyes", {}, Booleans));
    io:println(csv:parseString("a\n-0", {}, Ints));
    io:println(csv:parseString("a\n1\nx\n1.5", {}, IntOrString));
    io:println(csv:parseString("a\nnull\nx", {}, StringOrNil));
    io:println(csv:parseString("a,b,c,d,e,f\nA,1,2.5,true,null,B", {}, Finite));
    io:println(csv:parseString("a\nTRUE", {}, Finite));
    io:println(csv:parseString("a\n255\n256", {}, Bytes));
    io:println(csv:parseString("a\n4294967297", {}, Bytes));
    io:println(csv:parseString("a\n127\n-129", {}, Signed8));
    io:println(csv:parseString("a\n4294967295\n-1", {}, Unsigned32));
    io:println(csv:parseString("a\nx\nxy", {}, Chars));
    io:println(csv:parseString("a\n()\nx", {}, Nils));
    io:println(csv:parseString("a\n1", {}, Nested));
    // nilValue replaces the default nil spellings.
    io:println(csv:parseString("a\nN/A\nnull", {nilValue: "N/A"}, Anydata));
    // The locale decides the decimal separator and the minus sign.
    io:println(csv:parseString("a\n1,5", {locale: "de_DE"}, Floats));
    io:println(csv:parseString("a\n1.000", {locale: "de_DE"}, Anydata));
    io:println(csv:parseString("a\n-2", {locale: "sv_SE"}, Anydata));
    io:println(csv:parseString("a\n\u{2212}2", {locale: "sv_SE"}, Anydata));
}

// @output [[1],["+1"],[-1],[-0.0],[1],[1.0],[1.0],[".5"],[100000.0],["1e5"],[1e-5],[1,0],[9223372036854775807],[9.223372036854776e18],[null],[true],["x"]]
// @output [[1.0],[-0.0],[1.0],[1.0],[100000.0],[100000.0],[1500.0],[1e-5],[9.223372036854776e18]]
// @output [[Infinity]]
// @output [[1.0],[100.0],[0.001],[0.00010],[12345678],[1.2345678901234567E+19],[0.000010],[0]]
// @output error Error ("value '1E400' cannot be cast into 'decimal'")
// @output error Error ("value '1E5x' cannot be cast into 'float'")
// @output error Error ("value '1E+-5' cannot be cast into 'float'")
// @output [[true],[true],[false]]
// @output error Error ("value 'yes' cannot be cast into 'boolean'")
// @output error Error ("value '-0' cannot be cast into 'int'")
// @output [[1],["x"],["1.5"]]
// @output [[null],["x"]]
// @output [["A",1,2.5,true,null,"B"]]
// @output error Error ("value 'TRUE' cannot be cast into '(true|1|2.5|\"A\"|\"B\")?'")
// @output error Error ("value '256' cannot be cast into 'byte'")
// @output error Error ("value '4294967297' cannot be cast into 'byte'")
// @output error Error ("value '-129' cannot be cast into 'lang.int:Signed8'")
// @output error Error ("value '-1' cannot be cast into 'lang.int:Unsigned32'")
// @output error Error ("value 'xy' cannot be cast into 'lang.string:Char'")
// @output error Error ("value 'x' cannot be cast into '()'")
// @output error Error ("value '1' cannot be cast into 'int[]'")
// @output [[null],["null"]]
// @output [[1.0,5.0]]
// @output [["1.000"]]
// @output [["-2"]]
// @output [[-2]]
