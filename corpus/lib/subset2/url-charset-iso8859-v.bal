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
    // ASCII-only input: same result under any ISO-8859-1 superset charset.
    io:println(check url:encode("hello", "ISO-8859-1")); // @output hello
    io:println(check url:encode("hello world", "ISO-8859-1")); // @output hello%20world

    // Latin-1 characters above U+007F are encoded as their single ISO-8859-1 byte.
    io:println(check url:encode("café", "ISO-8859-1")); // @output caf%E9
    io:println(check url:encode("résumé", "ISO-8859-1")); // @output r%E9sum%E9

    // Decode the percent-encoded ISO-8859-1 bytes back to the original string.
    io:println(check url:decode("caf%E9", "ISO-8859-1")); // @output café
    io:println(check url:decode("r%E9sum%E9", "ISO-8859-1")); // @output résumé
    io:println(check url:decode("hello%20world", "ISO-8859-1")); // @output hello world

    // Roundtrip: encode then decode returns the original string.
    string original = "naïve";
    string encoded = check url:encode(original, "ISO-8859-1");
    string decoded = check url:decode(encoded, "ISO-8859-1");
    io:println(decoded == original); // @output true
}
