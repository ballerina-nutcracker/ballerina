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
    var element = {readonly document: xml `<a/>`};
    io:println(element); // @output {"document":<a/>}
    io:println(isReadonly(element.document)); // @output true

    var sequence = {readonly document: xml `<a/><!--c-->`};
    io:println(sequence); // @output {"document":<a/><!--c-->}
    io:println(isReadonly(sequence.document)); // @output true

    var interpolated = {readonly document: xml `<a>${"text"}</a>`};
    io:println(interpolated); // @output {"document":<a>text</a>}
    io:println(isReadonly(interpolated.document)); // @output true

    map<xml:Element> contextual = {readonly document: xml `<a><b/></a>`};
    io:println(contextual); // @output {"document":<a><b/></a>}
    io:println(isReadonly(contextual["document"])); // @output true

    var mutable = {document: xml `<a/>`};
    io:println(isReadonly(mutable.document)); // @output false

    xml:Element & readonly readonlyElement = xml `<a><b/></a>`;
    io:println(isReadonly(readonlyElement)); // @output true

    xml & readonly readonlySequence = xml `<a/><!--c-->`;
    io:println(isReadonly(readonlySequence)); // @output true
}

function isReadonly(any v) returns boolean {
    return v is readonly;
}
