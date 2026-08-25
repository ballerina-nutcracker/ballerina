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
    string path = "/tmp/bal_io_byte_channel_closed.bin";
    check io:fileWriteBytes(path, [1, 2, 3]);

    io:ReadableByteChannel reader = check io:openReadableFile(path);
    check reader.close();

    io:Error? secondClose = reader.close();
    io:println(secondClose is io:Error); // @output true
    if secondClose is io:Error {
        io:println(secondClose.message()); // @output Byte channel is already closed.
    }

    byte[]|io:Error readAfterClose = reader.read(1);
    io:println(readAfterClose is io:Error); // @output true

    io:WritableByteChannel writer = check io:openWritableFile(path);
    check writer.close();
    int|io:Error writeAfterClose = writer.write([1], 0);
    io:println(writeAfterClose is io:Error); // @output true
}
