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

// Behaviour of this port that differs from jBallerina, as listed under "Notable Behavioural Changes" in the
// data.xmldata README.
type Person record {|
    string name;
|};

type NestedUnion record {|
    record {| int a; |}|record {| string b; |} r;
|};

type UnionField record {|
    int|string a;
|};

type XmlField record {|
    xml a;
|};

type XmlArrayField record {|
    xml[] a;
|};

type Flag record {|
    byte y;
|};

type Pair record {|
    Person[2] p;
|};

type OptionalPair record {|
    string title;
    Person[2] p?;
|};

type IntRest record {|
    int...;
|};

type IntArrayRest record {|
    int[]...;
|};

type ZeroRest record {|
    int x;
    int[0]...;
|};

type Inner record {|
    string a;
|};

type OpenOuter record {
    Inner i;
};

type NilZero record {|
    int[0]? a;
|};

type ZeroOrInt record {|
    int[0]|int a;
|};

class FailingAfterDocument {
    int count = 0;

    public isolated function next() returns record {|byte[] value;|}|error? {
        self.count += 1;
        if self.count == 1 {
            return {value: "<p><name>x</name></p>".toBytes()};
        }
        return error("stream failure");
    }
}

// jBallerina's compiler plugin rejects this record (duplicate field found): the @Name override of `a` clashes
// with the field `b`. Without compiler-plugin support the check runs when the type is used as a target.
type Clash record {|
    @xmldata:Name {value: "b"}
    string a;
    string b;
|};

public function main() {
    // Malformed XML is reported with the Go XML tokenizer's description of the problem.
    Person|xmldata:Error malformed = xmldata:parseString("<p><name>x</p>");
    string prefix = "failed to parse xml: ";
    io:println(malformed is xmldata:Error && malformed.message().substring(0, prefix.length()) == prefix);
    // @output true

    // A target type with a union of records is parsed into an XML value first; jBallerina returns that parse
    // error as a plain error and panics on it, this port returns an xmldata:Error
    // (ballerina-platform/ballerina-library#9825).
    NestedUnion|xmldata:Error unionTarget = xmldata:parseString("<r><r><a>1</a></r>");
    io:println(unionTarget is xmldata:Error && unionTarget.message().substring(0, prefix.length()) == prefix);
    // @output true

    xmldata:Error? xsd = xmldata:validate(xml `<p><name>x</name></p>`, "person.xsd");
    io:println(xsd is xmldata:Error ? xsd.message() : "valid");
    // @output Invalid XML found: 'validating XML against an XSD file is not supported'

    Clash|xmldata:Error parsed = xmldata:parseString("<r><b>x</b></r>", {}, Clash);
    io:println(parsed is xmldata:Error ? parsed.message() : "parsed");
    // @output duplicate field 'b'
    Clash|xmldata:Error traversed = xmldata:parseAsType(xml `<r><b>x</b></r>`, {}, Clash);
    io:println(traversed is xmldata:Error ? traversed.message() : "parsed");
    // @output duplicate field 'b'

    // jBallerina leaves a required field whose type cannot hold () as () when its element is empty
    // (ballerina-platform/ballerina-library#9826).
    UnionField|xmldata:Error emptyUnion = xmldata:parseString("<r><a></a></r>");
    io:println(emptyUnion is xmldata:Error ? emptyUnion.message() : "parsed");
    // @output field 'a' cannot be converted into the type '(int|string)'

    // jBallerina's parseString leaves a required xml field as () or fails, where parseAsType gives the element
    // (ballerina-platform/ballerina-library#9826, ballerina-platform/ballerina-library#9910).
    XmlField|xmldata:Error emptyXml = xmldata:parseString("<r><a/></r>");
    io:println(emptyXml is xmldata:Error ? emptyXml.message() : emptyXml.a);
    // @output <a/>
    XmlField|xmldata:Error nestedXml = xmldata:parseString("<r><a><b>1</b></a></r>");
    io:println(nestedXml is xmldata:Error ? nestedXml.message() : nestedXml.a);
    // @output <a><b>1</b></a>
    XmlField|xmldata:Error textXml = xmldata:parseString("<r><a>t</a></r>");
    io:println(textXml is xmldata:Error ? textXml.message() : textXml.a);
    // @output <a>t</a>
    XmlField|xmldata:Error nsXml = xmldata:parseString("<r xmlns:p='urn:p' xmlns='urn:d'><a><p:b>1 &amp; 2</p:b></a></r>");
    io:println(nsXml is xmldata:Error ? nsXml.message() : nsXml.a);
    // @output <a xmlns="urn:d"><p:b xmlns:p="urn:p">1 &amp; 2</p:b></a>
    XmlArrayField|xmldata:Error xmlArray = xmldata:parseString("<r><a>1</a><a><b/></a></r>");
    io:println(xmlArray is xmldata:Error ? xmlArray.message() : xmlArray.a);
    // @output [<a>1</a>,<a><b/></a>]

    // jBallerina reads malformed UTF-8 as U+FFFD (ballerina-platform/ballerina-library#9909).
    byte[][] badUtf8 = [
        [...("<p><name>".toBytes()), 0xE9, ...("</name></p>".toBytes())],
        [...("<p a='".toBytes()), 0xE9, ...("'><name>x</name></p>".toBytes())],
        [...("<p><!--".toBytes()), 0xE9, ...("--><name>x</name></p>".toBytes())],
        [...("<p><?pi ".toBytes()), 0xE9, ...("?><name>x</name></p>".toBytes())]
    ];
    foreach byte[] bytes in badUtf8 {
        Person|xmldata:Error fromBytes = xmldata:parseBytes(bytes);
        io:println(fromBytes is xmldata:Error ? fromBytes.message() : fromBytes.name);
    }
    // @output failed to parse xml: XML syntax error on line 1: invalid UTF-8
    // @output failed to parse xml: XML syntax error on line 1: invalid UTF-8
    // @output failed to parse xml: XML syntax error on line 1: invalid UTF-8
    // @output failed to parse xml: XML syntax error on line 1: invalid UTF-8

    // jBallerina narrows a byte value to 32 bits before the range check, so 4294967297 becomes 1
    // (ballerina-platform/ballerina-library#9827).
    Flag|xmldata:Error wideByte = xmldata:parseAsType(xml `<r><y>4294967297</y></r>`);
    io:println(wideByte is xmldata:Error ? wideByte.message() : wideByte);
    // @output 'string' value '4294967297' cannot be converted to 'byte'

    // jBallerina fails with an internal error when a fixed-length array slot is left without a filler value
    // (ballerina-platform/ballerina-library#9828).
    Pair|xmldata:Error shortParsed = xmldata:parseString("<r><p><name>x</name></p></r>");
    io:println(shortParsed is xmldata:Error ? shortParsed.message() : shortParsed);
    // @output array size is not compatible with the expected size
    Pair|xmldata:Error shortTraversed = xmldata:parseAsType(xml `<r><p><name>x</name></p></r>`);
    io:println(shortTraversed is xmldata:Error ? shortTraversed.message() : shortTraversed);
    // @output array size is not compatible with the expected size
    Pair|xmldata:Error full = xmldata:parseString("<r><p><name>x</name></p><p><name>y</name></p></r>");
    io:println(full is xmldata:Error ? full.message() : full);
    // @output {"p":[{"name":"x"},{"name":"y"}]}

    // jBallerina's parseAsType fails with an internal error when a fixed-length array field is absent (ballerina-platform/ballerina-library#9431).
    OptionalPair|xmldata:Error optionalParsed = xmldata:parseString("<r><title>t</title></r>");
    io:println(optionalParsed is xmldata:Error ? optionalParsed.message() : optionalParsed);
    // @output {"title":"t"}
    OptionalPair|xmldata:Error optionalTraversed = xmldata:parseAsType(xml `<r><title>t</title></r>`);
    io:println(optionalTraversed is xmldata:Error ? optionalTraversed.message() : optionalTraversed);
    // @output {"title":"t"}
    Pair|xmldata:Error absentParsed = xmldata:parseString("<r/>");
    io:println(absentParsed is xmldata:Error ? absentParsed.message() : absentParsed);
    // @output required field 'p' not present in XML
    Pair|xmldata:Error absentTraversed = xmldata:parseAsType(xml `<r/>`);
    io:println(absentTraversed is xmldata:Error ? absentTraversed.message() : absentTraversed);
    // @output required field 'p' not present in XML

    // jBallerina stores an element's attributes as a map in a rest field whose type cannot hold one (ballerina-platform/ballerina-library#6907).
    IntRest|xmldata:Error intRestParsed = xmldata:parseString("<r><a x=\"1\">5</a></r>");
    io:println(intRestParsed is xmldata:Error ? intRestParsed.message() : intRestParsed);
    // @output {"a":5}
    IntRest|xmldata:Error intRestTraversed = xmldata:parseAsType(xml `<r><a x="1">5</a></r>`);
    io:println(intRestTraversed is xmldata:Error ? intRestTraversed.message() : intRestTraversed);
    // @output {"a":5}
    IntArrayRest|xmldata:Error arrayRestTraversed = xmldata:parseAsType(xml `<r><a x="1">5</a><a x="2">6</a></r>`);
    io:println(arrayRestTraversed is xmldata:Error ? arrayRestTraversed.message() : arrayRestTraversed);
    // @output {"a":[5,6]}
    IntRest|xmldata:Error emptyRestParsed = xmldata:parseString("<r><a x=\"1\"/></r>");
    io:println(emptyRestParsed is xmldata:Error ? emptyRestParsed.message() : emptyRestParsed);
    // @output field 'a' cannot be converted into the type 'int'
    IntRest|xmldata:Error nestedRestParsed = xmldata:parseString("<r><a><b>1</b></a></r>");
    io:println(nestedRestParsed is xmldata:Error ? nestedRestParsed.message() : nestedRestParsed);
    // @output field 'a' cannot be converted into the type 'int'
    IntArrayRest|xmldata:Error emptyArrayRestTraversed = xmldata:parseAsType(xml `<r><a x="1"/></r>`);
    io:println(emptyArrayRestTraversed is xmldata:Error ? emptyArrayRestTraversed.message() : emptyArrayRestTraversed);
    // @output field 'a' cannot be converted into the type 'int'

    // jBallerina's parseString panics with a TypeCastError for a zero-length array rest field (ballerina-platform/ballerina-library#9923).
    ZeroRest|xmldata:Error zeroRestParsed = xmldata:parseString("<r><x>1</x><y>2</y></r>");
    io:println(zeroRestParsed is xmldata:Error ? zeroRestParsed.message() : zeroRestParsed);
    // @output {"x":1,"y":[]}
    ZeroRest|xmldata:Error zeroRestStrict = xmldata:parseString("<r><x>1</x><y>2</y></r>", {allowDataProjection: false});
    io:println(zeroRestStrict is xmldata:Error ? zeroRestStrict.message() : zeroRestStrict);
    // @output array size is not compatible with the expected size

    // jBallerina's parseString fails with an internal error for a rest element after a nested record (ballerina-platform/ballerina-library#9928).
    OpenOuter|xmldata:Error restAfterRecord = xmldata:parseString("<r><i><a>1</a></i><z>9</z></r>");
    io:println(restAfterRecord is xmldata:Error ? restAfterRecord.message() : restAfterRecord);
    // @output {"i":{"a":"1"},"z":9}

    // jBallerina's parseAsType projects into a zero-length array union member that parseString never uses (ballerina-platform/ballerina-library#9929).
    NilZero|xmldata:Error nilZeroTraversed = xmldata:parseAsType(xml `<r><a>5</a></r>`);
    io:println(nilZeroTraversed is xmldata:Error ? nilZeroTraversed.message() : nilZeroTraversed);
    // @output field 'a' cannot be converted into the type '[]?'
    NilZero|xmldata:Error nilZeroParsed = xmldata:parseString("<r><a>5</a></r>");
    io:println(nilZeroParsed is xmldata:Error ? nilZeroParsed.message() : nilZeroParsed);
    // @output 'string' value '5' cannot be converted to '[]?'
    ZeroOrInt|xmldata:Error zeroOrIntTraversed = xmldata:parseAsType(xml `<r><a>5</a></r>`);
    io:println(zeroOrIntTraversed is xmldata:Error ? zeroOrIntTraversed.message() : zeroOrIntTraversed);
    // @output {"a":5}

    // jBallerina returns the stream's own error; this port returns an xmldata:Error with its message.
    stream<byte[], error?> failing = new (new FailingAfterDocument());
    Person|xmldata:Error streamed = xmldata:parseStream(failing);
    io:println(streamed is xmldata:Error ? streamed.message() : "parsed");
    // @output stream failure
}
