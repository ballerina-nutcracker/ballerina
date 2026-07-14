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
    string path = "/tmp/bal_io_bytes1.bin";
    byte[] written = [72, 101, 108, 108, 111];
    check io:fileWriteBytes(path, written);
    byte[] readBack = check io:fileReadBytes(path);
    io:println(readBack.length());
    io:println(readBack[0]);
    io:println(readBack[4]);
}
// @output 5
// @output 72
// @output 111
