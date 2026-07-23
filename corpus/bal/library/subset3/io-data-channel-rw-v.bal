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

function roundtrip(io:ByteOrder ord, string path) returns error? {
    io:WritableByteChannel wb = check io:openWritableFile(path);
    io:WritableDataChannel wd = new (wb, ord);
    check wd.writeInt16(4660);
    check wd.writeInt16(-2);
    check wd.writeInt32(305419896);
    check wd.writeInt32(-40000);
    check wd.writeInt64(81985529216486895);
    check wd.writeInt64(-5);
    check wd.writeFloat32(3.5);
    check wd.writeFloat64(2.718281828459045);
    check wd.writeBool(true);
    check wd.writeBool(false);
    check wd.writeString("héllo", "UTF-8");
    check wd.writeVarInt(0);
    check wd.writeVarInt(300);
    check wd.writeVarInt(-300);
    check wd.close();

    io:ReadableByteChannel rb = check io:openReadableFile(path);
    io:ReadableDataChannel rd = new (rb, ord);
    io:println(check rd.readInt16());
    io:println(check rd.readInt16());
    io:println(check rd.readInt32());
    io:println(check rd.readInt32());
    io:println(check rd.readInt64());
    io:println(check rd.readInt64());
    io:println(check rd.readFloat32());
    io:println(check rd.readFloat64());
    io:println(check rd.readBool());
    io:println(check rd.readBool());
    io:println(check rd.readString(6, "UTF-8"));
    io:println(check rd.readVarInt());
    io:println(check rd.readVarInt());
    io:println(check rd.readVarInt());
    check rd.close();
}

public function main() returns error? {
    check roundtrip(io:BIG_ENDIAN, "/tmp/bal_io_dc_be.bin");
    // @output 4660
    // @output -2
    // @output 305419896
    // @output -40000
    // @output 81985529216486895
    // @output -5
    // @output 3.5
    // @output 2.718281828459045
    // @output true
    // @output false
    // @output héllo
    // @output 0
    // @output 300
    // @output -300
    check roundtrip(io:LITTLE_ENDIAN, "/tmp/bal_io_dc_le.bin");
    // @output 4660
    // @output -2
    // @output 305419896
    // @output -40000
    // @output 81985529216486895
    // @output -5
    // @output 3.5
    // @output 2.718281828459045
    // @output true
    // @output false
    // @output héllo
    // @output 0
    // @output 300
    // @output -300

    // exact wire bytes, verified against jBallerina
    byte[] beBytes = check io:fileReadBytes("/tmp/bal_io_dc_be.bin");
    byte[] beExpected = [18, 52, 255, 254, 18, 52, 86, 120, 255, 255, 99, 192, 1, 35, 69, 103, 137, 171, 205, 239, 255, 255, 255, 255, 255, 255, 255, 251, 64, 96, 0, 0, 64, 5, 191, 10, 139, 20, 87, 105, 1, 0, 104, 195, 169, 108, 108, 111, 0, 130, 44, 253, 84];
    io:println(beBytes == beExpected); // @output true
    byte[] leBytes = check io:fileReadBytes("/tmp/bal_io_dc_le.bin");
    byte[] leExpected = [52, 18, 254, 255, 120, 86, 52, 18, 192, 99, 255, 255, 239, 205, 171, 137, 103, 69, 35, 1, 251, 255, 255, 255, 255, 255, 255, 255, 0, 0, 96, 64, 105, 87, 20, 139, 10, 191, 5, 64, 1, 0, 104, 195, 169, 108, 108, 111, 0, 172, 2, 212, 125];
    io:println(leBytes == leExpected); // @output true
}
