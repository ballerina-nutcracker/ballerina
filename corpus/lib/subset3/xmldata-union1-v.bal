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

type RecordUnion record {|
    record {| int a; |}|record {| string b; |} r;
    string id;
|};

type RecordUnionArray record {|
    (record {| int a; |}|record {| string b; |})[] r;
|};

type ScalarUnion record {|
    int|boolean|string v;
|};

type NilableRecord record {|
    record {| string x; |}? r;
|};

type A record {|
    int a;
|};

type B record {|
    string b;
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
    both("<x><r><b>s</b></r><id>1</id></x>", RecordUnion);
    // @output {"r":{"b":"s"},"id":"1"}
    // @output {"r":{"b":"s"},"id":"1"}
    both("<x><r><a>1</a></r><id>1</id></x>", RecordUnion);
    // @output {"r":{"a":1},"id":"1"}
    // @output {"r":{"a":1},"id":"1"}
    both("<x><r><c>1</c></r><id>1</id></x>", RecordUnion);
    // @output E: field 'r' cannot be converted
    // @output E: field 'r' cannot be converted
    both("<x><r><a>1</a></r><r><b>y</b></r></x>", RecordUnionArray);
    // @output {"r":[{"a":1},{"b":"y"}]}
    // @output {"r":[{"a":1},{"b":"y"}]}
    both("<x><v>1</v></x>", ScalarUnion);
    // @output {"v":1}
    // @output {"v":1}
    both("<x><v>true</v></x>", ScalarUnion);
    // @output {"v":true}
    // @output {"v":true}
    both("<x><v>text</v></x>", ScalarUnion);
    // @output {"v":"text"}
    // @output {"v":"text"}
    both("<x><r><x>y</x></r></x>", NilableRecord);
    // @output {"r":{"x":"y"}}
    // @output {"r":{"x":"y"}}
    A|B|xmldata:Error ab = xmldata:parseString("<x><b>s</b></x>");
    show(ab);
    // @output {"b":"s"}
    A|B|xmldata:Error ba = xmldata:parseAsType(xml `<x><a>1</a></x>`);
    show(ba);
    // @output {"a":1}
    A|B|xmldata:Error neither = xmldata:parseString("<x><c>s</c></x>");
    show(neither is xmldata:Error);
    // @output true

    // Text split by a comment is joined, then escaped once as BXml.toString does.
    A|B|xmldata:Error split = xmldata:parseString("<x><b>x &amp; y<!--c-->z</b></x>");
    show(split);
    // @output {"b":"x &amp; yz"}
    B|xmldata:Error splitTraversed = xmldata:parseAsType(checkpanic xml:fromString("<x><b>x &amp; y<!--c-->z</b></x>"));
    show(splitTraversed);
    // @output {"b":"x &amp; yz"}
}
