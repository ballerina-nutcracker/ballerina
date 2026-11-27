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
|};

type OptionalXml record {
    xml x?;
    int y?;
};

type Nested string|Nested[];

type RecursiveField record {|
    Nested n;
|};

type XmlArray record {|
    xml[] items;
|};

type ZeroInt record {|
    int[0] a;
|};

type ZeroXml record {|
    xml[0] a;
|};

type ZeroRecord record {|
    record {|int x;|}[0] a;
|};

type ZeroOrString record {|
    int[0]|string a;
|};

type ZeroRest record {|
    int b;
    int[0]...;
|};

type ZeroOuter record {|
    ZeroInt i;
    int j;
|};

type OpenZero record {
    int[0] a;
};

type FixedXml1 record {|
    xml[1] a;
|};

type FixedXml2 record {|
    xml[2] a;
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
    XmlFields fields = check xmldata:parseAsType(xml `<r><x><p>1</p></x><e><q a="1"/></e></r>`);
    io:println(fields.x);
    // @output <x><p>1</p></x>
    io:println(fields.e);
    // @output <e><q a="1"/></e>
    // A recursive field type is classified without expanding it without end.
    RecursiveField|xmldata:Error recursive = xmldata:parseString("<r><n>a</n></r>");
    show(recursive);
    // @output {"n":"a"}
    // Union members are tried in a fixed order, not declaration order, so the traversal tries the array member first, which also holds the text.
    RecursiveField|xmldata:Error recursiveTraversed = xmldata:parseAsType(xml `<r><n>a</n></r>`);
    show(recursiveTraversed);
    // @output {"n":["a"]}
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

    // An xml field leaves how parseString reads the other fields unchanged.
    show(xmldata:parseString("<r><y>1</y>tail</r>", {}, OptionalXml));
    // @output {"y":1}
    show(xmldata:parseString("<r>text</r>", {}, OptionalXml));
    // @output {}
    show(xmldata:parseString("<!DOCTYPE r [<!ENTITY e 'y'>]><r><x>&e;</x><e/></r>", {}, XmlFields));
    // @output E: xml root element is missing

    // A fixed-length xml array field keeps its elements; jBallerina fails with `invalid rest type 'xml'`
    // (ballerina-platform/ballerina-library#9910).
    FixedXml1|error f1 = xmldata:parseString("<r><a>1</a></r>");
    io:println(f1 is error ? f1.message() : f1.a);
    // @output [<a>1</a>]
    FixedXml2|error f2 = xmldata:parseString("<r><a>1</a><a>2</a></r>");
    io:println(f2 is error ? f2.message() : f2.a);
    // @output [<a>1</a>,<a>2</a>]
    FixedXml1|error f3 = xmldata:parseString("<r><a>1</a></r>", {allowDataProjection: false});
    io:println(f3 is error ? f3.message() : f3.a);
    // @output [<a>1</a>]

    // A zero-length array field projects every element away.
    show(xmldata:parseString("<r><a>1</a></r>", {}, ZeroInt));
    // @output {"a":[]}
    show(xmldata:parseString("<r><a>1</a></r>", {allowDataProjection: false}, ZeroInt));
    // @output E: array size is not compatible with the expected size
    show(xmldata:parseString("<r><a>1</a></r>", {}, ZeroXml));
    // @output {"a":[]}
    show(xmldata:parseString("<r><a><x>1</x></a><a><x>2</x></a></r>", {}, ZeroRecord));
    // @output {"a":[]}
    show(xmldata:parseString("<r><a><x>1</x></a></r>", {allowDataProjection: false}, ZeroRecord));
    // @output E: array size is not compatible with the expected size
    show(xmldata:parseAsType(xml `<r><a>1</a></r>`, {}, ZeroInt));
    // @output {"a":[]}
    show(xmldata:parseAsType(xml `<r><a>1</a></r>`, {allowDataProjection: false}, ZeroInt));
    // @output E: array size is not compatible with the expected size
    show(xmldata:parseAsType(xml `<r><a>1</a></r>`, {}, ZeroXml));
    // @output {"a":[]}
    show(xmldata:parseAsType(xml `<r><a><x>1</x></a><a><x>2</x></a></r>`, {}, ZeroRecord));
    // @output {"a":[]}
    show(xmldata:parseString("<r><a>x</a></r>", {}, ZeroOrString));
    // @output {"a":"x"}
    show(xmldata:parseAsType(xml `<r><a>x</a></r>`, {}, ZeroOrString));
    // @output {"a":"x"}
    show(xmldata:parseAsType(xml `<r><b>1</b><c>2</c></r>`, {}, ZeroRest));
    // @output {"b":1,"c":[]}
    show(xmldata:parseAsType(xml `<r><b>1</b><c>2</c></r>`, {allowDataProjection: false}, ZeroRest));
    // @output E: array size is not compatible with the expected size

    // Skipping a zero-length array element keeps the enclosing record open.
    ZeroOuter|error nested = xmldata:parseString("<r><i><a>1</a></i><j>2</j></r>");
    io:println(nested is error ? nested.message() : nested);
    // @output {"i":{"a":[]},"j":2}
    io:println(nested is ZeroOuter);
    // @output true
    show(xmldata:parseAsType(xml `<r><i><a>1</a></i><j>2</j></r>`, {}, ZeroOuter));
    // @output {"i":{"a":[]},"j":2}
    show(xmldata:parseString("<r><a>1</a><z>9</z></r>", {}, OpenZero));
    // @output {"a":[],"z":9}
    show(xmldata:parseAsType(xml `<r><a>1</a><z>9</z></r>`, {}, OpenZero));
    // @output {"a":[],"z":9}
}
