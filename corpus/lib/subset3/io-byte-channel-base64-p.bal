// Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
//
// WSO2 LLC. licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except
// in compliance with the License.
//
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

public function main() returns error? {
    // "!@#$" is not valid Base64; jBallerina panics here too (the Java
    // IllegalArgumentException escapes uncaught).
    io:ReadableByteChannel bad = check io:createReadableChannel([33, 64, 35, 36]);
    io:ReadableByteChannel dec = check bad.base64Decode(); // @panic illegal Base64 input
    byte[] _ = check dec.readAll();
}
