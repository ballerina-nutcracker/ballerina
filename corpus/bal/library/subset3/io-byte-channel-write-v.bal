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
    string path = "/tmp/bal_io_byte_channel_write.bin";

    io:WritableByteChannel writer = check io:openWritableFile(path);
    int|io:Error n1 = writer.write([1, 2, 3, 4, 5], 2);
    if n1 is int {
        io:println(n1); // @output 3
    }
    check writer.close();

    byte[] written = check io:fileReadBytes(path);
    io:println(written.length()); // @output 3
    io:println(written[0]); // @output 3
    io:println(written[2]); // @output 5

    io:WritableByteChannel appender = check io:openWritableFile(path, io:APPEND);
    _ = check appender.write([9, 8], 0);
    check appender.close();

    byte[] appended = check io:fileReadBytes(path);
    io:println(appended.length()); // @output 5
    io:println(appended[3]); // @output 9
    io:println(appended[4]); // @output 8
}
