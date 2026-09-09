// Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
//
// WSO2 LLC. licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except
// in compliance with the License.
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
import ballerina/mime;

public function main() returns error? {
    string|byte[]|io:ReadableByteChannel|mime:EncodeError enc = mime:base64Encode("Hello");
    if enc is string {
        io:println(enc);
    }

    string|byte[]|io:ReadableByteChannel|mime:DecodeError dec = mime:base64Decode("SGVsbG8=");
    if dec is string {
        io:println(dec);
    }

    byte[]|mime:EncodeError bEnc = mime:base64EncodeBlob([1, 2, 3]);
    if bEnc is byte[] {
        byte[]|mime:DecodeError bDec = mime:base64DecodeBlob(bEnc);
        if bDec is byte[] {
            io:println(bDec.length());
        }
    }

    byte[] longInput = [];
    foreach int i in 0 ..< 100 {
        longInput.push(<byte>i);
    }
    byte[]|mime:EncodeError longEnc = mime:base64EncodeBlob(longInput);
    if longEnc is byte[] {
        string encStr = check string:fromBytes(longEnc);
        io:println(encStr.length() > 76);

        byte[]|mime:DecodeError longDec = mime:base64DecodeBlob(longEnc);
        if longDec is byte[] {
            io:println(longDec.length());
        }
    }

    string|byte[]|io:ReadableByteChannel|mime:DecodeError invalidDec = mime:base64Decode("not-valid-base64!!!");
    io:println(invalidDec is mime:DecodeError);

    // charset controls how a string input is turned into bytes before encoding: 'é' is
    // 2 bytes in UTF-8 but 1 byte in ISO-8859-1, so the two encodings must differ.
    string|byte[]|io:ReadableByteChannel|mime:EncodeError utf8Enc = mime:base64Encode("café", "utf-8");
    string|byte[]|io:ReadableByteChannel|mime:EncodeError isoEnc = mime:base64Encode("café", "iso-8859-1");
    if utf8Enc is string && isoEnc is string {
        io:println(utf8Enc != isoEnc);
    }

    // Decoding with the matching charset round-trips back to the original string.
    if isoEnc is string {
        string|byte[]|io:ReadableByteChannel|mime:DecodeError isoDec = mime:base64Decode(isoEnc, "iso-8859-1");
        if isoDec is string {
            io:println(isoDec);
        }
    }

    // Encoding/decoding a byte channel: base64Encode/Decode dispatch to the shared
    // native byte[] path and hand back a fresh channel wrapping the result.
    io:ReadableByteChannel plainChannel = check io:createReadableChannel("Hello".toBytes());
    string|byte[]|io:ReadableByteChannel|mime:EncodeError chEnc = mime:base64Encode(plainChannel);
    if chEnc is io:ReadableByteChannel {
        byte[] chEncBytes = check chEnc.readAll();
        io:println(check string:fromBytes(chEncBytes));

        io:ReadableByteChannel encodedChannel = check io:createReadableChannel(chEncBytes);
        string|byte[]|io:ReadableByteChannel|mime:DecodeError chDec = mime:base64Decode(encodedChannel);
        if chDec is io:ReadableByteChannel {
            byte[] chDecBytes = check chDec.readAll();
            io:println(check string:fromBytes(chDecBytes));
        }
    }
}
// @output SGVsbG8=
// @output Hello
// @output 3
// @output true
// @output 100
// @output true
// @output true
// @output café
// @output SGVsbG8=
// @output Hello
