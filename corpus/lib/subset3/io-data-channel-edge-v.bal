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
    // closed channels error on every operation
    io:ReadableByteChannel cb = check io:createReadableChannel([1, 2]);
    io:ReadableDataChannel cd = new (cb);
    check cd.close();
    io:Error? closedAgain = cd.close();
    if closedAgain is io:Error {
        io:println(closedAgain.message()); // @output Data channel is already closed.
    }
    int|io:Error closedRead = cd.readInt16();
    if closedRead is io:Error {
        io:println(closedRead.message()); // @output Data channel is already closed.
    }

    // readString on an empty channel returns "" first, then errors
    io:ReadableByteChannel eb = check io:createReadableChannel([]);
    io:ReadableDataChannel ed = new (eb);
    string first = check ed.readString(3, "UTF-8");
    io:println(first.length()); // @output 0
    string|io:Error second = ed.readString(3, "UTF-8");
    if second is io:Error {
        io:println(second.message()); // @output EoF when reading from the channel
    }

    // a fixed-width read on an exhausted channel panics; trap recovers it
    io:ReadableByteChannel tb = check io:createReadableChannel([]);
    io:ReadableDataChannel td = new (tb);
    int|error trapped = trap td.readInt16();
    io:println(trapped is error); // @output true

    // short reads decode the bytes actually read
    io:ReadableByteChannel sb = check io:createReadableChannel([1, 2]);
    io:ReadableDataChannel sd = new (sb);
    io:println(check sd.readInt32()); // @output 258
    io:ReadableByteChannel lb = check io:createReadableChannel([1, 2]);
    io:ReadableDataChannel ld = new (lb, io:LITTLE_ENDIAN);
    io:println(check ld.readInt32()); // @output 0

    // full-range varint roundtrip
    string path = "/tmp/bal_io_dc_varint.bin";
    io:WritableByteChannel wb = check io:openWritableFile(path);
    io:WritableDataChannel wd = new (wb);
    check wd.writeVarInt(1);
    check wd.writeVarInt(-1);
    check wd.writeVarInt(9223372036854775807);
    check wd.writeVarInt(-9223372036854775807 - 1);
    check wd.close();
    io:ReadableByteChannel rb = check io:openReadableFile(path);
    io:ReadableDataChannel rd = new (rb);
    io:println(check rd.readVarInt()); // @output 1
    io:println(check rd.readVarInt()); // @output -1
    io:println(check rd.readVarInt()); // @output 9223372036854775807
    io:println(check rd.readVarInt()); // @output -9223372036854775808
    check rd.close();
}
