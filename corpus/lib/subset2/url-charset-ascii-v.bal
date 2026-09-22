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
    // US-ASCII: pure-ASCII input passes through the ASCII codec unchanged.
    io:println(check url:encode("hello world", "US-ASCII")); // @output hello%20world
    io:println(check url:decode("hello%20world", "US-ASCII")); // @output hello world

    // "ASCII" alias resolves to the same codec.
    io:println(check url:encode("a=b", "ASCII")); // @output a%3Db

    // Encoding a non-ASCII character under US-ASCII errors (the ASCII codec
    // rejects any byte > 0x7F instead of substituting).
    string|error enc = url:encode("café", "US-ASCII");
    io:println(enc is error); // @output true

    // A %XX escape that decodes to a non-ASCII byte is invalid under US-ASCII.
    string|error dec = url:decode("%FF", "US-ASCII");
    io:println(dec is error); // @output true

    // Charset-name aliases all resolve: UTF8, LATIN-1.
    io:println(check url:encode("a b", "UTF8")); // @output a%20b
    io:println(check url:encode("café", "LATIN-1")); // @output caf%E9

    // UTF-16 (BOM-prefixed) and UTF-16LE round-trip.
    string original = "Hi";
    string utf16 = check url:encode(original, "UTF-16");
    io:println(check url:decode(utf16, "UTF-16") == original); // @output true
    string utf16le = check url:encode(original, "UTF-16LE");
    io:println(check url:decode(utf16le, "UTF-16LE") == original); // @output true
}
