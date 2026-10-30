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

import ballerina/data.jsondata;
import ballerina/io;

// Inputs that jBallerina mishandles (see "Notable Behavioural Changes" in the data.jsondata README); this
// port returns a jsondata:Error for most of them instead.
type Pair record {|
    int a;
    int b;
|};

function show(anydata|error v) {
    if v is error {
        io:println(v is jsondata:Error, " ", v.message());
    } else {
        io:println(v);
    }
}

public function main() {
    stream<byte[], error?> s1 = ["{\"a\":\"x\"}".toBytes()].toStream();
    Pair|jsondata:Error r1 = jsondata:parseStream(s1);
    show(r1);
    // @output true incompatible value 'x' for type 'int' in field 'a'
    json|jsondata:Error r2 = jsondata:parseString("[1,,2]");
    show(r2);
    // @output true incompatible expected type 'json' for value ''
    decimal|jsondata:Error r3 = jsondata:parseString("1e6145");
    show(r3);
    // @output true incompatible expected type 'decimal' for value '1e6145'
    json|jsondata:Error r4 = jsondata:parseString("{\"a\":");
    show(r4);
    // @output true invalid JSON document
    Pair|jsondata:Error r5 = jsondata:parseString("{}");
    show(r5);
    // @output true required field 'a' not present in JSON
    foreach string input in ["\"\\ud83d\"", "\"\\uDC00\"", "\"\\uD800\\u0041\"", "\"\\uD800x\"", "\"\\uDE00\\uD83D\"", "\"\\uD800\\n\""] {
        json|jsondata:Error r6 = jsondata:parseString(input);
        show(r6);
    }
    // @output true 'unpaired surrogate escape '\uD83D'' at line: '1' column: '8'
    // @output true 'unpaired surrogate escape '\uDC00'' at line: '1' column: '7'
    // @output true 'unpaired surrogate escape '\uD800'' at line: '1' column: '13'
    // @output true 'unpaired surrogate escape '\uD800'' at line: '1' column: '8'
    // @output true 'unpaired surrogate escape '\uDE00'' at line: '1' column: '7'
    // @output true 'unpaired surrogate escape '\uD800'' at line: '1' column: '9'
    string|jsondata:Error r6s = jsondata:parseString("\"\\uD800\"");
    show(r6s);
    // @output true 'unpaired surrogate escape '\uD800'' at line: '1' column: '8'
    record {|string a;|}|jsondata:Error r6r = jsondata:parseBytes("{\"a\":\"\\uDC00\"}".toBytes());
    show(r6r);
    // @output true 'unpaired surrogate escape '\uDC00'' at line: '1' column: '12'
    map<string>|jsondata:Error r6k = jsondata:parseString("{\"\\uD800\":\"x\"}");
    show(r6k);
    // @output true 'unpaired surrogate escape '\uD800'' at line: '1' column: '9'
    stream<byte[], error?> s6 = ["[\"a\\uD800\"]".toBytes()].toStream();
    string[]|jsondata:Error r6t = jsondata:parseStream(s6);
    show(r6t);
    // @output true 'unpaired surrogate escape '\uD800'' at line: '1' column: '10'
    string|jsondata:Error r6p = jsondata:parseBytes("\"\\uD83D\\uDE00\"".toBytes());
    if r6p is string {
        io:println(r6p.length(), " ", r6p.toBytes().toBase16()); // @output 1 f09f9880
    }
    byte[] invalidUtf8 = [0x22, 0x61, 0xff, 0x62, 0x22];
    stream<byte[], error?> s7 = [invalidUtf8].toStream();
    string|jsondata:Error r7 = jsondata:parseStream(s7);
    show(r7);
    // @output true invalid UTF-8 byte at offset 2
    string|jsondata:Error r8 = jsondata:parseBytes(invalidUtf8);
    show(r8);
    // @output true invalid UTF-8 byte at offset 2
    foreach string input in ["01", "-01", "00", "01.5", "[01]", "{\"a\":01}"] {
        json|jsondata:Error r9 = jsondata:parseString(input);
        show(r9);
    }
    // @output true 'leading zero in number '01'' at line: '1' column: '4'
    // @output true 'leading zero in number '-01'' at line: '1' column: '5'
    // @output true 'leading zero in number '00'' at line: '1' column: '4'
    // @output true 'leading zero in number '01.5'' at line: '1' column: '6'
    // @output true 'leading zero in number '01'' at line: '1' column: '5'
    // @output true 'leading zero in number '01'' at line: '1' column: '9'
    json|jsondata:Error r10 = jsondata:parseBytes("[1,01]".toBytes());
    show(r10);
    // @output true 'leading zero in number '01'' at line: '1' column: '8'
    stream<byte[], error?> s11 = ["{\"a\":-00.5}".toBytes()].toStream();
    json|jsondata:Error r11 = jsondata:parseStream(s11);
    show(r11);
    // @output true 'leading zero in number '-00.5'' at line: '1' column: '12'
    foreach string input in ["0", "-0", "0.5", "[0,-0.5]", "{\"a\":0e1}"] {
        json|jsondata:Error r12 = jsondata:parseString(input);
        show(r12);
    }
    // @output 0
    // @output 0
    // @output 0.5
    // @output [0,-0.5]
    // @output {"a":0.0}
}
