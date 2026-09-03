// Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
//
// WSO2 LLC. licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except
// in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

import ballerina/io;

type XMLAlias xml;

isolated function checkStepTypes(xml value) {
    xml allChildren = value/*;
    xml<xml:Element> filtered = value/*.<item>;
    xml<xml:Element> indexed = value/<item>[0];
    xml<xml:Element> elementChildren = value/*.elementChildren();
    xml<xml:Element> got = value/<item>.get(0);
    xml<xml:Element> mapped = value/<item>.map(item => item);
    xml:Text text = value/*.text();
    xml<xml:Comment> comments = value/<item>.map(item => xml:createComment(item.data()));
    _ = [allChildren, filtered, indexed, elementChildren, got, mapped, text, comments];
}

function checkSpecialOperandTypes(xml|string value, XMLAlias alias, xml<never> empty) {
    if value is xml {
        xml narrowed = value/*;
        _ = narrowed;
    }
    xml aliased = alias/*;
    xml emptyResult = empty/*;
    _ = [aliased, emptyResult];
}

function checkReadonlyOperandType(xml & readonly readonlyValue) {
    xml readonlyResult = readonlyValue/*;
    _ = readonlyResult;
}

public function main() {
    checkStepTypes(xml `<root><item>A</item><item>B</item></root>`);
    checkSpecialOperandTypes(xml `<root><item>A</item></root>`,
        xml `<root><item>B</item></root>`, xml ``);
    io:println("xml-step-types ok"); // @output xml-step-types ok
}
