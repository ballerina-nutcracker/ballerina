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

// More behaviour of this port that differs from jBallerina, as listed under "Notable Behavioural Changes" in
// the data.xmldata README.
type Item record {|
    string item?;
    string name;
|};

type ItemRest record {|
    Item...;
|};

type ItemField record {|
    Item item;
|};

type Text record {|
    string t;
|};

type OptionalName record {|
    string name?;
|};

type IdAttribute record {|
    @xmldata:Attribute
    int id;
|};

type OptionalIdAttribute record {|
    @xmldata:Attribute
    int id?;
|};

type TextField record {|
    xml:Text t;
|};

type RenamedJson record {|
    @xmldata:Name {value: "x-data"}
    json data;
|};

type RenamedAnydata record {|
    @xmldata:Name {value: "x-data"}
    anydata data;
|};

type RenamedJsonArray record {|
    @xmldata:Name {value: "x-data"}
    json[] data;
|};

type ArrayUnion record {|
    int[]|string[] v;
|};

type ArrayOrString record {|
    int[]|string v;
|};

type Signed record {|
    int:Signed8 s;
    byte y;
|};

type SignedAttribute record {|
    @xmldata:Attribute
    int:Signed8 s;
|};

type SignedRest record {|
    int:Signed8...;
|};

function show(anydata|error v) {
    io:println(v is error ? "E: " + v.message() : v.toString());
}

function showCut(anydata|error v) {
    if v is error {
        string message = v.message();
        int? cut = message.indexOf(" into the type '");
        io:println("E: ", cut is int ? message.substring(0, cut) : message);
    } else {
        io:println(v);
    }
}

function both(string s, typedesc<record {}> td, xmldata:SourceOptions o = {}) {
    show(xmldata:parseString(s, o, td));
    xml|error x = xml:fromString(s);
    show(x is xml ? xmldata:parseAsType(x, o, td) : x);
}

public function main() {
    // A child with the same name as its record's element does not end that record.
    both("<r><item><item>text</item><name>x</name></item></r>", ItemRest);
    // @output {"item":{"item":"text","name":"x"}}
    // @output {"item":{"item":"text","name":"x"}}
    both("<r><item><item>text</item><name>x</name></item></r>", ItemField);
    // @output {"item":{"item":"text","name":"x"}}
    // @output {"item":{"item":"text","name":"x"}}

    // Text separated by several comments, or by CDATA sections, is joined.
    both("<r><t>x<!--a--><!--b-->y</t></r>", Text);
    // @output {"t":"xy"}
    // @output {"t":"xy"}
    show(xmldata:parseString("<r><t><![CDATA[y]]>z</t></r>", {}, Text));
    // @output {"t":"yz"}
    show(xmldata:parseString("<r><t><![CDATA[a]]><![CDATA[b]]></t></r>", {}, Text));
    // @output {"t":"ab"}

    // With projection, nothing inside an unmapped element binds to the enclosing record.
    both("<r><ignored><name>bad</name></ignored></r>", OptionalName, {allowDataProjection: true});
    // @output {}
    // @output {}
    both("<r><ignored><name>bad</name></ignored><name>good</name></r>", OptionalName, {allowDataProjection: true});
    // @output {"name":"good"}
    // @output {"name":"good"}

    // An attribute value that does not convert to its declared field type is an error.
    both("<r id=\"not-an-int\"/>", IdAttribute);
    // @output E: 'string' value 'not-an-int' cannot be converted to 'int'
    // @output E: 'string' value 'not-an-int' cannot be converted to 'int'
    both("<r id=\"not-an-int\"/>", OptionalIdAttribute);
    // @output E: 'string' value 'not-an-int' cannot be converted to 'int'
    // @output E: 'string' value 'not-an-int' cannot be converted to 'int'

    // An xml:Text field does not take an element.
    show(xmldata:parseAsType(xml `<r><t>plain</t></r>`, {}, TextField));
    // @output E: 'xml' value '<t>plain</t>' cannot be converted to 'xml:Text'

    // A renamed json or anydata field is stored under its field name.
    both("<r><x-data>1</x-data></r>", RenamedJson);
    // @output {"data":1}
    // @output {"data":1}
    both("<r><x-data><k>v</k></x-data></r>", RenamedAnydata);
    // @output {"data":{"k":"v"}}
    // @output {"data":{"k":"v"}}
    show(xmldata:parseAsType(xml `<r><x-data>1</x-data><x-data>2</x-data></r>`, {}, RenamedJsonArray));
    // @output {"data":[1,2]}

    // A repeated element that fits neither array member of the union is an error, not a dropped value.
    showCut(xmldata:parseAsType(xml `<r><v>1</v><v>x</v></r>`, {}, ArrayUnion));
    // @output E: field 'v' cannot be converted
    showCut(xmldata:parseAsType(xml `<r><v>1</v><v>z</v></r>`, {}, ArrayOrString));
    // @output E: field 'v' cannot be converted
    show(xmldata:parseAsType(xml `<r><v>1</v><v>2</v></r>`, {}, ArrayUnion));
    // @output {"v":[1,2]}

    // int subtype and byte fields are converted with a range check.
    both("<r><s>-8</s><y>7</y></r>", Signed);
    // @output {"s":-8,"y":7}
    // @output {"s":-8,"y":7}
    both("<r><s>200</s><y>7</y></r>", Signed);
    // @output E: 'string' value '200' cannot be converted to 'lang.int:Signed8'
    // @output E: 'string' value '200' cannot be converted to 'lang.int:Signed8'
    both("<r s=\"-8\"/>", SignedAttribute);
    // @output {"s":-8}
    // @output {"s":-8}
    both("<r s=\"200\"/>", SignedAttribute);
    // @output E: 'string' value '200' cannot be converted to 'lang.int:Signed8'
    // @output E: 'string' value '200' cannot be converted to 'lang.int:Signed8'
    both("<r><a>-8</a></r>", SignedRest);
    // @output {"a":-8}
    // @output {"a":-8}
}
