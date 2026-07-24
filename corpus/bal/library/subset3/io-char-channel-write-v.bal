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
    string path = "/tmp/bal_io_char_write.txt";
    io:WritableByteChannel wb = check io:openWritableFile(path);
    io:WritableCharacterChannel wc = new (wb, "UTF-8");
    // write returns the number of bytes written, not characters
    int n1 = check wc.write("héllo wörld", 0);
    io:println(n1); // @output 13
    int n2 = check wc.write("abc", 1);
    io:println(n2); // @output 2
    check wc.writeLine("!");
    check wc.close();
    io:Error? closedWrite = wc.close();
    if closedWrite is io:Error {
        io:println(closedWrite.message()); // @output Character channel is already closed.
    }
    int|io:Error afterClose = wc.write("x", 0);
    if afterClose is io:Error {
        io:println(afterClose.message()); // @output Character channel is already closed.
    }
    io:println(check io:fileReadString(path)); // @output héllo wörldbc!

    // append mode goes through the same byte channel option
    io:WritableByteChannel ab = check io:openWritableFile(path, io:APPEND);
    io:WritableCharacterChannel ac = new (ab, "UTF-8");
    _ = check ac.write("+more", 0);
    check ac.close();
    io:println(check io:fileReadString(path)); // @output héllo wörldbc!
    // @output +more

    // ISO-8859-1 encodes é as a single byte
    string lpath = "/tmp/bal_io_char_write_latin1.txt";
    io:WritableByteChannel lb = check io:openWritableFile(lpath);
    io:WritableCharacterChannel lc = new (lb, "ISO-8859-1");
    int n3 = check lc.write("café", 0);
    io:println(n3); // @output 4
    check lc.close();
    byte[] latinBytes = check io:fileReadBytes(lpath);
    io:println(latinBytes.length()); // @output 4
    io:ReadableByteChannel lrb = check io:openReadableFile(lpath);
    io:ReadableCharacterChannel lrc = new (lrb, "ISO-8859-1");
    io:println(check lrc.readString()); // @output café
    check lrc.close();
}
