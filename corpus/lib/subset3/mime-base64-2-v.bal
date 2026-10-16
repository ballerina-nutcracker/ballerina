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

public function main() {
    // Missing `=` padding and embedded whitespace are accepted, like Java's MIME decoder.
    io:println(mime:base64Decode("SGVsbG8")); // @output Hello
    io:println(mime:base64Decode("SGV sbG8=")); // @output Hello
    io:println(mime:base64Decode("SGVsbG8\r\n")); // @output Hello
    byte[] decoded = <byte[]>checkpanic mime:base64DecodeBlob("SGk".toBytes());
    io:println(string:fromBytes(decoded)); // @output Hi
    io:println(mime:base64Decode("S") is mime:DecodeError); // @output true
}
