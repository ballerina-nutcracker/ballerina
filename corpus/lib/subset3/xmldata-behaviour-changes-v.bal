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
}
