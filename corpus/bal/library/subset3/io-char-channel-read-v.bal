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
    string path = "/tmp/bal_io_char_read.txt";
    // content is "héllo\nwörld\r\nlast"; the CRLF is written as raw bytes so
    // no corpus golden carries a literal carriage return
    check io:fileWriteString(path, "héllo\nwörld");
    check io:fileWriteBytes(path, [13, 10], io:APPEND);
    check io:fileWriteString(path, "last", io:APPEND);

    io:ReadableByteChannel rb = check io:openReadableFile(path);
    io:ReadableCharacterChannel rc = new (rb, "UTF-8");
    string first = check rc.read(5);
    io:println(first); // @output héllo
    string rest = check rc.readString();
    io:println(rest); // @output
    // @output wörld
    // @output last
    string|io:Error atEnd = rc.read(1);
    if atEnd is io:Error {
        io:println(atEnd.message()); // @output EoF when reading from the channel
    }
    check rc.close();
    io:Error? closedAgain = rc.close();
    if closedAgain is io:Error {
        io:println(closedAgain.message()); // @output Character channel is already closed.
    }
    string|io:Error afterClose = rc.read(1);
    if afterClose is io:Error {
        io:println(afterClose.message()); // @output Character channel is already closed.
    }

    // reading exactly to the end returns an empty final read, then errors
    io:ReadableByteChannel mb = check io:createReadableChannel([97, 98]);
    io:ReadableCharacterChannel mc = new (mb, "UTF-8");
    io:println(check mc.read(2)); // @output ab
    string empty = check mc.read(1);
    io:println(empty.length()); // @output 0
    string|io:Error eof = mc.read(1);
    if eof is io:Error {
        io:println(eof.message()); // @output EoF when reading from the channel
    }
    check mc.close();

    // readAllLines and lineStream
    io:ReadableByteChannel lb = check io:openReadableFile(path);
    io:ReadableCharacterChannel lc = new (lb, "UTF-8");
    string[] lines = check lc.readAllLines();
    io:println(lines.length()); // @output 3
    io:println(lines[1]); // @output wörld
    check lc.close();

    io:ReadableByteChannel sb = check io:openReadableFile(path);
    io:ReadableCharacterChannel sc = new (sb, "UTF-8");
    stream<string, io:Error?> ls = check sc.lineStream();
    record {|string value;|}|io:Error? l1 = ls.next();
    if l1 is record {|string value;|} {
        io:println(l1.value); // @output héllo
    }
    record {|string value;|}|io:Error? l2 = ls.next();
    if l2 is record {|string value;|} {
        io:println(l2.value); // @output wörld
    }
    record {|string value;|}|io:Error? l3 = ls.next();
    if l3 is record {|string value;|} {
        io:println(l3.value); // @output last
    }
    record {|string value;|}|io:Error? l4 = ls.next();
    io:println(l4 is ()); // @output true
    check ls.close();
}
