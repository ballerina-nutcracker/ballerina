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

class FailingLineProvider {
    int n = 0;

    public isolated function next() returns record {|string value;|}|io:Error? {
        if self.n == 0 {
            self.n += 1;
            return {value: "first"};
        }
        return error("boom");
    }
}

class FailingBlockProvider {
    int n = 0;

    public isolated function next() returns record {|byte[] value;|}|io:Error? {
        if self.n == 0 {
            self.n += 1;
            return {value: [1, 2]};
        }
        return error("block boom");
    }
}

public function main() returns error? {
    // Reads from a non-existent file surface the error at the call, not at next().
    string missing = "/tmp/bal_io_stream_missing_xyz.dat";
    io:println(io:fileReadLinesAsStream(missing) is io:Error); // @output true
    io:println(io:fileReadBlocksAsStream(missing) is io:Error); // @output true

    // An invalid block size is rejected.
    string existing = "/tmp/bal_io_stream_blocksize_check.bin";
    check io:fileWriteBytes(existing, [1, 2, 3]);
    io:println(io:fileReadBlocksAsStream(existing, 0) is io:Error); // @output true

    // A stream that errors mid-way propagates the error from the write functions.
    string linesPath = "/tmp/bal_io_stream_write_lines_err1.txt";
    stream<string, io:Error?> lineStream = new (new FailingLineProvider());
    io:Error? lineResult = io:fileWriteLinesFromStream(linesPath, lineStream);
    io:println(lineResult is io:Error); // @output true
    if lineResult is io:Error {
        io:println(lineResult.message()); // @output boom
    }

    string blocksPath = "/tmp/bal_io_stream_write_blocks_err1.bin";
    stream<byte[], io:Error?> blockStream = new (new FailingBlockProvider());
    io:Error? blockResult = io:fileWriteBlocksFromStream(blocksPath, blockStream);
    io:println(blockResult is io:Error); // @output true
    if blockResult is io:Error {
        io:println(blockResult.message()); // @output block boom
    }
}
