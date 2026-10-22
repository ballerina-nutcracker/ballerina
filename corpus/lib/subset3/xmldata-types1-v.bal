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

import ballerina/data.xmldata;
import ballerina/io;

type XmlFields record {|
    xml x;
    xml:Element e;
    xml:Text t;
|};

type XmlArray record {|
    xml[] items;
|};

type UnionRest record {|
    string id;
    int|boolean...;
|};

type ArrayAttribute record {|
    string[] tag;
|};

@xmldata:Namespace {prefix: "p", uri: "http://p"}
type Parent record {|
    @xmldata:Namespace {prefix: "c", uri: "http://c"}
    record {| string v; |} child;
|};

type NestedUnion record {|
    record {| int a; |}|record {| string b; |} r;
|};

function show(anydata|error v) {
    if v is error {
        io:println("E: ", v.message());
    } else {
        io:println(v);
    }
}

public function main() returns error? {
    XmlFields fields = check xmldata:parseAsType(xml `<r><x><p>1</p></x><e><q a="1"/></e><t>plain</t></r>`);
    io:println(fields.x);
    // @output <x><p>1</p></x>
    io:println(fields.e);
    // @output <e><q a="1"/></e>
    io:println(fields.t);
    // @output <t>plain</t>
    XmlArray array = check xmldata:parseAsType(xml `<r><items><a/></items><items><b/></items></r>`);
    io:println(array.items.length());
    // @output 2
    io:println(array.items[1]);
    // @output <items><b/></items>

    UnionRest|xmldata:Error u1 = xmldata:parseAsType(xml `<r><id>1</id><n>5</n><f>true</f></r>`);
    show(u1);
    // @output {"id":"1","n":5,"f":true}
    UnionRest|xmldata:Error u2 = xmldata:parseString("<r><id>1</id><n>5</n><f>true</f></r>");
    show(u2);
    // @output {"id":"1","n":5,"f":true}

    ArrayAttribute|xmldata:Error a1 = xmldata:parseAsType(xml `<r tag="x"><other/></r>`);
    show(a1);
    // @output E: attribute 'tag' cannot be converted into the array type 'string[]'

    Parent|xmldata:Error n1 = xmldata:parseAsType(
            xml `<p:r xmlns:p="http://p" xmlns:c="http://c"><c:child><v>x</v></c:child></p:r>`, {}, Parent);
    show(n1);
    // @output {"child":{"v":"x"}}
    Parent|xmldata:Error n2 = xmldata:parseAsType(
            xml `<p:r xmlns:p="http://p" xmlns:d="http://d"><d:child><v>x</v></d:child></p:r>`, {}, Parent);
    show(n2 is xmldata:Error);
    // @output true

    NestedUnion|xmldata:Error good = xmldata:parseString("<x><r><b>s</b></r></x>");
    show(good);
    // @output {"r":{"b":"s"}}
}
