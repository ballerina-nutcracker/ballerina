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

@xmldata:Name {value: "item"}
type Item record {|
    @xmldata:Attribute
    string id;
    @xmldata:Attribute
    @xmldata:Name {value: "item-kind"}
    string kind;
    @xmldata:Attribute
    string note?;
    @xmldata:Name {value: "unit-price"}
    decimal price;
    string name;
|};

@xmldata:Namespace {prefix: "bk", uri: "http://b"}
type NsBook record {|
    @xmldata:Namespace {uri: "http://a"}
    string author;
    @xmldata:Namespace {prefix: "bk", uri: "http://b"}
    string title;
|};

@xmldata:Namespace {uri: "http://b"}
type DefaultNs record {|
    string title;
|};

type SameName record {|
    string id;
|};

type SameNameAttribute record {|
    @xmldata:Attribute
    string id;
|};

type TwoNamespaces record {|
    @xmldata:Namespace {uri: "http://a"}
    @xmldata:Name {value: "v"}
    string fromA;
    @xmldata:Namespace {uri: "http://b"}
    @xmldata:Name {value: "v"}
    string fromB;
|};

type Prefixed record {|
    string attr_lang;
    string body;
|};

type Content record {|
    string lang;
    string text;
|};

// Messages that name a record type are cut before the type: type names are printed differently.
function show(anydata|error v) {
    if v is error {
        string message = v.message();
        int? typeStart = message.indexOf(" in record '");
        int? nsStart = message.indexOf(" for the type: '");
        int? intoStart = message.indexOf(" into the type '");
        int? cut = typeStart ?: nsStart ?: intoStart;
        io:println("E: ", cut is int ? message.substring(0, cut) : message);
    } else {
        io:println(v);
    }
}

function both(string s, typedesc<record {}> td, xmldata:SourceOptions o = {}) {
    record {}|error parsed = xmldata:parseString(s, o, td);
    show(parsed);
    xml|error x = xml:fromString(s);
    record {}|error traversed = x is xml ? xmldata:parseAsType(x, o, td) : x;
    show(traversed);
}

public function main() {
    both("<item id=\"1\" item-kind=\"pen\"><unit-price>2.50</unit-price><name>P</name></item>", Item);
    // @output {"id":"1","kind":"pen","price":2.50,"name":"P"}
    // @output {"id":"1","kind":"pen","price":2.50,"name":"P"}
    both("<item id=\"1\" item-kind=\"pen\" note=\"n\"><unit-price>2.50</unit-price><name>P</name></item>", Item);
    // @output {"id":"1","kind":"pen","note":"n","price":2.50,"name":"P"}
    // @output {"id":"1","kind":"pen","note":"n","price":2.50,"name":"P"}
    both("<thing id=\"1\" item-kind=\"pen\"><unit-price>2.50</unit-price><name>P</name></thing>", Item);
    // @output E: record type name 'item' mismatch with given XML name 'thing'
    // @output E: record type name 'item' mismatch with given XML name 'thing'
    both("<item item-kind=\"pen\"><unit-price>2.50</unit-price><name>P</name></item>", Item);
    // @output E: required attribute 'id' not present in XML
    // @output E: required attribute 'id' not present in XML
    both("<item id=\"1\" item-kind=\"pen\" extra=\"e\"><unit-price>2.50</unit-price><name>P</name></item>", Item, {allowDataProjection: false});
    // @output E: undefined field 'extra'
    // @output E: undefined field 'extra'
    both("<bk:book xmlns:bk=\"http://b\" xmlns:a=\"http://a\"><a:author>A</a:author><bk:title>T</bk:title></bk:book>", NsBook);
    // @output {"author":"A","title":"T"}
    // @output {"author":"A","title":"T"}
    both("<x:book xmlns:x=\"http://b\" xmlns:a=\"http://a\"><a:author>A</a:author><x:title>T</x:title></x:book>", NsBook);
    // @output E: namespace mismatched
    // @output E: namespace mismatched
    both("<x:book xmlns:x=\"http://b\" xmlns:a=\"http://a\"><a:author>A</a:author><x:title>T</x:title></x:book>", NsBook, {useSemanticEquality: false});
    // @output E: namespace mismatched
    // @output E: namespace mismatched
    both("<bk:book xmlns:bk=\"http://c\"><bk:title>T</bk:title></bk:book>", NsBook);
    // @output E: namespace mismatched
    // @output E: namespace mismatched
    both("<book xmlns=\"http://b\"><title>T</title></book>", DefaultNs);
    // @output {"title":"T"}
    // @output {"title":"T"}
    both("<book><title>T</title></book>", DefaultNs);
    // @output E: namespace mismatched
    // @output E: namespace mismatched
    both("<a id=\"x\"><id>y</id></a>", SameName);
    // @output {"id":"y"}
    // @output {"id":"y"}
    both("<a id=\"x\"/>", SameName);
    // @output {"id":"x"}
    // @output {"id":"x"}
    both("<a id=\"x\"><id>y</id></a>", SameNameAttribute);
    // @output {"id":"x"}
    // @output {"id":"x"}
    both("<r xmlns:a=\"http://a\" xmlns:b=\"http://b\"><a:v>1</a:v><b:v>2</b:v></r>", TwoNamespaces);
    // @output {"fromA":"1","fromB":"2"}
    // @output {"fromA":"1","fromB":"2"}
    both("<r lang=\"si\"><body>t</body></r>", Prefixed, {attributePrefix: "attr_"});
    // @output {"attr_lang":"si","body":"t"}
    // @output {"attr_lang":"si","body":"t"}
    both("<p lang=\"si\">hello</p>", Content, {textFieldName: "text"});
    // @output {"lang":"si","text":"hello"}
    // @output {"lang":"si","text":"hello"}
}
