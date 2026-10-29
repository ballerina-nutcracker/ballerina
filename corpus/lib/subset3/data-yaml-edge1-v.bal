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

type Config record {| string name; int age?; |};

annotation record {| string value; |} FieldName on record field;

type Person record {|
    @FieldName {value: "other"}
    @yaml:Name {value: "full-name"}
    string name;
|};

function repeat(string s, int n) returns string {
    string out = "";
    foreach int _ in 0 ..< n {
        out += s;
    }
    return out;
}

function show(anydata|error value) {
    if value is error {
        io:println(value.message());
    } else {
        io:println(value);
    }
}

public function main() {
    json|yaml:Error deepFlow = yaml:parseString(repeat("[", 10001));
    show(deepFlow);
    json|yaml:Error deepBlock = yaml:parseString(repeat("- ", 10001) + "x");
    show(deepBlock);
    json|yaml:Error nested = yaml:parseString(repeat("[", 100) + "1" + repeat("]", 100));
    show(nested is json);

    string laughs = "a: &a [x, x, x, x, x, x, x, x, x, x]\n";
    string previous = "a";
    foreach string name in ["b", "c", "d", "e", "f", "g", "h", "i"] {
        laughs += name + ": &" + name + " [" + repeat("*" + previous + ", ", 9) + "*" + previous + "]\n";
        previous = name;
    }
    json|yaml:Error expanded = yaml:parseString(laughs);
    show(expanded);
    json|yaml:Error aliased = yaml:parseString("a: &a [1, 2]\nb: [*a, *a, *a]");
    show(aliased);
    string aliasHeavy = "---\na: &a [1, 2, 3, 4, 5, 6, 7, 8, 9, 10]\nb: [*a" + repeat(", *a", 84) + "]\n";
    json[]|yaml:Error aliasHeavyStream = yaml:parseString(repeat(aliasHeavy, 500));
    show(aliasHeavyStream is error ? aliasHeavyStream : aliasHeavyStream.length());

    json|yaml:Error crossDocument = yaml:parseString("--- &x 1\n--- *x");
    show(crossDocument);
    json|yaml:Error redefined = yaml:parseString("--- &x 1\n--- &x 2", {allowAnchorRedefinition: false});
    show(redefined);

    anydata[] cyclic = [1];
    cyclic.push(cyclic);
    show(yaml:toYamlString(cyclic));
    show(yaml:toYamlString({a: cyclic}, {flowStyle: true}));
    anydata[] shared = [1];
    show(yaml:toYamlString([shared, shared]));
    show(yaml:toYamlString({a: 1}, {indentationPolicy: 1025}));

    string|yaml:Error empty = yaml:toYamlString({a: [], b: {}, c: [[], {}]});
    show(empty);
    if empty is string {
        json|yaml:Error back = yaml:parseString(empty);
        show(back);
    }
    show(yaml:toYamlString({}));
    show(yaml:toYamlString([]));

    string|yaml:Error numeric = yaml:toYamlString([".inf", "-.Inf", ".NaN", "0x1F", "0o17", "+12", "1.", ".5e3", "1_000"]);
    show(numeric);

    Config|yaml:Error emptyValue = yaml:parseString("name: a\nage:", {allowDataProjection: {nilAsOptionalField: true}});
    show(emptyValue);
    Config|yaml:Error flowEmpty = yaml:parseString("{name: a, age: }", {allowDataProjection: {nilAsOptionalField: true}});
    show(flowEmpty);

    Person|yaml:Error person = yaml:parseString("full-name: Ann", {}, Person);
    show(person);
}

// @output 'exceeded max depth of 10000' at line: '1' column: '10,000'
// @output 'exceeded max depth of 10000' at line: '1' column: '20,000'
// @output true
// @output 'document contains excessive aliasing' at line: '4' column: '9'
// @output {"a":[1,2],"b":[[1,2],[1,2],[1,2]]}
// @output 500
// @output 'anchor does not exist' at line: '2' column: '5'
// @output [1,2]
// @output cannot write a value that has a cyclic reference
// @output cannot write a value that has a cyclic reference
// @output -
// @output   - 1
// @output -
// @output   - 1
// @output indentationPolicy must not be greater than 1024
// @output a: []
// @output b: {}
// @output c:
// @output - []
// @output - {}
// @output {"a":[],"b":{},"c":[[],{}]}
// @output {}
// @output []
// @output - ".inf"
// @output - "-.Inf"
// @output - ".NaN"
// @output - "0x1F"
// @output - "0o17"
// @output - "+12"
// @output - "1."
// @output - ".5e3"
// @output - 1_000
// @output {"name":"a"}
// @output {"name":"a"}
// @output {"name":"Ann"}
