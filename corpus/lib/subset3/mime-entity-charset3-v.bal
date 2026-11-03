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
    mime:Entity ascii = new;
    ascii.setText("hé€", "text/plain; charset=ascii");
    io:println(check ascii.getByteArray()); // @output [104,63,63]

    mime:Entity latin1 = new;
    latin1.setText("hé€", "text/plain; charset=iso8859-1");
    io:println(check latin1.getByteArray()); // @output [104,233,63]

    mime:Entity javaAlias = new;
    javaAlias.setText("hé€", "text/plain; charset=8859_1");
    io:println(check javaAlias.getByteArray()); // @output [104,233,63]

    mime:Entity defaultAlias = new;
    defaultAlias.setText("hé", "text/plain; charset=default");
    io:println(defaultAlias.getByteArray()); // @output error ParserError ("Error occurred while extracting blob data from entity: default")

    string path = check file:createTemp();
    check io:fileWriteString(path, "hi");
    mime:Entity fileText = new;
    fileText.setFileAsEntityBody(path, "text/plain; charset=nope");
    string|error text = fileText.getText();
    if text is error {
        io:println(text); // @output error ParserError ("Error occurred while extracting text data from entity: nope")
    }

    check io:fileWriteString(path, "--b\r\nContent-Type: text/plain; charset=nope\r\n\r\nhi\r\n--b--\r\n");
    mime:Entity multipart = new;
    multipart.setFileAsEntityBody(path, "multipart/mixed; boundary=b");
    mime:Entity[] parts = check multipart.getBodyParts();
    string|error partText = parts[0].getText();
    if partText is error {
        io:println(partText.message()); // @output Error occurred while extracting text data from entity: nope
    }
    check file:remove(path);
}
