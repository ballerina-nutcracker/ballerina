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

public function main() {
    byte[] input = [104, 101, 108, 108, 111]; // "hello"
    byte[] salt = [115, 97, 108, 116];        // "salt"

    io:println(crypto:hashMd5(input).length());       // @output 16
    io:println(crypto:hashSha1(input).length());      // @output 20
    io:println(crypto:hashSha384(input).length());    // @output 48
    io:println(crypto:hashSha512(input).length());    // @output 64
    io:println(crypto:hashKeccak256(input).length());  // @output 32

    // Salted variants exercise the salt-prepend branch.
    io:println(crypto:hashMd5(input, salt).length());     // @output 16
    io:println(crypto:hashSha256(input, salt).length());  // @output 32

    // crc32b returns a hex checksum string (printed directly to keep this file's
    // langlib imports to a single module, so the desugared import order is stable).
    io:println(crypto:crc32b(input)); // @output 3610A686
}
