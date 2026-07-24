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
    string path = "/tmp/bal_io_byte_readonly.bin";
    check io:fileWriteBytes(path, [1, 2, 3, 4, 5]);

    readonly & byte[] bytes = check io:fileReadBytes(path);
    io:println(bytes is readonly & byte[]); // @output true
    io:println(bytes.length()); // @output 5

    stream<io:Block, io:Error?> blocks = check io:fileReadBlocksAsStream(path, 2);
    record {|io:Block value;|}|io:Error? first = blocks.next();
    if first is record {|io:Block value;|} {
        io:println(first.value is readonly & byte[]); // @output true
    }
    check blocks.close();

    io:ReadableByteChannel channel = check io:openReadableFile(path);
    readonly & byte[] all = check channel.readAll();
    io:println(all is readonly & byte[]); // @output true
    io:println(all[4]); // @output 5
    check channel.close();
}
