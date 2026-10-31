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

import ballerina/data.yaml;
import ballerina/io;

type Ints int[];

type Floats float[];

type Decimals decimal[];

type Strings string[];

type Booleans boolean[];

type Nils ()[];

type Bytes byte[];

type Signed8s int:Signed8[];

type Chars string:Char[];

type Finite ("A"|"B"|1|2.5|true)[];

type IntOrString (int|string)[];

type StringOrBoolean (string|boolean)[];

type Anydata anydata[];

type OptionalName record {|
    string name?;
|};

type OptionalAge record {|
    int age?;
|};

type NilableAge record {|
    int? age;
|};

type OneOrString (1|"1")[];

type TrueOrString (true|"true"|()|"null")[];

function show(anydata|error value) {
    if value is error {
        io:println(value.message());
    } else {
        io:println(value);
    }
}

public function main() {
    // Explicit types convert untagged scalars, quoted or not.
    show(yaml:parseString("[1, +2, -3, 007, \"4\"]", {}, Ints));
    show(yaml:parseString("[1, 1.5, 1e3, 1., .5, Infinity, -0.0, 0x1p3, \"2.5\"]", {}, Floats));
    show(yaml:parseString("[1.50, 1e3, 12345678901234567890.123456789]", {}, Decimals));
    show(yaml:parseString("[true, false]", {}, Booleans));
    show(yaml:parseString("[null, ]", {}, Nils));
    show(yaml:parseString("[0, 255]", {}, Bytes));
    show(yaml:parseString("[A, 1, 2.5, true]", {}, Finite));
    show(yaml:parseString("[1, x, 1.5]", {}, IntOrString));
    show(yaml:parseString("[true, x]", {}, StringOrBoolean));
    show(yaml:parseString("[a]", {}, Chars));
    // json and anydata infer int, float, nil, boolean and string in that order for plain scalars.
    show(yaml:parseString("[1, 1.5, null, true, text, \"123\", 99999999999999999999, +12, 0x1F, .inf]", {}, Anydata));
    // A quoted, block or !!str scalar is a string for json, anydata, unions and reserved words.
    show(yaml:parseString("a: '1'\nb: \"1.\"\nc: \"true\"\nd: 'null'\ne: '~'\nf: |\n  12\ng: !!str 3\nh: 4", {}, json));
    show(yaml:parseString("['1', \"x\", 2]", {}, IntOrString));
    show(yaml:parseString("[\"true\", 'True', \"~\"]", {}, Strings));
    show(yaml:parseString("name: 'null'", {allowDataProjection: {nilAsOptionalField: true}}, OptionalName));
    show(yaml:parseString("name: null", {allowDataProjection: {nilAsOptionalField: true}}, OptionalName));
    // A string scalar binds to a string singleton before converting to another singleton.
    show(yaml:parseString("[1, '1', \"1\"]", {}, OneOrString));
    show(yaml:parseString("[true, \"true\", null, 'null']", {}, TrueOrString));
    // A quoted null is a string, so it is not () for a nilable or optional int.
    show(yaml:parseString("age: null", {}, NilableAge));
    show(yaml:parseString("age: \"null\"", {}, NilableAge));
    show(yaml:parseString("age: null", {allowDataProjection: {nilAsOptionalField: true}}, OptionalAge));
    show(yaml:parseString("age: \"null\"", {allowDataProjection: {nilAsOptionalField: true}}, OptionalAge));
    // Values that do not convert are errors.
    show(yaml:parseString("[x]", {}, Ints));
    show(yaml:parseString("[1.5]", {}, Ints));
    show(yaml:parseString("[9223372036854775808]", {}, Ints));
    show(yaml:parseString("[1.0f]", {}, Floats));
    show(yaml:parseString("[.inf]", {}, Floats));
    show(yaml:parseString("[x]", {}, Decimals));
    show(yaml:parseString("[True]", {}, Booleans));
    show(yaml:parseString("[~]", {}, Nils));
    show(yaml:parseString("[256]", {}, Bytes));
    show(yaml:parseString("[200]", {}, Signed8s));
    show(yaml:parseString("[ab]", {}, Chars));
    show(yaml:parseString("[C]", {}, Finite));
    show(yaml:parseString("[True]", {}, Anydata));
    // The schema decides which plain words a string cannot be.
    show(yaml:parseString("[true]", {}, Strings));
    show(yaml:parseString("[True, ~]", {}, Strings));
    show(yaml:parseString("[True, ~]", {schema: yaml:JSON_SCHEMA}, Strings));
    show(yaml:parseString("[true, null]", {schema: yaml:FAILSAFE_SCHEMA}, Strings));
    show(yaml:parseString("a: 1\nb: True\nc: Null\nd: x", {schema: yaml:JSON_SCHEMA}));
    // Under the core schema, values that fit no type are dropped from maps.
    show(yaml:parseString("a: 1\nb: True\nc: ~\nd: x"));
}

// @output [1,2,-3,7,4]
// @output [1.0,1.5,1000.0,1.0,0.5,Infinity,-0.0,8.0,2.5]
// @output [1.50,1E+3,12345678901234567890.123456789]
// @output [true,false]
// @output [null]
// @output [0,255]
// @output ["A",1,2.5,true]
// @output [1,"x","1.5"]
// @output [true,"x"]
// @output ["a"]
// @output [1,1.5,null,true,"text","123",1e20,12,"0x1F",".inf"]
// @output {"a":"1","b":"1.","c":"true","d":"null","e":"~","f":"12\n","g":"3","h":4}
// @output ["1","x",2]
// @output ["true","True","~"]
// @output {"name":"null"}
// @output {}
// @output [1,"1","1"]
// @output [true,"true",null,"null"]
// @output {"age":null}
// @output incompatible value 'null' for type 'int?' in field 'age'
// @output {}
// @output incompatible value 'null' for type 'int' in field 'age'
// @output incompatible expected type 'int' for value 'x'
// @output incompatible expected type 'int' for value '1.5'
// @output incompatible expected type 'int' for value '9223372036854775808'
// @output incompatible expected type 'float' for value '1.0f'
// @output incompatible expected type 'float' for value '.inf'
// @output incompatible expected type 'decimal' for value 'x'
// @output incompatible expected type 'boolean' for value 'True'
// @output incompatible expected type '()' for value '~'
// @output incompatible expected type 'byte' for value '256'
// @output incompatible expected type 'lang.int:Signed8' for value '200'
// @output incompatible expected type 'lang.string:Char' for value 'ab'
// @output incompatible expected type 'true|1|2.5|"A"|"B"' for value 'C'
// @output incompatible expected type 'anydata' for value 'True'
// @output incompatible expected type 'string' for value 'true'
// @output incompatible expected type 'string' for value 'True'
// @output ["True","~"]
// @output ["true","null"]
// @output {"a":1,"b":"True","c":"Null","d":"x"}
// @output {"a":1,"d":"x"}
