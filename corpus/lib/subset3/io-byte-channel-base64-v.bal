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
    // encode "Hi!" and decode it back
    io:ReadableByteChannel src = check io:createReadableChannel([72, 105, 33]);
    io:ReadableByteChannel enc = check src.base64Encode();
    byte[] encBytes = check enc.readAll();
    io:println(check string:fromBytes(encBytes)); // @output SGkh

    io:ReadableByteChannel back = check io:createReadableChannel(encBytes);
    io:ReadableByteChannel dec = check back.base64Decode();
    byte[] decBytes = check dec.readAll();
    io:println(check string:fromBytes(decBytes)); // @output Hi!

    // encoding a drained channel yields an empty channel
    io:ReadableByteChannel drainedEnc = check src.base64Encode();
    byte[] emptyBytes = check drainedEnc.readAll();
    io:println(emptyBytes.length()); // @output 0

    // final padding is optional when decoding ("SGk" -> "Hi")
    io:ReadableByteChannel unpadded = check io:createReadableChannel([83, 71, 107]);
    io:ReadableByteChannel dec2 = check unpadded.base64Decode();
    byte[] unpaddedBytes = check dec2.readAll();
    io:println(check string:fromBytes(unpaddedBytes)); // @output Hi

    // file-backed channels encode the same way
    string path = "/tmp/bal_io_byte_channel_base64.bin";
    check io:fileWriteBytes(path, [72, 105, 33]);
    io:ReadableByteChannel fileCh = check io:openReadableFile(path);
    io:ReadableByteChannel fileEnc = check fileCh.base64Encode();
    byte[] fileEncBytes = check fileEnc.readAll();
    io:println(check string:fromBytes(fileEncBytes)); // @output SGkh
    check fileCh.close();

    // encoding a closed channel errors
    io:ReadableByteChannel closedCh = check io:createReadableChannel([1, 2, 3]);
    check closedCh.close();
    io:ReadableByteChannel|io:Error encClosed = closedCh.base64Encode();
    if encClosed is io:Error {
        io:println(encClosed.message()); // @output Channel is already closed.
    }
    io:ReadableByteChannel|io:Error decClosed = closedCh.base64Decode();
    if decClosed is io:Error {
        io:println(decClosed.message()); // @output Channel is already closed.
    }
}
