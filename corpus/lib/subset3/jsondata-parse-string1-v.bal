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

import ballerina/data.jsondata;
import ballerina/io;

type Item record {|
    string name;
    int qty;
|};

type Order record {|
    string id;
    Item[] items;
    map<string> tags;
|};

type Summary record {|
    string id;
|};

type Shape record {|
    string kind;
    record {|int side;|}|record {|int radius;|} size;
|};

type OptionalNote record {|
    string id;
    string note?;
|};

type NilableNote record {|
    string id;
    string? note;
|};

type IntField record {|
    int a;
|};

type MapField record {|
    map<int> m;
|};

function show(anydata|error v) {
    if v is error {
        io:println("error: ", v.message());
    } else {
        io:println(v);
    }
}

// jBallerina keeps the characters of a token it failed in the middle of and prefixes them to the next
// token it reads; parsing a document whose first token is discarded keeps later results comparable.
function reset() {
    json|error discarded = jsondata:parseString("{\"\":0}");
    if discarded is error {
        panic discarded;
    }
}

public function main() {
    string orderJson = string `{
        "id": "o-1",
        "items": [{"name": "pen", "qty": 2}, {"name": "ink", "qty": 1}],
        "tags": {"priority": "high"}
    }`;
    Order|error o1 = jsondata:parseString(orderJson);
    show(o1);
    // @output {"id":"o-1","items":[{"name":"pen","qty":2},{"name":"ink","qty":1}],"tags":{"priority":"high"}}
    Summary|error o2 = jsondata:parseString(orderJson);
    show(o2);
    // @output {"id":"o-1"}
    Summary|error o3 = jsondata:parseString(orderJson, {allowDataProjection: false});
    show(o3);
    // @output error: undefined field 'items'

    Shape|error s1 = jsondata:parseString(string `{"kind":"square","size":{"side":2}}`);
    show(s1);
    // @output {"kind":"square","size":{"side":2}}
    Shape|error s2 = jsondata:parseString(string `{"kind":"circle","size":{"radius":3}}`);
    show(s2);
    // @output {"kind":"circle","size":{"radius":3}}
    (int|string)[]|error s3 = jsondata:parseString(string `[1, "two", 3]`);
    show(s3);
    // @output [1,"two",3]
    IntField|string[]|error s4 = jsondata:parseString(string `["x", "y"]`);
    show(s4);
    // @output ["x","y"]
    IntField|string[]|error s5 = jsondata:parseString(string `{"a": 4}`);
    show(s5);
    // @output {"a":4}

    json|error j1 = jsondata:parseString(string `{"a":[1,-2,3.5,true,false,null,"s",{"b":[]},{}]}`);
    show(j1);
    // @output {"a":[1,-2,3.5,true,false,null,"s",{"b":[]},{}]}
    anydata|error j2 = jsondata:parseString(string `[{"x":1},[2,[3]],"y"]`);
    show(j2);
    // @output [{"x":1},[2,[3]],"y"]
    map<json>|error j3 = jsondata:parseString(string `{"a":{"b":{"c":[1,2]}}}`);
    show(j3);
    // @output {"a":{"b":{"c":[1,2]}}}
    json[]|error j4 = jsondata:parseString("[]");
    show(j4);
    // @output []
    map<int>|error j5 = jsondata:parseString("{}");
    show(j5);
    // @output {}
    json|error j6 = jsondata:parseString(" \t\r\n[ 1 ,\n 2 ] \n");
    show(j6);
    // @output [1,2]

    string|error e1 = jsondata:parseString(string `"q\"b\\s\/b\bf\fn\nr\rt\t"`);
    show(e1 is string ? e1.toBytes() : e1);
    // @output [113,34,98,92,115,47,98,8,102,12,110,10,114,13,116,9]
    string|error e2 = jsondata:parseString(string `"\u00e9\u0041\ud83d\ude00"`);
    show(e2);
    // @output éA😀
    map<string>|error e3 = jsondata:parseString(string `{"k\u0065y\n":"v\u00e9", "arr":"x"}`);
    show(e3 is map<string> ? e3.keys()[0].toBytes() : e3);
    // @output [107,101,121,10]
    string[]|error e4 = jsondata:parseString(string `["a\"", "\u0062", "\\"]`);
    show(e4 is string[] ? e4[0].toBytes() : e4);
    // @output [97,34]
    show(e4 is string[] ? e4[2].toBytes() : e4);
    // @output [92]

    int|error n1 = jsondata:parseString("42");
    show(n1);
    // @output 42
    int|error n2 = jsondata:parseString("-7");
    show(n2);
    // @output -7
    int|error n3 = jsondata:parseString("+5");
    show(n3);
    // @output 5
    int|error n4 = jsondata:parseString("1.5");
    show(n4);
    // @output error: incompatible expected type 'int' for value '1.5'
    int|error n5 = jsondata:parseString("9223372036854775808");
    show(n5);
    // @output error: incompatible expected type 'int' for value '9223372036854775808'
    float|error n6 = jsondata:parseString("2");
    show(n6);
    // @output 2.0
    float|error n7 = jsondata:parseString("1.5e2");
    show(n7);
    // @output 150.0
    float|error n8 = jsondata:parseString("1.5f");
    show(n8);
    // @output error: incompatible expected type 'float' for value '1.5f'
    float|error n23 = jsondata:parseString("1e400");
    show(n23);
    // @output Infinity
    float|error n24 = jsondata:parseString("+NaN");
    show(n24);
    // @output NaN
    float|error n25 = jsondata:parseString("-NaN");
    show(n25);
    // @output NaN
    decimal|error n9 = jsondata:parseString("12.340");
    show(n9);
    // @output 12.340
    decimal|error n10 = jsondata:parseString("1e2");
    show(n10);
    // @output 1E+2
    decimal|error n11 = jsondata:parseString("1.");
    show(n11);
    // @output 1
    decimal|error n12 = jsondata:parseString("0x10");
    show(n12);
    // @output error: incompatible expected type 'decimal' for value '0x10'
    json|error n13 = jsondata:parseString("7");
    show(n13);
    // @output 7
    json|error n14 = jsondata:parseString("7.25");
    show(n14);
    // @output 7.25
    json|error n15 = jsondata:parseString("-0.0");
    show(n15);
    // @output -0.0
    byte|error n16 = jsondata:parseString("255");
    show(n16);
    // @output 255
    byte|error n17 = jsondata:parseString("256");
    show(n17);
    // @output error: incompatible expected type 'byte' for value '256'
    int:Signed8|error n18 = jsondata:parseString("-129");
    show(n18);
    // @output error: incompatible expected type 'lang.int:Signed8' for value '-129'
    int:Unsigned16|error n19 = jsondata:parseString("65535");
    show(n19);
    // @output 65535
    int|float|error n20 = jsondata:parseString("3");
    show(n20 is int);
    // @output true
    int|float|error n21 = jsondata:parseString("3.0");
    show(n21 is float);
    // @output true
    decimal|string|error n22 = jsondata:parseString("3.0");
    show(n22 is decimal);
    // @output true

    boolean|error b1 = jsondata:parseString("true");
    show(b1);
    // @output true
    boolean|error b2 = jsondata:parseString("false");
    show(b2);
    // @output false
    boolean|error b3 = jsondata:parseString("0");
    show(b3);
    // @output false
    boolean|error b4 = jsondata:parseString("tru");
    show(b4);
    // @output error: incompatible expected type 'boolean' for value 'tru'
    boolean|error b5 = jsondata:parseString("fals");
    show(b5);
    // @output error: incompatible expected type 'boolean' for value 'fals'
    ()|error b6 = jsondata:parseString("null");
    show(b6);
    // @output 
    ()|error b7 = jsondata:parseString("nul");
    show(b7);
    // @output error: incompatible expected type '()' for value 'nul'
    int|error b8 = jsondata:parseString("x1");
    show(b8);
    // @output error: incompatible expected type 'int' for value 'x1'

    string|error t1 = jsondata:parseString(string `"plain"`);
    show(t1);
    // @output plain
    int|error t2 = jsondata:parseString(string `"5"`);
    show(t2);
    // @output error: incompatible expected type 'int' for value '5'
    string|error t3 = jsondata:parseString("5");
    show(t3);
    // @output error: incompatible expected type 'string' for value '5'
    string:Char|error t4 = jsondata:parseString(string `"x"`);
    show(t4);
    // @output x
    string:Char|error t5 = jsondata:parseString(string `"xy"`);
    show(t5);
    // @output error: incompatible expected type 'lang.string:Char' for value 'xy'
    "A"|"B"|"C"|error t6 = jsondata:parseString(string `"B"`);
    show(t6);
    // @output B
    "A"|"B"|"C"|error t7 = jsondata:parseString(string `"D"`);
    show(t7);
    // @output error: incompatible expected type '("A"|"B"|"C")' for value 'D'
    1|2|3|error t8 = jsondata:parseString("2");
    show(t8);
    // @output 2
    1|2|3|error t9 = jsondata:parseString("4");
    show(t9);
    // @output error: incompatible expected type '(1|2|3)' for value '4'
    true|error t10 = jsondata:parseString("true");
    show(t10);
    // @output true
    string?|error t11 = jsondata:parseString("null");
    show(t11);
    // @output 

    OptionalNote|error p1 = jsondata:parseString(string `{"id":"1","note":null}`,
            {allowDataProjection: {nilAsOptionalField: true}});
    show(p1);
    // @output {"id":"1"}
    OptionalNote|error p2 = jsondata:parseString(string `{"id":"1","note":null}`);
    show(p2);
    // @output error: incompatible value 'null' for type 'string' in field 'note'
    NilableNote|error p3 = jsondata:parseString(string `{"id":"1"}`, {allowDataProjection: {absentAsNilableType: true}});
    show(p3);
    // @output {"id":"1","note":null}
    NilableNote|error p4 = jsondata:parseString(string `{"id":"1"}`);
    show(p4);
    // @output error: required field 'note' not present in JSON
    IntField|error p5 = jsondata:parseString(string `{"a":"x"}`);
    show(p5);
    // @output error: incompatible value 'x' for type 'int' in field 'a'
    IntField|error p6 = jsondata:parseString(string `{"a":1,"b":2}`, {allowDataProjection: false});
    show(p6);
    // @output error: undefined field 'b'
    IntField|error p7 = jsondata:parseString(string `{"a":1,"skip":{"x":[1,{"y":2}]},"list":[[1],{"z":3}]}`);
    show(p7);
    // @output {"a":1}
    int[2]|error p8 = jsondata:parseString("[1, 2, [3, 4], {\"a\": 5}, 6]");
    show(p8);
    // @output [1,2]
    int[2]|error p9 = jsondata:parseString("[1, 2, 3]", {allowDataProjection: false});
    show(p9);
    // @output error: array size is not compatible with the expected size
    [int, string]|error p10 = jsondata:parseString(string `[1, "a", 2]`, {allowDataProjection: false});
    show(p10);
    // @output error: array size is not compatible with the expected size
    [int, string]|error p11 = jsondata:parseString(string `[9]`);
    show(p11);
    // @output [9,""]
    MapField|error p12 = jsondata:parseString(string `{"m":{"a":"x"}}`);
    show(p12);
    // @output error: incompatible expected type 'int' for value 'x'
    int|error p13 = jsondata:parseString("[1]");
    show(p13);
    // @output error: invalid type 'int' expected 'list type'
    int|error p15 = jsondata:parseString(string `{"a":1}`);
    show(p15);
    // @output error: invalid type 'int' expected 'map type'
    record {| int[] a; |}|error p16 = jsondata:parseString(string `{"a":{"b":1}}`);
    show(p16);
    // @output error: invalid type for field 'a'
    IntField|error p14 = jsondata:parseString(string `{"a":1,"a":2}`);
    show(p14);
    // @output {"a":2}

    json|error x1 = jsondata:parseString("");
    show(x1);
    // @output error: 'empty JSON document' at line: '1' column: '1'
    json|error x2 = jsondata:parseString("  ");
    show(x2);
    // @output error: 'empty JSON document' at line: '1' column: '3'
    json|error x3 = jsondata:parseString("{\"a\":1} x");
    show(x3);
    // @output error: 'JSON document has already ended' at line: '1' column: '10'
    json|error x4 = jsondata:parseString("{\"a\" 1}");
    show(x4);
    // @output error: 'expected ':'' at line: '1' column: '6'
    json|error x5 = jsondata:parseString("{\"a\":1 \"b\":2}");
    show(x5);
    // @output error: 'expected ',' or '}'' at line: '1' column: '9'
    json|error x6 = jsondata:parseString("[1 2]");
    show(x6);
    // @output error: 'expected ',' or ']'' at line: '1' column: '5'
    json|error x7 = jsondata:parseString("{1:2}");
    show(x7);
    // @output error: 'expected '"' or '}'' at line: '1' column: '2'
    json|error x8 = jsondata:parseString("{\"a\":1,}");
    show(x8);
    // @output error: 'expected '"'' at line: '1' column: '9'
    json|error x10 = jsondata:parseString("{\"a\":1");
    show(x10);
    // @output error: 'unexpected end of JSON document' at line: '1' column: '8'
    reset();
    json|error x11 = jsondata:parseString("{\"ab");
    show(x11);
    // @output error: 'unexpected end of JSON document' at line: '1' column: '5'
    reset();
    json|error x12 = jsondata:parseString("[\"ab");
    show(x12);
    // @output error: 'unexpected end of JSON document' at line: '1' column: '5'
    reset();
    json|error x13 = jsondata:parseString("\"ab");
    show(x13);
    // @output error: 'unexpected end of JSON document' at line: '1' column: '4'
    reset();
    json|error x14 = jsondata:parseString("[1, 2");
    show(x14);
    // @output error: 'unexpected end of JSON document' at line: '1' column: '8'
    reset();
    json|error x15 = jsondata:parseString("\"a\\qb\"");
    show(x15);
    // @output error: 'expected 'escaped characters'' at line: '1' column: '4'
    reset();
    json|error x16 = jsondata:parseString("\"\\u00zz\"");
    show(x16);
    // @output error: 'expected 'hexadecimal value of an unicode character'' at line: '1' column: '6'
    reset();
    json|error x17 = jsondata:parseString("{\n  \"a\": 1,\n  \"b\" 2\n}");
    show(x17);
    // @output error: 'expected ':'' at line: '3' column: '7'
    json|error x18 = jsondata:parseString("[1]\n\n  ]");
    show(x18);
    // @output error: 'JSON document has already ended' at line: '3' column: '3'
    json|error x19 = jsondata:parseString("{\"a\":\"x\"\"b\"}");
    show(x19);
    // @output error: 'expected ',' or '}'' at line: '1' column: '9'
    json|error x20 = jsondata:parseString("[\"a\" \"b\"]");
    show(x20);
    // @output error: 'expected ',' or ']'' at line: '1' column: '6'
    json|error x21 = jsondata:parseString("{\"\\u0061\\x\":1}");
    show(x21);
    // @output error: 'expected 'escaped characters'' at line: '1' column: '10'
    reset();
    json|error x22 = jsondata:parseString("[\"\\u12\"]");
    show(x22);
    // @output error: 'expected 'hexadecimal value of an unicode character'' at line: '1' column: '7'
    reset();
    json|error x23 = jsondata:parseString("{\"a\":\"\\u12\"}");
    show(x23);
    // @output error: 'expected 'hexadecimal value of an unicode character'' at line: '1' column: '11'
    reset();
    json|error x24 = jsondata:parseString("{\"\\u12\":1}");
    show(x24);
    // @output error: 'expected 'hexadecimal value of an unicode character'' at line: '1' column: '7'
    reset();
    json|error x25 = jsondata:parseString("{\"a\":\"b\\");
    show(x25);
    // @output error: 'expected 'escaped characters'' at line: '1' column: '9'
    reset();
}
