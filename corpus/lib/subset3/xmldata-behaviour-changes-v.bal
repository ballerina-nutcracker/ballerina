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

type Flag record {|
    byte y;
|};

type Pair record {|
    Person[2] p;
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
    // error as a plain error and panics on it, this port returns an xmldata:Error.
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

    // jBallerina leaves a required field whose type cannot hold () as () when its element is empty.
    UnionField|xmldata:Error emptyUnion = xmldata:parseString("<r><a></a></r>");
    io:println(emptyUnion is xmldata:Error ? emptyUnion.message() : "parsed");
    // @output field 'a' cannot be converted into the type '(int|string)'
    XmlField|xmldata:Error emptyXml = xmldata:parseString("<r><a/></r>");
    io:println(emptyXml is xmldata:Error ? emptyXml.message() : "parsed");
    // @output field 'a' cannot be converted into the type 'xml'

    // jBallerina narrows a byte value to 32 bits before the range check, so 4294967297 becomes 1.
    Flag|xmldata:Error wideByte = xmldata:parseAsType(xml `<r><y>4294967297</y></r>`);
    io:println(wideByte is xmldata:Error ? wideByte.message() : wideByte);
    // @output 'string' value '4294967297' cannot be converted to 'byte'

    // jBallerina fails with an internal error when a fixed-length array slot is left without a filler value.
    Pair|xmldata:Error shortParsed = xmldata:parseString("<r><p><name>x</name></p></r>");
    io:println(shortParsed is xmldata:Error ? shortParsed.message() : shortParsed);
    // @output array size is not compatible with the expected size
    Pair|xmldata:Error shortTraversed = xmldata:parseAsType(xml `<r><p><name>x</name></p></r>`);
    io:println(shortTraversed is xmldata:Error ? shortTraversed.message() : shortTraversed);
    // @output array size is not compatible with the expected size
    Pair|xmldata:Error full = xmldata:parseString("<r><p><name>x</name></p><p><name>y</name></p></r>");
    io:println(full is xmldata:Error ? full.message() : full);
    // @output {"p":[{"name":"x"},{"name":"y"}]}

    // jBallerina returns the stream's own error; this port returns an xmldata:Error with its message.
    stream<byte[], error?> failing = new (new FailingAfterDocument());
    Person|xmldata:Error streamed = xmldata:parseStream(failing);
    io:println(streamed is xmldata:Error ? streamed.message() : "parsed");
    // @output stream failure
}
