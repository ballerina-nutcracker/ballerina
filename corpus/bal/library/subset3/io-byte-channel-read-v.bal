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
    string path = "/tmp/bal_io_byte_channel_read.bin";
    check io:fileWriteBytes(path, [1, 2, 3, 4, 5, 6, 7]);

    io:ReadableByteChannel channel = check io:openReadableFile(path);
    io:println(channel is io:ReadableByteChannel); // @output true

    byte[]|io:Error r1 = channel.read(3);
    if r1 is byte[] {
        io:println(r1.length()); // @output 3
        io:println(r1[0]); // @output 1
        io:println(r1[2]); // @output 3
    }

    byte[]|io:Error r2 = channel.read(100);
    if r2 is byte[] {
        io:println(r2.length()); // @output 4
        io:println(r2[3]); // @output 7
    }

    byte[]|io:Error r3 = channel.read(10);
    io:println(r3 is io:Error); // @output true

    check channel.close();
}
