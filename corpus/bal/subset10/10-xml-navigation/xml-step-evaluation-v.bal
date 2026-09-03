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

int receiverCalls = 0;
int extensionCalls = 0;
int methodArgumentCalls = 0;
int abruptCalls = 0;

function receiver() returns xml {
    receiverCalls += 1;
    return xml `<item><name>A</name></item><item><name>B</name></item>`;
}

function index() returns int {
    extensionCalls += 1;
    return 0;
}

function methodEnd() returns int {
    methodArgumentCalls += 1;
    return 1;
}

function abruptIndex() returns int {
    abruptCalls += 1;
    if abruptCalls == 2 {
        panic error("stop iteration");
    }
    return 0;
}

function assert(anydata actual, anydata expected) {
    if actual != expected {
        panic error("values are not equal");
    }
}

public function main() {
    xml selected = receiver()/*[index()];
    assert(selected, xml `<name>A</name><name>B</name>`);
    assert(receiverCalls, 1);
    assert(extensionCalls, 2);

    xml:Text empty = xml `text`;
    _ = empty/*[index()];
    assert(extensionCalls, 2);

    xml perElement = receiver()/*.get(0);
    assert(perElement, xml `<name>A</name><name>B</name>`);
    assert(receiverCalls, 2);

    xml sliced = receiver()/*.slice(0, methodEnd());
    assert(sliced, xml `<name>A</name><name>B</name>`);
    assert(receiverCalls, 3);
    assert(methodArgumentCalls, 2);

    xml<never> noElements = xml ``;
    _ = noElements/*.slice(0, methodEnd());
    assert(methodArgumentCalls, 2);

    any|error stopped = trap receiver()/*[abruptIndex()];
    assert(stopped is error, true);
    assert(abruptCalls, 2);

    xml document = xml `<item><value>A</value></item><item><value>B</value></item>`;
    assert(document/*[0], document/*.get(0));
    xml nestedIndex = document/*[((xml `<root><child/></root>`)/*).length() - 1];
    assert(nestedIndex, xml `<value>A</value><value>B</value>`);
    xml nestedArgument = document/*.slice(0, ((xml `<root><child/></root>`)/*).length());
    assert(nestedArgument, xml `<value>A</value><value>B</value>`);
    xml nestedCallback = document/*.map(item => item/*);
    assert(nestedCallback, xml `AB`);
    io:println("xml-step-evaluation ok"); // @output xml-step-evaluation ok
}
