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

type Inner record {|
    int age;
|};

type Closed record {|
    string name;
    int age;
|};

type Nested record {|
    string name;
    Inner inner;
    int[] nums?;
|};

type Open record {
    string name;
};

type Optional record {|
    string name;
    int? nick;
    int age?;
|};

type Rest record {|
    string name;
    int...;
|};

type Named record {|
    @yaml:Name {value: "full-name"}
    string name;
    int age;
|};

type ReadonlyRecord readonly & record {|
    string a;
    int[] b;
|};

function show(anydata|error value) {
    if value is error {
        io:println(value.message());
    } else {
        io:println(value);
    }
}

public function main() {
    show(yaml:parseString("name: a\nage: 2", {}, Closed));
    // Unknown keys are dropped from closed records unless projection is disabled.
    show(yaml:parseString("name: a\nage: 2\nextra: x", {}, Closed));
    show(yaml:parseString("name: a\nage: 2\nextra: x", {allowDataProjection: false}, Closed));
    show(yaml:parseString("name: a\nage: 2\nextra: x", {}, Open));
    // Values that do not convert to the rest type are dropped.
    show(yaml:parseString("name: a\nx: 1\ny: z\nz: ~", {}, Rest));
    show(yaml:parseString("name: a", {}, Closed));
    show(yaml:parseString("name: a\nage: x", {}, Closed));
    show(yaml:parseString("name: a\ninner:\n  age: x", {}, Nested));
    show(yaml:parseString("name: a\ninner:\n  age: 1\nnums: [1, x]", {}, Nested));
    show(yaml:parseString("name: [a]\ninner:\n  age: 1", {}, Nested));
    show(yaml:parseString("name: {a: 1}\ninner:\n  age: 1", {}, Nested));
    show(yaml:parseString("name: a\ninner: 5", {}, Nested));
    // Optional and nilable fields.
    show(yaml:parseString("name: a\nnick: null", {}, Optional));
    show(yaml:parseString("name: a\nnick:", {}, Optional));
    show(yaml:parseString("name: a", {}, Optional));
    show(yaml:parseString("name: a", {allowDataProjection: {absentAsNilableType: true}}, Optional));
    show(yaml:parseString("name: a\nnick: 1\nage: null", {}, Optional));
    show(yaml:parseString("name: a\nnick: 1\nage: null", {allowDataProjection: {nilAsOptionalField: true}}, Optional));
    // A Name annotation renames a field of the record type the typedesc names.
    show(yaml:parseString("full-name: a\nage: 2", {}, Named));
    show(yaml:parseString("name: a\nage: 2", {}, Named));
    show(yaml:parseString("a: x\nb: [1, 2]", {}, ReadonlyRecord));
    // Maps.
    map<int>|yaml:Error ints = yaml:parseString("a: 1\nb: x\nc: 2");
    show(ints);
    map<string>|yaml:Error strings = yaml:parseString("a: true\nb: 1\nc: null");
    show(strings);
    map<int>|yaml:Error scalar = yaml:parseString("5");
    show(scalar);
    map<int>|yaml:Error list = yaml:parseString("[1]");
    show(list);
    // Keys are strings; an empty key is "null".
    show(yaml:parseString("1: a\nnull: b\n\"a b\": c\n? d\n: e"));
    show(yaml:parseString("a: 1\na: 2"));
    show(yaml:parseString("a: 1\na: 2", {allowMapEntryRedefinition: true}));
    show(yaml:parseString("a: 1\nb: 2\na: 3", {allowMapEntryRedefinition: true}));
    show(yaml:parseString("{a, b: 2, c: }"));
    // Non-list targets read the first document only.
    show(yaml:parseString("name: a\nage: 1\n---\nname: b\nage: 2", {}, Closed));
}

// @output {"name":"a","age":2}
// @output {"name":"a","age":2}
// @output undefined field 'extra'
// @output {"name":"a","age":2,"extra":"x"}
// @output {"name":"a","x":1}
// @output required field 'age' not present in YAML
// @output incompatible value 'x' for type 'int' in field 'age'
// @output incompatible value 'x' for type 'int' in field 'inner.age'
// @output incompatible value 'x' for type 'int' in field 'nums'
// @output invalid type 'string' expected 'list type'
// @output invalid type for field 'name'
// @output incompatible value '5' for type '{| age: int, never... |}' in field 'inner'
// @output {"name":"a","nick":null}
// @output {"name":"a","nick":null}
// @output required field 'nick' not present in YAML
// @output {"name":"a","nick":null}
// @output incompatible value 'null' for type 'int' in field 'age'
// @output {"name":"a","nick":1}
// @output {"name":"a","age":2}
// @output required field 'name' not present in YAML
// @output {"a":"x","b":[1,2]}
// @output {"a":1,"c":2}
// @output {"b":"1"}
// @output incompatible expected type '{| int... |}' for value '5'
// @output invalid type '{| int... |}' expected 'list type'
// @output {"1":"a","null":"b","a b":"c","d":"e"}
// @output 'cannot have duplicate map entries for '${key.toString()}' at line: '2' column: '3'
// @output {"a":2}
// @output {"a":3,"b":2}
// @output {"a":null,"b":2,"c":null}
// @output {"name":"a","age":1}
