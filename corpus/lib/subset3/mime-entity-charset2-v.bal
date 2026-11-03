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
import ballerina/file;
import ballerina/io;
import ballerina/mime;

public function main() returns error? {
    string path = check file:createTemp();
    check io:fileWriteBytes(path, [104, 233]);
    mime:Entity fileText = new;
    fileText.setFileAsEntityBody(path, "text/plain; charset=iso-8859-1");
    io:println((check fileText.getText()).toBytes()); // @output [104,195,169]

    check io:fileWriteBytes(path, [34, 233, 34]);
    mime:Entity fileJson = new;
    fileJson.setFileAsEntityBody(path, "application/json; charset=iso-8859-1");
    io:println((check fileJson.getJson()).toString().toBytes()); // @output [195,169]

    check io:fileWriteString(path, "--b\r\n\r\nnoct\r\n--b--\r\n");
    mime:Entity noCt = new;
    noCt.setFileAsEntityBody(path, "multipart/mixed; boundary=b");
    mime:Entity[] noCtParts = check noCt.getBodyParts();
    io:println(noCtParts[0].hasHeader("content-type")); // @output false
    check file:remove(path);

    byte[] multipartBytes = [...("--b\r\nContent-Type: text/plain; charset=iso-8859-1\r\n\r\nh").toBytes(), 233, ...("\r\n--b--\r\n").toBytes()];
    mime:Entity multipart = new;
    multipart.setByteArray(multipartBytes, "multipart/mixed; boundary=b");
    mime:Entity[] parts = check multipart.getBodyParts();
    io:println((check parts[0].getText()).toBytes()); // @output [104,195,169]

    mime:Entity byteArray = new;
    byteArray.setByteArray([104, 233], "text/plain; charset=iso-8859-1");
    io:println((check byteArray.getText()).toBytes()); // @output [104,239,191,189]

    mime:Entity utf8Alias = new;
    utf8Alias.setText("hé", "text/plain; charset=utf8");
    io:println(check utf8Alias.getByteArray()); // @output [104,195,169]

    io:println(check mime:base64Encode("h€", "iso-8859-1")); // @output aD8=
    io:println(check mime:base64Encode("hé", "utf8")); // @output aMOp
    io:println(check mime:base64Decode("aGVs*bG8=")); // @output hello
    io:println(check mime:base64Decode("aMOp", "utf8")); // @output hé
}
