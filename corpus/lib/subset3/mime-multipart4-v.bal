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

public function main() returns error? {
    string path = check file:createTemp();
    check io:fileWriteString(path, "--b\r\nContent-Type: text/plain\r\n\r\nhello\r\n--b--\r\n");
    mime:Entity fileBacked = new;
    fileBacked.setFileAsEntityBody(path, "multipart/mixed; boundary=b");
    mime:Entity[] parts = check fileBacked.getBodyParts();
    io:println(parts.length()); // @output 1
    io:println(check parts[0].getText()); // @output hello
    check file:remove(path);

    mime:Entity empty = new;
    empty.setHeader("Content-Type", "multipart/mixed; boundary=b");
    io:println(check empty.getBodyParts()); // @output []

    mime:Entity text = new;
    text.setText("--b\r\n\r\nx\r\n--b--\r\n", "multipart/mixed; boundary=b");
    io:println((check text.getBodyParts()).length()); // @output 0

    mime:Entity jsonBody = new;
    jsonBody.setJson({"a": 1}, "multipart/mixed; boundary=b");
    io:println((check jsonBody.getBodyParts()).length()); // @output 0

    mime:Entity noContentType = new;
    mime:Entity[]|mime:ParserError r = noContentType.getBodyParts();
    io:println(r is mime:ParserError ? r.message() : "parts"); // @output Entity body is not a type of composite media type. Received content-type : null
}
