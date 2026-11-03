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

function show(anydata|error result) {
    io:println(result is error ? result.message() : result);
}

public function main() {
    mime:Entity empty = new;
    show(empty.getText()); // @output Error occurred while extracting text data from entity
    show(empty.getJson()); // @output Error occurred while extracting json data from entity
    show(empty.getXml() is mime:ParserError); // @output true
    show(empty.getXml()); // @output Error occurred while extracting xml data from entity

    mime:Entity part = new;
    part.setText("x");
    mime:Entity multipart = new;
    multipart.setBodyParts([part]);
    show(multipart.getText()); // @output Error occurred while extracting text data from entity
    show(multipart.getJson()); // @output Error occurred while extracting json data from entity
    show(multipart.getXml()); // @output Error occurred while extracting xml data from entity

    mime:Entity malformed = new;
    malformed.setText("{bad");
    show(malformed.getJson()); // @output Error occurred while extracting json data from entity
    malformed.setText("<a>");
    show(malformed.getXml()); // @output Error occurred while extracting xml data from entity
    malformed.setText("<a/><b/>");
    show(malformed.getXml()); // @output Error occurred while extracting xml data from entity

    mime:Entity emptyText = new;
    emptyText.setText("");
    show(emptyText.getText()); // @output
    show(emptyText.getXml()); // @output
}
