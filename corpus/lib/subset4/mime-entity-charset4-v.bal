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
import ballerina/file;
import ballerina/io;
import ballerina/mime;

// jBallerina ignores the RFC 2231 `charset*` form (ballerina-platform/ballerina-library#9906).
function encode(string contentType) returns byte[]|error {
    mime:Entity e = new;
    e.setText("é", contentType);
    return e.getByteArray();
}

public function main() returns error? {
    io:println(encode("text/plain; charset*=us-ascii''iso-8859-1")); // @output [233]
    io:println(encode("text/plain; charset*=utf-8''iso%2D8859-1")); // @output [233]
    io:println(encode("text/plain; charset=utf-8; charset*=us-ascii''iso-8859-1")); // @output [233]
    io:println(encode("text/plain; charset*0*=us-ascii''iso-8859; charset*1=-1")); // @output [233]
    io:println(encode("text/plain; charset*=iso-8859-1; charset=utf-8")); // @output [195,169]
    io:println(encode("text/plain; charset*=us-ascii''iso%ZZ; charset=iso-8859-1")); // @output [233]
    io:println(encode("text/plain; charset*=us-ascii''%C3%A9; charset=iso-8859-1")); // @output [233]
    io:println(encode("text/plain; charset*0*=iso-8859; charset*1=-1; charset=utf-8")); // @output [195,169]
    io:println(encode("text/plain; charset*=us-ascii''iso-8859-1; charset=ascii")); // @output [233]
    io:println(encode("text/plain; charset=iso-8859-1; charset*0=\"é\"")); // @output [233]
    io:println(encode("text/plain; charset=iso-8859-1; charset*0*=us-ascii''ascii; charset*2=-ignored")); // @output [233]
    io:println(encode("text/plain; charset=iso-8859-1; charset*=us-ascii''ascii'")); // @output [233]
    io:println(encode("text/plain; charset=iso-8859-1; charset*=us-ascii''ascii*")); // @output [233]
    io:println(encode("text/plain; charset=iso-8859-1; charset*=\"us-ascii''ascii x\"")); // @output [233]
    io:println(encode("text/plain; charset=iso-8859-1; charset*=\"us-ascii'en us'utf-8\"")); // @output [233]
    io:println(encode("text/plain; charset=iso-8859-1; charset*=\"us ascii''utf-8\"")); // @output [233]
    io:println(encode("text/plain; charset*=us-ascii'en-GB'iso-8859-1")); // @output [233]
    io:println(encode("text/plain; charset*=us-ascii''nope")); // @output error ParserError ("Error occurred while extracting blob data from entity: nope")

    string path = check file:createTemp();
    check io:fileWriteBytes(path, [...("--b\r\nContent-Type: text/plain; charset*=us-ascii''iso-8859-1\r\n\r\n").toBytes(), 0xE9,
        ...("\r\n--b--\r\n").toBytes()]);
    mime:Entity multipart = new;
    multipart.setFileAsEntityBody(path, "multipart/mixed; boundary=b");
    mime:Entity[] parts = check multipart.getBodyParts();
    io:println(check parts[0].getText()); // @output é
    check file:remove(path);
}
