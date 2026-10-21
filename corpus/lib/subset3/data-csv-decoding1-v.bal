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

type Rows record {|string a;|}[];

function decode(byte[] value) returns string|csv:Error {
    byte[] bytes = [97, 10];
    foreach byte b in value {
        bytes.push(b);
    }
    Rows rows = check csv:parseBytes(bytes);
    return show(rows[0].a);
}

function decodeBlocks(byte[][] blocks) returns string|csv:Error {
    stream<byte[], error?> s = blocks.toStream();
    Rows rows = check csv:parseStream(s);
    return show(rows[0].a);
}

function show(string value) returns string {
    string shown = "";
    foreach string:Char c in value {
        shown += c == "\u{FFFD}" ? "R" : c;
    }
    return shown;
}

public function main() {
    // Each invalid byte is replaced on its own.
    io:println(decode([255, 255]));
    io:println(decode([128, 128]));
    io:println(decode([192, 175]));
    io:println(decode([245, 128]));
    io:println(decode([248, 136, 128, 128, 128]));
    io:println(decode([224, 128, 128]));
    io:println(decode([240, 128, 128, 128]));
    io:println(decode([244, 144, 128, 128]));
    // A truncated sequence is replaced as a whole.
    io:println(decode([226, 130]));
    io:println(decode([226, 130, 65]));
    io:println(decode([240, 159, 152]));
    io:println(decode([240, 159, 152, 65]));
    io:println(decode([194, 65]));
    io:println(decode([65, 240, 159]));
    // An encoded surrogate is replaced as a whole.
    io:println(decode([237, 160, 128]));
    io:println(decode([237, 160, 65]));
    // Valid sequences are kept.
    io:println(decode([226, 130, 172]));
    io:println(decode([240, 159, 152, 128]));
    io:println(decode([239, 191, 189]));
    // Block boundaries do not change the decoding.
    io:println(decodeBlocks([[97, 10, 192], [192]]));
    io:println(decodeBlocks([[97, 10, 240, 159], [152, 128]]));
    io:println(decodeBlocks([[97, 10, 226], [130]]));
}

// @output RR
// @output RR
// @output RR
// @output RR
// @output RRRRR
// @output RRR
// @output RRRR
// @output RRRR
// @output R
// @output RA
// @output R
// @output RA
// @output RA
// @output AR
// @output R
// @output RA
// @output €
// @output 😀
// @output R
// @output RR
// @output 😀
// @output R
