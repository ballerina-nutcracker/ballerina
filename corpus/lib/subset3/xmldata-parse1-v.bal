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

type Author record {|
    string name;
    string country;
|};

type Book record {|
    string isbn;
    int id;
    string title;
    Author author;
    string[] tag;
    decimal price;
    boolean available;
    float rating;
|};

type Summary record {|
    int id;
|};

type Tags record {|
    string[2] tag;
|};

type Scalars record {|
    int|string u;
    string? n;
    int o?;
    "A"|"B" g;
    1|2 h;
    byte y;
    int:Signed8 s;
    float f;
|};

type Primitives record {|
    boolean b;
    float f;
    int i;
    int? n;
    byte y;
|};

type Text record {|
    string t;
|};

type Num record {|
    int t;
|};

type Lists record {|
    record {| string name; int age; |}[] person;
    int[] score;
|};

type Deep record {|
    record {|
        record {| string c; |} b;
    |} a;
|};

// Messages that name a record type are cut before the type: type names are printed differently.
function show(anydata|error v) {
    if v is error {
        string message = v.message();
        int? typeStart = message.indexOf(" in record '");
        io:println("E: ", typeStart is int ? message.substring(0, typeStart) : message);
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
    string bookXml = string `<book isbn="978-0">
        <id>601970</id>
        <title>Dune</title>
        <author><name>Herbert</name><country>US</country></author>
        <tag>scifi</tag>
        <tag>classic</tag>
        <price>10.50</price>
        <available>true</available>
        <rating>4.5</rating>
    </book>`;
    both(bookXml, Book);
    // @output {"isbn":"978-0","id":601970,"title":"Dune","author":{"name":"Herbert","country":"US"},"tag":["scifi","classic"],"price":10.50,"available":true,"rating":4.5}
    // @output {"isbn":"978-0","id":601970,"title":"Dune","author":{"name":"Herbert","country":"US"},"tag":["scifi","classic"],"price":10.50,"available":true,"rating":4.5}
    both(bookXml, Summary);
    // @output {"id":601970}
    // @output {"id":601970}
    both(bookXml, Summary, {allowDataProjection: false});
    // @output E: undefined field 'isbn'
    // @output E: undefined field 'isbn'
    both("<book><id>x</id></book>", Summary);
    // @output E: 'string' value 'x' cannot be converted to 'int'
    // @output E: 'string' value 'x' cannot be converted to 'int'
    both("<book><id>1</id><id>2</id></book>", Summary);
    // @output E: expected 'int' value for the field 'id' found 'array' value
    // @output E: expected 'int' value for the field 'id' found 'array' value
    both("<book/>", Summary);
    // @output E: required field 'id' not present in XML
    // @output E: required field 'id' not present in XML
    both("<book><tag>a</tag><tag>b</tag><tag>c</tag></book>", Tags);
    // @output {"tag":["a","b"]}
    // @output {"tag":["a","b"]}
    both("<book><tag>a</tag><tag>b</tag><tag>c</tag></book>", Tags, {allowDataProjection: false});
    // @output E: array size is not compatible with the expected size
    // @output E: array size is not compatible with the expected size
    both("<book><tag>a</tag></book>", Tags);
    // @output {"tag":["a",""]}
    // @output {"tag":["a",""]}
    both("<r><u>5</u><n>t</n><o>3</o><g>B</g><h>2</h><y>7</y><s>-8</s><f>2</f></r>", Scalars);
    // @output E: invalid rest type 'byte'
    // @output E: unsupported input type
    both("<r><u>x</u><n/><g>A</g><h>1</h><y>255</y><s>127</s><f>1.5</f></r>", Scalars);
    // @output E: invalid rest type 'byte'
    // @output E: unsupported input type
    both("<r><u>x</u><n/><g>C</g><h>1</h><y>255</y><s>127</s><f>1.5</f></r>", Scalars);
    // @output E: 'string' value 'C' cannot be converted to '("A"|"B")'
    // @output E: 'string' value 'C' cannot be converted to '("A"|"B")'
    both("<r><u>x</u><n/><g>A</g><h>1</h><y>256</y><s>127</s><f>1.5</f></r>", Scalars);
    // @output E: invalid rest type 'byte'
    // @output E: 'string' value '256' cannot be converted to 'byte'
    both("<r><u>x</u><n/><g>A</g><h>1</h><y>2</y><s>1</s><f>1.5f</f></r>", Scalars);
    // @output E: invalid rest type 'byte'
    // @output E: unsupported input type
    both("<r><b>false</b><f>2.5</f><i>1</i><n>4</n><y>3</y></r>", Primitives);
    // @output E: invalid rest type 'byte'
    // @output {"b":false,"f":2.5,"i":1,"n":4,"y":3}
    both("<r><b/><f/><i/><n/><y/></r>", Primitives);
    // @output {"b":false,"f":0.0,"i":0,"n":null,"y":0}
    // @output E: field 'b' cannot be converted into the type 'boolean'
    both("<r><t>  spaced  </t></r>", Text);
    // @output {"t":"  spaced  "}
    // @output {"t":"  spaced  "}
    both("<r><t>a &amp; &lt;b&gt; &#65;</t></r>", Text);
    // @output {"t":"a & <b> A"}
    // @output {"t":"a &amp; &lt;b&gt; A"}
    both("<r><t><![CDATA[ <raw> ]]></t></r>", Text);
    // @output {"t":" <raw> "}
    // @output {"t":" &lt;raw&gt; "}
    both("<r><t>x<![CDATA[y]]>z</t></r>", Text);
    // @output {"t":"z"}
    // @output {"t":"xyz"}
    both("<r><t>x<!-- c -->z</t></r>", Text);
    // @output {"t":"xz"}
    // @output {"t":"xz"}
    both("<r><t>1<!-- c -->2</t></r>", Num);
    // @output E: invalid type expected 'int' but found 'string'
    // @output E: invalid type expected 'int' but found 'string'
    both("<r><t/></r>", Text);
    // @output E: field 't' cannot be converted into the type 'string'
    // @output E: field 't' cannot be converted into the type 'string'
    Text|xmldata:Error declared = xmldata:parseString("<?xml version=\"1.0\"?><!-- lead --><?pi data?><r><t>v</t></r><!-- tail -->");
    show(declared);
    // @output {"t":"v"}
    Text|xmldata:Error doctype = xmldata:parseString("<!DOCTYPE r><r><t>v</t></r>");
    show(doctype);
    // @output E: xml root element is missing
    both("<r><person><name>A</name><age>1</age></person><score>7</score><person><name>B</name><age>2</age></person><score>8</score></r>", Lists);
    // @output {"person":[{"name":"A","age":1},{"name":"B","age":2}],"score":[7,8]}
    // @output {"person":[{"name":"A","age":1},{"name":"B","age":2}],"score":[7,8]}
    both("<r><a><b><c>deep</c></b></a></r>", Deep);
    // @output {"a":{"b":{"c":"deep"}}}
    // @output {"a":{"b":{"c":"deep"}}}
    both("<r><a><b><c>deep</c><d>x</d></b></a></r>", Deep, {allowDataProjection: false});
    // @output E: undefined field 'd'
    // @output E: undefined field 'd'
    both("<r><a><b/></a></r>", Deep);
    // @output E: required field 'c' not present in XML
    // @output E: required field 'c' not present in XML
    Text|xmldata:Error textOnly = xmldata:parseAsType(xml:createText("text only"));
    show(textOnly);
    // @output E: xml root element is missing
    Text|xmldata:Error sequence = xmldata:parseAsType(xml `<r><t>a</t></r><r><t>b</t></r>`);
    show(sequence);
    // @output E: xml root element is missing
}
