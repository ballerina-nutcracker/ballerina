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

type Row record {|string a; string b;|};

type Rows Row[];

type Lists string[][];

public function main() {
    // With only CRLF, a bare LF is data.
    Lists|csv:Error crlfOnly = csv:parseString("a,b\r\n1,x\ny\r\n2,z\r\n", {lineTerminator: csv:CRLF});
    io:println(crlfOnly is Lists && crlfOnly == [["1", "x\ny"], ["2", "z"]]);
    Rows|csv:Error crlfList = csv:parseString("a,b\r\n1,x\ny\r\n2,z", {lineTerminator: [csv:CRLF]});
    io:println(crlfList is Rows && crlfList == [{a: "1", b: "x\ny"}, {a: "2", b: "z"}]);
    // A union target parses with the same line terminators.
    Rows|Lists|csv:Error union = csv:parseString("a,b\r\n1,x\ny\r\n", {lineTerminator: csv:CRLF});
    io:println(union is Rows && union == [{a: "1", b: "x\ny"}]);
    // With only LF, a CRLF still ends a line and the CR is dropped.
    io:println(csv:parseString("a,b\r\n1,x\r\n2,y", {lineTerminator: csv:LF}, Rows));
    // The default accepts both.
    io:println(csv:parseString("a,b\r\n1,x\n2,y\r\n", {}, Rows));
    // A CR inside quotes is kept.
    Rows|csv:Error quotedCr = csv:parseString("a,b\n1,\"x\ry\"\n", {});
    io:println(quotedCr is Rows && quotedCr[0].b == "x\ry");
    Rows|csv:Error quotedCrlf = csv:parseString("a,b\r\n1,\"x\r\ny\"\r\n2,\"z\"\r\n", {});
    io:println(quotedCrlf is Rows && quotedCrlf == [{a: "1", b: "x\r\ny"}, {a: "2", b: "z"}]);
    Lists|csv:Error quotedHeader = csv:parseString("\"a\rb\",c\r\n1,2", {outputWithHeaders: true});
    io:println(quotedHeader is Lists && quotedHeader == [["a\rb", "c"], ["1", "2"]]);
}

// @output true
// @output true
// @output true
// @output [{"a":"1","b":"x"},{"a":"2","b":"y"}]
// @output [{"a":"1","b":"x"},{"a":"2","b":"y"}]
// @output true
// @output true
// @output true
