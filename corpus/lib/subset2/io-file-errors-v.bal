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
    // Reads from a non-existent file return errors from every reader.
    string missing = "/tmp/bal_io_missing_xyz.dat";
    io:println(io:fileReadString(missing) is io:Error);  // @output true
    io:println(io:fileReadLines(missing) is io:Error);   // @output true
    io:println(io:fileReadBytes(missing) is io:Error);   // @output true
    io:println(io:fileReadJson(missing) is io:Error);    // @output true
    io:println(io:fileReadXml(missing) is io:Error);     // @output true

    // Malformed JSON content -> parse error.
    string badJson = "/tmp/bal_io_bad_json.json";
    check io:fileWriteString(badJson, "{not valid json}");
    io:println(io:fileReadJson(badJson) is io:Error);    // @output true

    // Trailing content after a complete JSON value -> trailing-content error.
    string trailingJson = "/tmp/bal_io_trailing_json.json";
    check io:fileWriteString(trailingJson, "{\"k\": 1} extra");
    io:println(io:fileReadJson(trailingJson) is io:Error); // @output true
}
