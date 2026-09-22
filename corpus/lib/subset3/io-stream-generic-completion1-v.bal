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

// The write-from-stream functions accept a generic `error?` completion type, so
// a stream held as `stream<_, error?>` (wider than `io:Error?`) can be written
// back directly — unlike jBallerina, which rejects it.
public function main() returns error? {
    string inPath = "/tmp/bal_io_stream_generic_in.txt";
    string outPath = "/tmp/bal_io_stream_generic_out.txt";

    // Blocks round-trip through a generic-error stream.
    check io:fileWriteBytes(inPath, [72, 101, 108, 108, 111]);
    stream<byte[], error?> blocks = check io:fileReadBlocksAsStream(inPath, 2);
    check io:fileWriteBlocksFromStream(outPath, blocks);
    byte[] outBytes = check io:fileReadBytes(outPath);
    io:println(outBytes.length()); // @output 5

    // Lines round-trip through a generic-error stream.
    check io:fileWriteLines(inPath, ["Alpha", "Beta"]);
    stream<string, error?> lines = check io:fileReadLinesAsStream(inPath);
    check io:fileWriteLinesFromStream(outPath, lines);
    string[] outLines = check io:fileReadLines(outPath);
    io:println(outLines[0]); // @output Alpha
    io:println(outLines[1]); // @output Beta
}
