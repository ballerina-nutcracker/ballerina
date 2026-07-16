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

import ballerina/mime;
import ballerina/io;

public function main() returns error? {
    // ---- setBodyParts / getBodyParts round-trip ----
    mime:Entity part1 = new;
    part1.setText("hello world");
    mime:ContentDisposition disposition = check mime:getContentDispositionObject("form-data; name=\"field1\"");
    part1.setContentDisposition(disposition);

    mime:Entity part2 = new;
    part2.setJson({x: 1, y: 2});

    mime:Entity whole = new;
    whole.setBodyParts([part1, part2], "multipart/form-data; boundary=XYZBOUNDARY");
    io:println(whole.getContentType()); // @output multipart/form-data; boundary=XYZBOUNDARY

    mime:Entity[] parts = check whole.getBodyParts();
    io:println(parts.length());          // @output 2
    io:println(check parts[0].getText()); // @output hello world
    map<json> partJson = <map<json>>check parts[1].getJson();
    io:println(partJson["x"]); // @output 1
    io:println(partJson["y"]); // @output 2

    // memoized second call returns the same parts without re-decoding
    mime:Entity[] partsAgain = check whole.getBodyParts();
    io:println(partsAgain.length()); // @output 2

    // ---- decode raw multipart bytes ----
    string raw = "--XYZBOUNDARY\r\nContent-Disposition: form-data; name=\"a\"\r\n\r\nvalue-a\r\n" +
        "--XYZBOUNDARY\r\nContent-Type: application/json\r\n\r\n{\"k\":\"v\"}\r\n--XYZBOUNDARY--\r\n";
    mime:Entity decoded = new;
    decoded.setByteArray(raw.toBytes(), "multipart/form-data; boundary=XYZBOUNDARY");
    mime:Entity[] decodedParts = check decoded.getBodyParts();
    io:println(decodedParts.length());                               // @output 2
    io:println(check decodedParts[0].getHeader("content-disposition")); // @output form-data; name="a"
    io:println(check decodedParts[0].getText());                      // @output value-a
    io:println(decodedParts[1].getContentType());                     // @output application/json
    io:println(check decodedParts[1].getJson());                      // @output {"k":"v"}

    // ---- part with no Content-Type header defaults to text/plain ----
    string rawNoContentType = "--B1\r\nContent-Disposition: form-data; name=\"noct\"\r\n\r\nplain value\r\n--B1--\r\n";
    mime:Entity noCtEntity = new;
    noCtEntity.setByteArray(rawNoContentType.toBytes(), "multipart/form-data; boundary=B1");
    mime:Entity[] noCtParts = check noCtEntity.getBodyParts();
    io:println(noCtParts[0].getContentType()); // @output text/plain
    io:println(check noCtParts[0].getText());  // @output plain value

    // ---- setBody's Entity[] arm delegates to setBodyParts ----
    mime:Entity viaSetBody = new;
    viaSetBody.setBody([part1, part2]);
    io:println(viaSetBody.getContentType()); // @output multipart/form-data
}
