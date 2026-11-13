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
    // @output {"u":5,"n":"t","o":3,"g":"B","h":2,"y":7,"s":-8,"f":2.0}
    // @output {"u":5,"n":"t","o":3,"g":"B","h":2,"y":7,"s":-8,"f":2.0}
    both("<r><u>x</u><n/><g>A</g><h>1</h><y>255</y><s>127</s><f>1.5</f></r>", Scalars);
    // @output {"u":"x","g":"A","h":1,"y":255,"s":127,"f":1.5,"n":null}
    // @output {"u":"x","g":"A","h":1,"y":255,"s":127,"f":1.5,"n":null}
    both("<r><u>x</u><n/><g>C</g><h>1</h><y>255</y><s>127</s><f>1.5</f></r>", Scalars);
    // @output E: 'string' value 'C' cannot be converted to '("A"|"B")'
    // @output E: 'string' value 'C' cannot be converted to '("A"|"B")'
    both("<r><u>x</u><n/><g>A</g><h>1</h><y>256</y><s>127</s><f>1.5</f></r>", Scalars);
    // @output E: 'string' value '256' cannot be converted to 'byte'
    // @output E: 'string' value '256' cannot be converted to 'byte'
    both("<r><u>x</u><n/><g>A</g><h>1</h><y>2</y><s>1</s><f>1.5f</f></r>", Scalars);
    // @output E: 'string' value '1.5f' cannot be converted to 'float'
    // @output E: 'string' value '1.5f' cannot be converted to 'float'
    both("<r><b>false</b><f>2.5</f><i>1</i><n>4</n><y>3</y></r>", Primitives);
    // @output {"b":false,"f":2.5,"i":1,"n":4,"y":3}
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
    // @output {"t":"xyz"}
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

    // Elements nest at most 1000 levels deep, as in jBallerina's parseString with a record target.
    record {}|error atLimit = xmldata:parseString(nested(1000));
    io:println(atLimit is record {});
    // @output true
    record {}|error tooDeep = xmldata:parseString(nested(1001));
    show(tooDeep);
    // @output E: failed to parse xml: Maximum Element Depth limit (1000) Exceeded
    record {}|error tooDeepBytes = xmldata:parseBytes(nested(1001).toBytes());
    show(tooDeepBytes);
    // @output E: failed to parse xml: Maximum Element Depth limit (1000) Exceeded
    // jBallerina panics with a TypeCastError for a union target and applies no limit in parseAsType.
    record {|record {}|string a?;|}|error tooDeepUnion = xmldata:parseString(nested(1001));
    show(tooDeepUnion);
    // @output E: failed to parse xml: Maximum Element Depth limit (1000) Exceeded
    // The limit applies before building the XML value for a union target, which takes time quadratic in the depth.
    string millionDeep = repeat("<a>", 1000000) + repeat("</a>", 1000000);
    record {|record {}|string a?;|}|error millionDeepUnion = xmldata:parseString(millionDeep);
    show(millionDeepUnion);
    // @output E: failed to parse xml: Maximum Element Depth limit (1000) Exceeded
    // Building the value accepts text and further elements beside the root, so they do not skip the limit.
    record {|record {}|string a?;|}|error leadingTextUnion = xmldata:parseString("x" + millionDeep);
    show(leadingTextUnion);
    // @output E: failed to parse xml: Maximum Element Depth limit (1000) Exceeded
    record {|record {}|string a?;|}|error secondRootUnion = xmldata:parseString("<b/>" + millionDeep);
    show(secondRootUnion);
    // @output E: failed to parse xml: Maximum Element Depth limit (1000) Exceeded
    // A build error before the element that exceeds the limit is reported instead.
    record {|record {}|string a?;|}|error prefixFirstUnion = xmldata:parseString("<p:a>" + millionDeep + "</p:a>");
    show(prefixFirstUnion);
    // @output E: failed to parse xml: 'undeclared XML namespace prefix "p"'
    record {}|error valueAtLimit = xmldata:parseAsType(nestedValue(1000));
    io:println(valueAtLimit is record {});
    // @output true
    record {}|error tooDeepValue = xmldata:parseAsType(nestedValue(1001));
    show(tooDeepValue);
    // @output E: failed to parse xml: Maximum Element Depth limit (1000) Exceeded

    // An attribute may appear only once per expanded name, whatever prefix names it.
    record {}|error duplicate = xmldata:parseString("<r id='one' id='two'/>");
    show(duplicate);
    // @output E: failed to parse xml: XML syntax error on line 1: Duplicate attribute 'id'.
    record {}|error duplicateNs = xmldata:parseString("<r xmlns:p='urn:n' xmlns:q='urn:n' p:id='one' q:id='two'/>");
    show(duplicateNs);
    // @output E: failed to parse xml: XML syntax error on line 1: Duplicate attribute '{urn:n}id'.
    record {}|error distinctNs = xmldata:parseString("<r xmlns:p='urn:p' xmlns:q='urn:q' p:id='one' q:id='two'/>");
    io:println(distinctNs is record {});
    // @output true

    // Namespace declarations follow the Namespaces in XML rules: one per prefix, no empty URI for a prefix, and the xml and xmlns bindings are fixed.
    string[] namespaces = [
        "<r xmlns:p='a' xmlns:p='b' id='x'/>",
        "<r xmlns='a' xmlns='b' id='x'/>",
        "<r xmlns:p='' p:id='1'/>",
        "<r xmlns:xml='urn:other' id='1'/>",
        "<r xmlns:xmlns='urn:x' id='1'/>",
        "<r xmlns:p='http://www.w3.org/XML/1998/namespace' id='1'/>",
        "<r xmlns='http://www.w3.org/2000/xmlns/' id='1'/>",
        "<r xmlns:xml='http://www.w3.org/XML/1998/namespace' xmlns='' xml:lang='en' id='1'/>",
        "<r xmlns:p='a'><c xmlns:p='b' id='1'/></r>"
    ];
    foreach string namespace in namespaces {
        record {|string id?;|}|error parsed = xmldata:parseString(namespace);
        show(parsed);
    }
    // @output E: failed to parse xml: XML syntax error on line 1: Duplicate declaration for namespace prefix 'p'.
    // @output E: failed to parse xml: XML syntax error on line 1: Duplicate default namespace declaration.
    // @output E: failed to parse xml: XML syntax error on line 1: prefixed namespace declaration cannot have an empty URI
    // @output E: failed to parse xml: XML syntax error on line 1: invalid xml namespace binding
    // @output E: failed to parse xml: XML syntax error on line 1: reserved xmlns namespace binding
    // @output E: failed to parse xml: XML syntax error on line 1: invalid xml namespace binding
    // @output E: failed to parse xml: XML syntax error on line 1: reserved xmlns namespace binding
    // @output {"id":"1"}
    // @output {}

    // The XML declaration is allowed only at the very start; xml in any case is otherwise a reserved target.
    string[] declarations = [
        "<?xml version='1.0'?><r id='ok'/>",
        "<r id='ok'/><?xml version='1.0'?>",
        "<r id='ok'/><?XML version='1.0'?>",
        "<r id='ok'><?xml version='1.0'?></r>",
        "<!-- c --><?xml version='1.0'?><r id='ok'/>",
        " <?xml version='1.0'?><r id='ok'/>",
        "<?Xml version='1.0'?><r id='ok'/>",
        "<r id='ok'/><?xml-stylesheet href='a'?>"
    ];
    foreach string declaration in declarations {
        record {|string id?;|}|error parsed = xmldata:parseString(declaration);
        show(parsed);
    }
    // @output {"id":"ok"}
    // @output E: failed to parse xml: XML syntax error on line 1: Illegal processing instruction target ("xml"); xml (case insensitive) is reserved by the specs.
    // @output E: failed to parse xml: XML syntax error on line 1: Illegal processing instruction target ("XML"); xml (case insensitive) is reserved by the specs.
    // @output E: failed to parse xml: XML syntax error on line 1: Illegal processing instruction target ("xml"); xml (case insensitive) is reserved by the specs.
    // @output E: failed to parse xml: XML syntax error on line 1: Illegal processing instruction target ("xml"); xml (case insensitive) is reserved by the specs.
    // @output E: failed to parse xml: XML syntax error on line 1: Illegal processing instruction target ("xml"); xml (case insensitive) is reserved by the specs.
    // @output E: failed to parse xml: XML syntax error on line 1: Illegal processing instruction target ("Xml"); xml (case insensitive) is reserved by the specs.
    // @output {"id":"ok"}
}

function nestedValue(int depth) returns xml {
    xml element = xml:createElement("a");
    foreach int _ in 1 ..< depth {
        element = xml:createElement("a", {}, element);
    }
    return element;
}

function nested(int depth) returns string {
    string open = "";
    string close = "";
    foreach int _ in 0 ..< depth {
        open += "<a>";
        close += "</a>";
    }
    return open + close;
}

function repeat(string s, int n) returns string {
    if n == 0 {
        return "";
    }
    string half = repeat(s, n / 2);
    return n % 2 == 0 ? half + half : half + half + s;
}
