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
    string raw = "--B\r\nContent-Type: text/plain\r\nContent-Transfer-Encoding: quoted-printable\r\n\r\n"
        + "H=C3=A9llo=3D soft=\r\nbreak\r\n"
        + "--B\r\nContent-Type: text/plain\r\nContent-Transfer-Encoding: BASE64\r\n\r\nSGVs\r\nbG8=\r\n"
        + "--B\r\nContent-Type: text/plain\r\nContent-Transfer-Encoding: 8bit\r\n\r\na=3Db\r\n--B--\r\n";
    mime:Entity whole = new;
    whole.setByteArray(raw.toBytes(), "multipart/mixed; boundary=B");
    mime:Entity[] parts = check whole.getBodyParts();

    io:println(check parts[0].getText()); // @output Héllo= softbreak
    io:println(check parts[0].getHeader("Content-Transfer-Encoding")); // @output quoted-printable
    io:println(check parts[1].getText()); // @output Hello
    io:println(check parts[1].getHeader("Content-Transfer-Encoding")); // @output BASE64
    io:println(check parts[2].getText()); // @output a=3Db

    mime:Entity quotedBoundary = new;
    quotedBoundary.setByteArray("--B C\r\n\r\nx\r\n--B C--\r\n".toBytes(), "multipart/mixed; boundary=\"B C\"");
    io:println(check (check quotedBoundary.getBodyParts())[0].getText()); // @output x
}
