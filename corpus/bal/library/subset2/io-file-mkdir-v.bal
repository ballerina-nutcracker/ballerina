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
    // Writing to a path whose parent directories don't exist yet creates them,
    // matching jBallerina's io module, for every writer.
    string stringPath = "/tmp/bal_io_mkdir_xyz/string/a/b/string.txt";
    check io:fileWriteString(stringPath, "hello");
    io:println(check io:fileReadString(stringPath)); // @output hello

    string linesPath = "/tmp/bal_io_mkdir_xyz/lines/a/b/lines.txt";
    check io:fileWriteLines(linesPath, ["one", "two"]);
    io:println(check io:fileReadLines(linesPath)); // @output ["one","two"]

    string bytesPath = "/tmp/bal_io_mkdir_xyz/bytes/a/b/bytes.dat";
    check io:fileWriteBytes(bytesPath, [1, 2, 3]);
    io:println(check io:fileReadBytes(bytesPath)); // @output [1,2,3]

    string jsonPath = "/tmp/bal_io_mkdir_xyz/json/a/b/data.json";
    check io:fileWriteJson(jsonPath, {"k": "v"});
    io:println(check io:fileReadJson(jsonPath)); // @output {"k":"v"}

    string xmlPath = "/tmp/bal_io_mkdir_xyz/xml/a/b/data.xml";
    check io:fileWriteXml(xmlPath, xml `<a/>`);
    io:println(check io:fileReadXml(xmlPath)); // @output <a/>

    // Appending also creates missing parent directories, even on the very
    // first write to that path.
    string appendPath = "/tmp/bal_io_mkdir_xyz/append/c/d/append.txt";
    check io:fileWriteString(appendPath, "First", io:APPEND);
    check io:fileWriteString(appendPath, "Second", io:APPEND);
    io:println(check io:fileReadString(appendPath)); // @output FirstSecond
}
