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

xml moduleSource = xml `<root><item>A</item></root>`;
xml moduleResult = moduleSource/<item>;
var inferredModuleResult = moduleSource/*;
public xml publicModuleResult = moduleSource/*;

class Holder {
    xml items = moduleSource/*;
}

type Result record {|
    xml items = xml `<root><item>A</item></root>`/*;
|};

function withDefault(xml value = moduleSource/*) returns xml => value;

function checkedReceiver() returns xml|error {
    return (check xmlProducer())/*;
}

function xmlProducer() returns xml|error => xml `<root><item>C</item></root>`;

function directReturn() returns xml => moduleSource/*;

function identity(xml value) returns xml => value;

function assert(anydata actual, anydata expected) {
    if actual != expected {
        panic error("values are not equal");
    }
}

public function main() returns error? {
    Result result = {};
    xml local = (true ? moduleSource : xml `<root/>`)/*;
    xml cast = (<xml>moduleSource)/*;
    assert(moduleResult, xml `<item>A</item>`);
    assert(inferredModuleResult, xml `<item>A</item>`);
    assert(publicModuleResult, xml `<item>A</item>`);
    io:println("xml-step-module-contexts ok"); // @output xml-step-module-contexts ok
    assert(result.items, xml `<item>A</item>`);
    assert(withDefault(), xml `<item>A</item>`);
    assert(check checkedReceiver(), xml `<item>C</item>`);
    assert(local, xml `<item>A</item>`);
    assert(cast, xml `<item>A</item>`);
    assert(directReturn(), xml `<item>A</item>`);
    assert(identity(moduleSource/*), xml `<item>A</item>`);
    io:println("xml-step-expression-contexts ok"); // @output xml-step-expression-contexts ok
    any|error trapped = trap moduleSource/*[-1];
    if trapped is error {
        assert(trapped.message(), "XML index out of range");
    }
    io:println("xml-step-check-contexts ok"); // @output xml-step-check-contexts ok
    io:println("xml-step-contexts ok"); // @output xml-step-contexts ok
}
