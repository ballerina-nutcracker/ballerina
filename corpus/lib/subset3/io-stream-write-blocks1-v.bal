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

class BlockProvider {
    byte[][] blocks;
    int idx = 0;

    isolated function init(byte[][] blocks) {
        self.blocks = blocks;
    }

    public isolated function next() returns record {|byte[] value;|}|io:Error? {
        if self.idx >= self.blocks.length() {
            return ();
        }
        byte[] block = self.blocks[self.idx];
        self.idx += 1;
        return {value: block};
    }
}

public function main() returns error? {
    string path = "/tmp/bal_io_stream_write_blocks1.bin";
    stream<byte[], io:Error?> byteStream = new (new BlockProvider([[1, 2, 3], [4, 5], [6]]));
    check io:fileWriteBlocksFromStream(path, byteStream);

    byte[] readBack = check io:fileReadBytes(path);
    io:println(readBack.length()); // @output 6
    io:println(readBack[0]); // @output 1
    io:println(readBack[5]); // @output 6

    stream<byte[], io:Error?> appendStream = new (new BlockProvider([[7, 8]]));
    check io:fileWriteBlocksFromStream(path, appendStream, io:APPEND);
    byte[] combined = check io:fileReadBytes(path);
    io:println(combined.length()); // @output 8
    io:println(combined[7]); // @output 8
}
