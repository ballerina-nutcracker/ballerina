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
import ballerina/lang.'xml;

public function main() {
    xml:Element element = xml `<p id="1"><q>inner</q></p>`;
    xml:Element elementClone = element.clone();

    elementClone.setName("r");
    io:println(element.getName()); // @output p
    io:println(elementClone.getName()); // @output r

    map<string> attrs = elementClone.getAttributes();
    attrs["id"] = "2";
    io:println(element); // @output <p id="1"><q>inner</q></p>

    foreach xml:Element child in elementClone.elementChildren() {
        child.setName("s");
    }
    io:println(element); // @output <p id="1"><q>inner</q></p>
    io:println(elementClone); // @output <r id="2"><s>inner</s></r>

    elementClone.setChildren(xml `<t/>`);
    io:println(element); // @output <p id="1"><q>inner</q></p>
}
