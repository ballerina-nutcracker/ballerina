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
    // Invalid hex digits after '%' -> decode error.
    string|error badHex = url:decode("%GG", "UTF-8");
    io:println(badHex is error); // @output true

    // A %XX escape that yields a byte that is not valid UTF-8 -> decode error.
    string|error badUtf8 = url:decode("%FF", "UTF-8");
    io:println(badUtf8 is error); // @output true

    // A trailing lone '%' (incomplete %XX) is passed through as a literal byte.
    io:println(check url:decode("a%", "UTF-8")); // @output a%

    // Lowercase hex digits in %XX escapes decode the same as uppercase.
    io:println(check url:decode("%2a%2b", "UTF-8")); // @output *+

    // Literal non-ASCII characters in the input are written through unchanged,
    // not re-interpreted through the charset decoder.
    io:println(check url:decode("café", "ISO-8859-1")); // @output café

    // Pending escaped bytes that are invalid under the charset are flushed when a
    // literal non-ASCII byte is hit mid-loop -> decode error.
    string|error flushErr = url:decode("%FFé", "US-ASCII");
    io:println(flushErr is error); // @output true
}
