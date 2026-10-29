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

type Lists string[][];

public function main() {
    // Quoted values keep delimiters; a doubled quote inside a quoted value keeps both quotes.
    io:println(csv:parseString("a,b\n1,\"x,y\"\n2,\"p\"\"q\"", {}, Rows));
    // Text after a closing quote reopens the quote.
    Rows|csv:Error reopened = csv:parseString("a,b\n1,\"xy\" ,2\n3,4", {}, Rows);
    io:println(reopened is Rows && reopened.length() == 1 && reopened[0].b == "xy\" ,2\n3,4");
    // A quote in the middle of a value starts quoting and keeps the text before it.
    io:println(csv:parseString("a,b\n1,x\"y,z\"", {}, Rows));
    // A doubled quote in a quoted header becomes a single quote.
    io:println(csv:parseString("\"a\"\"q\",b\n1,2", {outputWithHeaders: true}, Lists));
    // An unterminated quote at the end of the input keeps its content.
    io:println(csv:parseString("a,b\n1,\"abc", {}, Rows));
    // Comments end a row anywhere outside quotes; comment-only and blank lines are ignored.
    io:println(csv:parseString("a,b # header comment\n1,x # c\n# full\n\n2,y\n3,\"x#y\"", {}, Rows));
    // Escapes are recognised inside and outside quotes.
    Rows|csv:Error escaped = csv:parseString("a,b\n1,\\u0041\\t z\\n\n2,\"q\\\"\\/\\b\\f\\r\\\\\"", {}, Rows);
    if escaped is Rows {
        io:println(escaped[0].b == "A\t z", " ", escaped[1].b == "q\"/\u{8}\u{C}\r\\");
    }
    // Two escaped UTF-16 surrogates form one character.
    io:println(csv:parseString("a,b\n1,\\uD83D\\uDE00", {}, Rows));
    // Escapes in a header.
    io:println(csv:parseString("\\u0061,\"\\u0062\"\n1,2", {outputWithHeaders: true}, Lists));
    // Carriage returns outside quotes are dropped.
    io:println(csv:parseString("a,b\r\n1,ab\rc\r\n", {}, Rows));
    // Unquoted values are trimmed; quoted values are not.
    io:println(csv:parseString("a,b\n  1  ,  x  y  \n2,\"  z  \"", {}, Rows));
    // Custom delimiter, quote, escape and comment characters.
    io:println(csv:parseString("a;b\n1;'x;y'\n%c\n2;|n", {delimiter: ";", textEnclosure: "'", escapeChar: "|", comment: "%"}, Rows));
    io:println(csv:parseString("a b\n1  x", {delimiter: " "}, Lists));
    // The header is the line after `header` skipped lines.
    io:println(csv:parseString("skip me\nand me\na,b\n1,q", {header: 2}, Rows));
    // Without a header row, custom headers name the columns.
    io:println(csv:parseString("1,q\n2,r", {header: (), customHeadersIfHeadersAbsent: ["a", "b"]}, Rows));
    // A row with only blank or quoted-empty values is dropped; a delimiter-only row is kept.
    io:println(csv:parseString("a,b\n\"\"\n,\n1,x", {}, Lists));
    // U+FFFF is ordinary data.
    io:println(csv:parseString("a\nx\u{FFFF}y", {}, Lists));
}

// @output [{"a":1,"b":"x,y"},{"a":2,"b":"p\"\"q"}]
// @output true
// @output [{"a":1,"b":"xy,z"}]
// @output [["a\"q","b"],["1","2"]]
// @output [{"a":1,"b":"abc"}]
// @output [{"a":1,"b":"x"},{"a":2,"b":"y"},{"a":3,"b":"x#y"}]
// @output true true
// @output [{"a":1,"b":"😀"}]
// @output [["a","b"],["1","2"]]
// @output [{"a":1,"b":"abc"}]
// @output [{"a":1,"b":"x  y"},{"a":2,"b":"  z  "}]
// @output [{"a":1,"b":"x;y"},{"a":2,"b":""}]
// @output [["1","","x"]]
// @output [{"a":1,"b":"q"}]
// @output [{"a":1,"b":"q"},{"a":2,"b":"r"}]
// @output [["",""],["1","x"]]
// @output [["x\uffffy"]]
