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

import ballerina/crypto;
import ballerina/io;

public function main() returns error? {
    byte[] input = [104, 101, 108, 108, 111]; // "hello"
    byte[] salt = [115, 97, 108, 116];        // "salt"
    byte[] info = [105, 110, 102, 111];       // "info"

    // Default salt/info.
    io:println((check crypto:hkdfSha256(input, 32)).length()); // @output 32
    // With salt and info.
    byte[] d1 = check crypto:hkdfSha256(input, 64, salt, info);
    io:println(d1.length()); // @output 64

    // Deterministic for identical inputs.
    byte[] d2 = check crypto:hkdfSha256(input, 64, salt, info);
    io:println(d1 == d2); // @output true

    // equalConstantTime over byte[] HashValues.
    io:println(crypto:equalConstantTime(d1, d2)); // @output true
    byte[] other = check crypto:hkdfSha256(input, 64, info, salt);
    io:println(crypto:equalConstantTime(d1, other)); // @output false

    // equalConstantTime over string HashValues (the string branch of hashValueToBytes).
    io:println(crypto:equalConstantTime("abc123", "abc123")); // @output true
    io:println(crypto:equalConstantTime("abc123", "def456")); // @output false

    // A non-positive output length is rejected.
    byte[]|crypto:Error badLen = crypto:hkdfSha256(input, 0, salt, info);
    io:println(badLen is crypto:Error); // @output true
}
