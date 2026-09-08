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
    map<xml> element = {readonly d: xml `<a/>`};
    io:println(element); // @output {"d":<a/>}
    io:println(isReadonly(element["d"])); // @output true

    map<xml> sequence = {readonly d: xml `<a/><b/>`};
    io:println(sequence); // @output {"d":<a/><b/>}
    io:println(isReadonly(sequence["d"])); // @output true

    map<xml> nested = {readonly d: xml `<a><b/></a>`};
    xml:Element parent = <xml:Element>nested["d"];
    io:println(isReadonly(parent)); // @output true
    io:println(isReadonly(parent.getChildren())); // @output true

    xml & readonly topLevel = xml `<a><b/></a>`;
    xml:Element topParent = <xml:Element>topLevel;
    io:println(isReadonly(topParent.getChildren())); // @output true
}

function isReadonly(any v) returns boolean {
    return v is readonly;
}
