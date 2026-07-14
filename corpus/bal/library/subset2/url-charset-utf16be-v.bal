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

import ballerina/io;
import ballerina/url;

public function main() returns error? {
    // UTF-16BE encodes each character as two bytes (big-endian), so ASCII
    // characters gain a leading %00 byte.
    io:println(check url:encode("AB", "UTF-16BE")); // @output %00A%00B
    io:println(check url:encode("hi", "UTF-16BE")); // @output %00h%00i

    // Decode reverses the two-byte sequences back to the original string.
    io:println(check url:decode("%00A%00B", "UTF-16BE")); // @output AB
    io:println(check url:decode("%00h%00i", "UTF-16BE")); // @output hi

    // Roundtrip.
    string original = "Go";
    string encoded = check url:encode(original, "UTF-16BE");
    string decoded = check url:decode(encoded, "UTF-16BE");
    io:println(decoded == original); // @output true
}
