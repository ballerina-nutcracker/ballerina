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

public function main() {
    xml:Element element = xml:createElement("original", {}, xml``);
    error? xmlnsNameFailure = trap element.setName("{http://www.w3.org/2000/xmlns/}bad");
    if xmlnsNameFailure is error {
        io:println(xmlnsNameFailure.message()); // @output element name cannot use the XMLNS namespace
    }
    error? childrenFailure = trap element.setChildren("\u{FFFE}");
    io:println(childrenFailure is error); // @output true
    io:println(element); // @output <original/>

    xml concatenated = xml `<a/>` + xml ``;
    io:println(concatenated); // @output <a/>

    error|xml:Element badName = trap xml:createElement("1bad", {}, xml``);
    if badName is error {
        io:println(badName.message()); // @output invalid XML element name
    }

    xml|error duplicatePrefix = xml:fromString("<a xmlns:p=\"urn:one\" xmlns:p=\"urn:two\"/>");
    if duplicatePrefix is error {
        io:println(duplicatePrefix.message()); // @output lang.xml:fromString: duplicate namespace declaration for prefix "p"
    }
    xml|error duplicateAttribute = xml:fromString("<a x=\"1\" x=\"2\"/>");
    if duplicateAttribute is error {
        io:println(duplicateAttribute.message()); // @output lang.xml:fromString: duplicate XML attribute "x"
    }
}
