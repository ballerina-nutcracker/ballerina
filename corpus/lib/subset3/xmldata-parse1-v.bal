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
    // byte and int subtype fields are converted with a range check (ballerina-platform/ballerina-library#9510).
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
    // jBallerina's parseString rejects the byte field y (ballerina-platform/ballerina-library#9510).
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
    // Text around a CDATA section is joined (ballerina-platform/ballerina-library#9504).
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
    // jBallerina panics with a TypeCastError for a union target and applies no limit in parseAsType
    // (ballerina-platform/ballerina-library#9643).
    record {|record {}|string a?;|}|error tooDeepUnion = xmldata:parseString(nested(1001));
    show(tooDeepUnion);
    // @output E: failed to parse xml: Maximum Element Depth limit (1000) Exceeded
    // The limit applies before building the XML value for a union target, which takes time quadratic in the depth.
    string millionDeep = repeat("<a>", 1000000) + repeat("</a>", 1000000);
    record {|record {}|string a?;|}|error millionDeepUnion = xmldata:parseString(millionDeep);
    show(millionDeepUnion);
    // @output E: failed to parse xml: Maximum Element Depth limit (1000) Exceeded
    // As for a record target, an error before the element that exceeds the limit is reported instead; jBallerina
    // panics on it for a union target (ballerina-platform/ballerina-library#9825).
    record {|record {}|string a?;|}|error leadingTextUnion = xmldata:parseString("x" + millionDeep);
    show(leadingTextUnion);
    // @output E: failed to parse xml: XML syntax error on line 1: unexpected character data "x" outside the root element
    record {|record {}|string a?;|}|error secondRootUnion = xmldata:parseString("<b/>" + millionDeep);
    show(secondRootUnion);
    // @output E: failed to parse xml: XML syntax error on line 1: multiple root elements
    record {|record {}|string a?;|}|error prefixFirstUnion = xmldata:parseString("<p:a>" + millionDeep + "</p:a>");
    show(prefixFirstUnion);
    // @output E: failed to parse xml: XML syntax error on line 1: undeclared namespace prefix "p"
    // parseAsType applies the same depth limit to an XML value (ballerina-platform/ballerina-library#9643).
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

    // The declaration's pseudo-attributes are version, then optionally encoding and standalone, in that order.
    string[] pseudoAttributes = [
        "<?xml?><r id='ok'/>",
        "<?xml encoding='UTF-8'?><r id='ok'/>",
        "<?xml version='1.0' standalone='maybe'?><r id='ok'/>",
        "<?xml version='1.0' foo='bar'?><r id='ok'/>",
        "<?xml version='1.0' standalone='yes' encoding='UTF-8'?><r id='ok'/>",
        "<?xml version='1.0' encoding=''?><r id='ok'/>",
        "<?xml version='1.0'encoding='UTF-8'?><r id='ok'/>",
        "<?xml  version = '1.0' encoding=\"UTF-8\" standalone='no' ?><r id='ok'/>",
        "<?xml version='1.1'?><r id='ok'/>",
        "<?xml version = 'abc'?><r id='ok'/>",
        "<?xml version='1.9'?><r id='ok'/>",
        "<?xml version '1.0'?><r id='ok'/>",
        "<?xml version?><r id='ok'/>",
        "<?xml version=1.0?><r id='ok'/>",
        "<?xml version=?><r id='ok'/>",
        "<?xml version='1.0' é='x'?><r id='ok'/>"
    ];
    foreach string declaration in pseudoAttributes {
        record {|string id?;|}|error parsed = xmldata:parseString(declaration);
        show(parsed);
    }
    // @output E: failed to parse xml: XML syntax error on line 1: Illegal processing instruction target ("xml"); xml (case insensitive) is reserved by the specs.
    // @output E: failed to parse xml: XML syntax error on line 1: Unexpected character 'e' (code 101); expected keyword 'version'
    // @output E: failed to parse xml: XML syntax error on line 1: Invalid XML pseudo-attribute 'standalone' value 'maybe'; expected "yes" or "no"
    // @output E: failed to parse xml: XML syntax error on line 1: Unexpected character 'f' (code 102); expected "?>" end marker
    // @output E: failed to parse xml: XML syntax error on line 1: Unexpected character 'e' (code 101); expected "?>" end marker
    // @output E: failed to parse xml: XML syntax error on line 1: Missing XML pseudo-attribute 'encoding' value
    // @output E: failed to parse xml: XML syntax error on line 1: Unexpected character 'e' (code 101); expected either '?' or white space
    // @output {"id":"ok"}
    // @output {"id":"ok"}
    // @output E: failed to parse xml: XML syntax error on line 1: Invalid XML pseudo-attribute 'version' value 'abc'; expected "1.0" or "1.1"
    // @output E: failed to parse xml: XML syntax error on line 1: Invalid XML pseudo-attribute 'version' value '1.9'; expected "1.0" or "1.1"
    // @output E: failed to parse xml: XML syntax error on line 1: Unexpected character ''' (code 39); expected '=' after 'version'
    // @output E: failed to parse xml: XML syntax error on line 1: Unexpected character '?' (code 63); expected '=' after 'version'
    // @output E: failed to parse xml: XML syntax error on line 1: Unexpected character '1' (code 49); expected a quote character enclosing value for 'version'
    // @output E: failed to parse xml: XML syntax error on line 1: Unexpected character '?' (code 63); expected a quote character enclosing value for 'version'
    // @output E: failed to parse xml: XML syntax error on line 1: Unexpected character 'é' (code 233); expected "?>" end marker

    // Markup the tokenizer accepts anywhere is checked against where XML 1.0 allows it.
    string[] placements = [
        "<r id='ok'><!DOCTYPE x></r>",
        "<r id='ok'/><!DOCTYPE r>",
        "<!ELEMENT r ANY><r id='ok'/>",
        "<r id='ok'/><!FOO>",
        "<![CDATA[ ]]><r id='ok'/>",
        "<r id='ok'/><![CDATA[x]]>",
        "\u{A0}<r id='ok'/>",
        "<r id='ok'/>\u{A0}",
        "<r id='&#xD800;'/>",
        "<r><id>&#xDFFF;</id></r>",
        "<r><id><![CDATA[&#xD800;]]></id></r>",
        "<r: id='ok'/>",
        "<:r id='ok'/>",
        "<r :a='1' id='ok'/>",
        "<r a:='1' id='ok'/>"
    ];
    foreach string placement in placements {
        record {|string id?;|}|error parsed = xmldata:parseString(placement);
        show(parsed);
    }
    // @output E: failed to parse xml: XML syntax error on line 1: Unrecognized XML directive; expected CDATA or comment ('<![CDATA[' or '<!--').
    // @output E: failed to parse xml: XML syntax error on line 1: Can not have DOCTYPE declaration in epilog
    // @output E: failed to parse xml: XML syntax error on line 1: Unexpected character 'E' (code 69) after '<!' (malformed comment?)
    // @output E: failed to parse xml: XML syntax error on line 1: Unexpected character 'F' (code 70) after '<!' (malformed comment?)
    // @output E: failed to parse xml: XML syntax error on line 1: CDATA not allowed in prolog/epilog
    // @output E: failed to parse xml: XML syntax error on line 1: CDATA not allowed in prolog/epilog
    // @output E: failed to parse xml: XML syntax error on line 1: unexpected character data "\u00a0" outside the root element
    // @output E: failed to parse xml: XML syntax error on line 1: unexpected character data "\u00a0" outside the root element
    // @output E: failed to parse xml: XML syntax error on line 1: Illegal character entity: expansion character (code 0xd800
    // @output E: failed to parse xml: XML syntax error on line 1: Illegal character entity: expansion character (code 0xdfff
    // @output {"id":"&#xD800;"}
    // @output E: failed to parse xml: XML syntax error on line 1: invalid XML name: r:
    // @output E: failed to parse xml: XML syntax error on line 1: Unexpected character ':' (code 58) (missing namespace prefix?)
    // @output E: failed to parse xml: XML syntax error on line 1: Unexpected character ':' (code 58) (missing namespace prefix?)
    // @output E: failed to parse xml: XML syntax error on line 1: invalid XML name: a:

    // A target with a union of records goes through the same checks.
    string[] unionInputs = [
        "<?xml version='1.0'?><t><r><id>x</id></r></t>",
        "<?xml version='1.1'?><t><r><id>x</id></r></t>",
        "<?xml version='2.0'?><t><r><id>x</id></r></t>",
        "\u{A0}<t><r><id>x</id></r></t>",
        "<![CDATA[ ]]><t><r><id>x</id></r></t>",
        "<t><r><id>&#xD800;</id></r></t>",
        "<t><r><n>1</n></r></t><!DOCTYPE t>",
        "<!DOCTYPE t><t><r><id>x</id></r></t>",
        "<!-- c --><!DOCTYPE t [<!ELEMENT t ANY>]><t><r><n>1</n></r></t>"
    ];
    foreach string input in unionInputs {
        record {|record {|string id;|}|record {|int n;|} r;|}|error parsed = trap xmldata:parseString(input);
        io:println(parsed is error);
    }
    // @output false
    // @output false
    // @output true
    // @output true
    // @output true
    // @output true
    // @output true
    // @output false
    // @output false

    // A DOCTYPE and processing instructions get the same checks for a union target, and the entities declared in
    // the internal subset expand.
    string body = "<t><r><id>x</id></r></t>";
    string[] doctypeInputs = [
        "<!DOCTYPE>" + body,
        "<!DOCTYPEt>" + body,
        "<!DOCTYPE 1t>" + body,
        "<!DOCTYPE a><!DOCTYPE b>" + body,
        "<!DOCTYPE t [garbage]>" + body,
        "<?p \u{1}?>" + body,
        "<t><r><id>x<?p \u{1}?></id></r></t>",
        "<t><!--\u{1}--><r><id>x</id></r></t>",
        // U+FFFE and U+FFFF are rejected (ballerina-platform/ballerina-library#9742).
        "<?p \u{FFFE}?>" + body,
        "<!--\u{FFFF}-->" + body,
        "<!DOCTYPE t [<!-- c --><?pi x?><!ATTLIST t a CDATA #IMPLIED><!ENTITY % p \"v\">]>" + body,
        "<!DOCTYPE t [<!ENTITY e \"v&#65;\"><!ENTITY e \"w\">]><t><r><id>&e;</id></r></t>",
        "<!DOCTYPE t [<!ENTITY e 'a'><!ENTITY f \"&e;b&#38;#60;\">]><t><r><id a=\"&f;\">&f;</id></r></t>",
        "<!DOCTYPE t [<!ENTITY e \"<b>v</b>\">]><t><r><id>&e;</id></r></t>",
        "<!DOCTYPE t SYSTEM \"t.dtd\" [<!ENTITY e SYSTEM \"e.xml\">]><t><r><id>&e;</id></r></t>",
        "<!DOCTYPE t [<!ENTITY e \"&f;\"><!ENTITY f \"&e;\">]><t><r><id>&e;</id></r></t>",
        "<!DOCTYPE t [<!ENTITY e \"v\">]><t><r><id>&g;</id></r></t>"
    ];
    foreach string input in doctypeInputs {
        record {|record {|string id;|}|record {|int n;|} r;|}|error parsed = trap xmldata:parseString(input);
        show(parsed);
    }
    // @output E: failed to parse xml: XML syntax error on line 1: Unexpected character '>' (code 62) (expected a name start character)
    // @output E: failed to parse xml: XML syntax error on line 1: Unrecognized XML directive '<!DOCTYPEt' (misspelled DOCTYPE?).
    // @output E: failed to parse xml: XML syntax error on line 1: Unexpected character '1' (code 49) (expected a name start character)
    // @output E: failed to parse xml: XML syntax error on line 1: Duplicate DOCTYPE declaration
    // @output E: failed to parse xml: XML syntax error on line 1: Unexpected character 'g' (code 103) (in internal DTD subset; expected a '<' to start a directive)
    // @output E: failed to parse xml: XML syntax error on line 1: Illegal character ((CTRL-CHAR, code 1))
    // @output E: failed to parse xml: XML syntax error on line 1: Illegal character ((CTRL-CHAR, code 1))
    // @output E: failed to parse xml: XML syntax error on line 1: Illegal character ((CTRL-CHAR, code 1))
    // @output E: failed to parse xml: XML syntax error on line 1: illegal character code U+FFFE
    // @output E: failed to parse xml: XML syntax error on line 1: illegal character code U+FFFF
    // @output {"r":{"id":"x"}}
    // @output {"r":{"id":"vA"}}
    // @output {"r":{"id":"ab&lt;"}}
    // @output E: failed to parse xml: XML syntax error on line 1: Entity "e" contains markup, which is not supported
    // @output E: failed to parse xml: XML syntax error on line 1: External entity "e" is not supported
    // @output E: failed to parse xml: XML syntax error on line 1: Illegal entity expansion: entity "e" expands itself recursively
    // @output E: failed to parse xml: XML syntax error on line 1: invalid character entity &g;
    // @output E: failed to parse xml: XML syntax error on line 1: Unexpected character '>' (code 62) (expected a name start character)
    // @output E: xml root element is missing
    // @output E: failed to parse xml: XML syntax error on line 1: Illegal character ((CTRL-CHAR, code 1))
    // A record target stops at the DOCTYPE, as in jBallerina, after checking the declaration up to its subset.
    foreach string input in [doctypeInputs[0], doctypeInputs[4], doctypeInputs[5], doctypeInputs[6], doctypeInputs[7], doctypeInputs[8], doctypeInputs[9]] {
        record {|record {|string id;|} r;|}|error parsed = xmldata:parseString(input);
        show(parsed);
    }
    // @output E: failed to parse xml: XML syntax error on line 1: Illegal character ((CTRL-CHAR, code 1))
    // @output E: failed to parse xml: XML syntax error on line 1: Illegal character ((CTRL-CHAR, code 1))
    // @output E: failed to parse xml: XML syntax error on line 1: illegal character code U+FFFE
    // @output E: failed to parse xml: XML syntax error on line 1: illegal character code U+FFFF

    // The internal subset's attribute defaults apply, with the first declaration of an attribute binding, and its
    // parameter entities expand to the declarations they hold.
    string[] subsetInputs = [
        "<!DOCTYPE t [<!ATTLIST r a CDATA \"p\tq&#9;r\" b NMTOKENS '  m  n ' c CDATA #IMPLIED><!ATTLIST r a CDATA 'two' c CDATA 'c'>]>" + body,
        "<!DOCTYPE t [<!ATTLIST r a CDATA #FIXED 'd'>]><t><r a='given'><id>x</id></r></t>",
        "<!DOCTYPE t [<!ENTITY e 'ev'><!ATTLIST r a (x|y) '&e;!' b NOTATION (n) 'n'>]>" + body,
        "<!DOCTYPE t [<!ATTLIST r a CDATA '&e;'><!ENTITY e 'ev'>]>" + body,
        "<!DOCTYPE t [<!ATTLIST r a CDATA 'a<b'>]>" + body,
        "<!DOCTYPE t [<!ATTLIST r a FOO 'd'>]>" + body,
        "<!DOCTYPE t [<!ENTITY % pe \"<!ENTITY e 'w'>\"><!ENTITY % pe \"<!ENTITY e 'v'>\"> %pe; %undeclared;]><t><r><id>&e;</id></r></t>",
        "<!DOCTYPE t [<!ENTITY % pe SYSTEM 'pe.dtd'> %pe;]>" + body,
        "<!DOCTYPE t [<!ENTITY % pe \"<!ENTITY e \"> %pe; 'w'>]>" + body
    ];
    foreach string input in subsetInputs {
        record {|record {|string id; string a?; string b?; string c?;|}|record {|int n;|} r;|}|error parsed = trap xmldata:parseString(input);
        show(parsed);
    }
    // @output {"r":{"a":"p q\tr","b":"m n","id":"x"}}
    // @output {"r":{"a":"given","id":"x"}}
    // @output {"r":{"a":"ev!","b":"n","id":"x"}}
    // @output E: failed to parse xml: XML syntax error on line 1: Undeclared general entity "e"
    // @output E: failed to parse xml: XML syntax error on line 1: Unexpected '<' in attribute value
    // @output E: failed to parse xml: XML syntax error on line 1: Unrecognized attribute type "FOO"
    // @output {"r":{"id":"w"}}
    // @output E: failed to parse xml: XML syntax error on line 1: External parameter entity "pe" is not supported
    // @output E: failed to parse xml: XML syntax error on line 1: Unexpected end of DOCTYPE declaration; expected ']' to end the internal subset

    // Entities expand only when referenced. As in jBallerina, a document makes at most 100000 expansions, nested
    // ones and those of parameter entities included, nested at most 500 deep. The text they add is also capped at
    // 2^26 characters, which jBallerina leaves unbounded (ballerina-platform/ballerina-library#9775).
    string[] limitInputs = [
        entityLevels(8) + body,
        entityLevels(4) + "<t><r><id>" + repeat("&a4;", 9) + "</id></r></t>",
        entityLevels(4) + "<t><r><id>" + repeat("&a4;", 10) + "</id></r></t>",
        entityLevels(4) + "<t><r a='" + repeat("&a4;", 5) + "'><id>" + repeat("&a4;", 5) + "</id></r></t>",
        parameterLevels(4) + body,
        parameterLevels(5) + body,
        entityChain(500) + "<t><r><id>&e500;</id></r></t>",
        entityChain(501) + "<t><r><id>&e501;</id></r></t>",
        parameterChain(500) + "<t><r><id>&z;</id></r></t>",
        parameterChain(501) + "<t><r><id>&z;</id></r></t>",
        "<!DOCTYPE t [<!ENTITY b \"" + repeat("b", 10000) + "\">]><t><r><id>" + repeat("&b;", 6711) + "</id></r></t>"
    ];
    foreach string input in limitInputs {
        record {|record {|string id;|}|record {|int n;|} r;|}|error parsed = trap xmldata:parseString(input);
        if parsed is error {
            show(parsed);
        } else {
            io:println(parsed.r is record {|string id;|} ? (<record {|string id;|}>parsed.r).id.length() : -1);
        }
    }
    // @output 1
    // @output 90000
    // @output E: failed to parse xml: XML syntax error on line 1: Maximum entity expansion count limit (100000) exceeded
    // @output E: failed to parse xml: XML syntax error on line 1: Maximum entity expansion count limit (100000) exceeded
    // @output 1
    // @output E: failed to parse xml: XML syntax error on line 1: Maximum entity expansion count limit (100000) exceeded
    // @output 1
    // @output E: failed to parse xml: XML syntax error on line 1: Maximum entity expansion depth limit (500) exceeded
    // @output 1
    // @output E: failed to parse xml: XML syntax error on line 1: Maximum entity expansion depth limit (500) exceeded
    // @output E: failed to parse xml: XML syntax error on line 1: Maximum entity expansion length limit (67108864 characters) exceeded
    // References after an empty element count once (55555 expansions). An attribute default counts towards the
    // text cap for each element it is applied to, and one built from entities also when declared
    // (ballerina-platform/ballerina-library#9775).
    string entityDefault = "<!DOCTYPE t [<!ENTITY b \"" + repeat("b", 10000) + "\"><!ATTLIST r a CDATA \"" + repeat("&b;", 67) + "\">]>";
    string literalDefault = "<!DOCTYPE t [<!ATTLIST r a CDATA \"" + repeat("b", 1000000) + "\">]>";
    string[] repeatedInputs = [
        entityLevels(4) + "<t>" + repeat("<x/><r a='&a4;'/>", 5) + "</t>",
        entityDefault + "<t>" + repeat("<r/>", 99) + "<r a='x'/></t>",
        entityDefault + "<t>" + repeat("<r/>", 100) + "</t>",
        literalDefault + "<t>" + repeat("<r/>", 67) + "</t>",
        literalDefault + "<t>" + repeat("<r/>", 68) + "</t>"
    ];
    foreach string input in repeatedInputs {
        record {|record {|string a?;|}[]|record {|int n;|} r;|}|error parsed = trap xmldata:parseString(input);
        if parsed is error {
            show(parsed);
        } else {
            io:println(parsed.r is record {|string a?;|}[] ? (<record {|string a?;|}[]>parsed.r).length() : -1);
        }
    }
    // @output 5
    // @output 100
    // @output E: failed to parse xml: XML syntax error on line 1: Maximum entity expansion length limit (67108864 characters) exceeded
    // @output 67
    // @output E: failed to parse xml: XML syntax error on line 1: Maximum entity expansion length limit (67108864 characters) exceeded
    record {|record {|string id;|} r;|}|error unreferenced = xmldata:parseString(entityLevels(8) + body);
    show(unreferenced);
    // @output E: xml root element is missing
}

// entityLevels declares entities a0 to a<levels>, where a<i> holds ten references to a<i-1>.
function entityLevels(int levels) returns string {
    string s = "<!DOCTYPE t [<!ENTITY a0 \"a\">";
    foreach int i in 1 ... levels {
        s += "<!ENTITY a" + i.toString() + " \"" + repeat("&a" + (i - 1).toString() + ";", 10) + "\">";
    }
    return s + "]>";
}

// parameterLevels expands parameter entity p<levels>, where p<i> holds ten references to p<i-1>.
function parameterLevels(int levels) returns string {
    string s = "<!DOCTYPE t [<!ENTITY % p0 \"<!-- c -->\">";
    foreach int i in 1 ... levels {
        s += "<!ENTITY % p" + i.toString() + " \"" + repeat("&#37;p" + (i - 1).toString() + ";", 10) + "\">";
    }
    return s + "%p" + levels.toString() + ";]>";
}

// entityChain declares entities e1 to e<n>, where e<i> references e<i-1>, so &e<n>; nests n expansions deep.
function entityChain(int n) returns string {
    string s = "<!DOCTYPE t [<!ENTITY e1 \"z\">";
    foreach int i in 2 ... n {
        s += "<!ENTITY e" + i.toString() + " \"&e" + (i - 1).toString() + ";\">";
    }
    return s + "]>";
}

// parameterChain declares parameter entities p1 to p<n>, where p<i> declares an entity and references p<i+1>,
// so %p1; nests n expansions deep.
function parameterChain(int n) returns string {
    string s = "<!DOCTYPE t [";
    foreach int i in 1 ..< n {
        s += "<!ENTITY % p" + i.toString() + " \"<!ENTITY y" + i.toString() + " 'y'> &#37;p" + (i + 1).toString() + ";\">";
    }
    return s + "<!ENTITY % p" + n.toString() + " \"<!ENTITY z 'z'>\"> %p1;]>";
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
