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
    mime:Entity part = new;
    mime:Entity whole = new;
    whole.setBodyParts([part]);
    io:println(whole.getBodyParts() is mime:Entity[]); // @output true

    // getBodyParts returns the list that was set, not a copy.
    mime:Entity[] parts = check whole.getBodyParts();
    mime:Entity extra = new;
    parts.push(extra);
    io:println((check whole.getBodyParts()).length()); // @output 2

    // Decoded parts are real Entity values with their headers in a stable order.
    string raw = "--B\r\nX-Z: 1\r\nContent-Disposition: form-data; name=\"a\"\r\nX-A: 2\r\nX-A: 3\r\n\r\nvalue\r\n--B--\r\n";
    mime:Entity decoded = new;
    decoded.setByteArray(raw.toBytes(), "multipart/form-data; boundary=B");
    mime:Entity[] decodedParts = check decoded.getBodyParts();
    any firstPart = decodedParts[0];
    io:println(firstPart is mime:Entity); // @output true
    io:println(decodedParts[0].getHeaderNames()); // @output ["Content-Disposition","Content-Type","X-A","X-Z"]
    io:println(check decodedParts[0].getHeaders("x-a")); // @output ["2","3"]
    io:println(check decodedParts[0].getText()); // @output value
    io:println(decodedParts[0].getContentDisposition().name); // @output a
}
