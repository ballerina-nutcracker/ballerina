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

type Strings string[];

function show(anydata|error value) {
    if value is error {
        io:println(value.message());
    } else {
        io:println(value);
    }
}

public function main() {
    // Core schema tags.
    show(yaml:parseString("[!!int 0x1F, !!int 0o17, !!int \"12\", !!float 1, !!float .inf, !!float -.Inf, !!bool True, !!null ~]"));
    show(yaml:parseString("!!float .NaN"));
    // !!str has no effect: the expected type still decides.
    show(yaml:parseString("!!str 123"));
    show(yaml:parseString("[!!str 123]", {}, Strings));
    show(yaml:parseString("!<tag:yaml.org,2002:str> bar"));
    show(yaml:parseString("!!seq [1, 2]"));
    show(yaml:parseString("!!map {a: 1}"));
    // JSON schema tags.
    show(yaml:parseString("[!!int 12, !!bool true, !!null null]", {schema: yaml:JSON_SCHEMA}));
    show(yaml:parseString("!!bool True", {schema: yaml:JSON_SCHEMA}));
    // A tagged value must belong to the expected type.
    show(yaml:parseString("[!!bool true]", {}, Ints));
    // Malformed and unsupported tags.
    show(yaml:parseString("!!int abc"));
    show(yaml:parseString("!!int 12abc"));
    show(yaml:parseString("!!int 99999999999999999999"));
    show(yaml:parseString("!!float abc1"));
    show(yaml:parseString("!foo bar"));
    show(yaml:parseString("!!binary aGk="));
    show(yaml:parseString("!!seq {a: 1}"));
    show(yaml:parseString("a: !!str\nb: 1"));
    show(yaml:parseString("!!int 1", {schema: yaml:FAILSAFE_SCHEMA}));
    show(yaml:parseString("!e!int 1"));
    // Tag directives may only name the core tag prefix.
    show(yaml:parseString("%TAG !e! tag:yaml.org,2002:\n---\n!e!int \"5\""));
    show(yaml:parseString("%TAG !e! tag:example.com,2000:\n---\n!e!int 5"));
    show(yaml:parseString("%TAG !e! tag:yaml.org,2002:\n%TAG !e! tag:yaml.org,2002:\n---\na"));
    show(yaml:parseString("%YAML 1.2\n---\na: 1"));
    show(yaml:parseString("%YAML 1.1\n---\na: 1"));
    show(yaml:parseString("%YAML 2.0\n---\na: 1"));
    show(yaml:parseString("%YAML 1.2\n%YAML 1.2\n---\na: 1"));
    show(yaml:parseString("%FOO bar\n---\na: 1"));
    show(yaml:parseString("%YAML 1.2\na: 1"));
}

// @output [31,15,12,1.0,Infinity,-Infinity,true,null]
// @output NaN
// @output 123
// @output ["123"]
// @output bar
// @output [1,2]
// @output {"a":1}
// @output [12,true,null]
// @output 'cannot cast True to boolean' at line: '1' column: '10'
// @output incompatible expected type 'int' for value 'true'
// @output 'cannot cast abc to int' at line: '1' column: '8'
// @output 'cannot cast 12abc to int' at line: '1' column: '10'
// @output 'cannot cast 99999999999999999999 to int' at line: '1' column: '25'
// @output 'cannot cast abc1 to float' at line: '1' column: '11'
// @output 'tag schema not supported' at line: '1' column: '7'
// @output 'tag schema not supported' at line: '1' column: '12'
// @output 'tag schema not supported' at line: '1' column: '8'
// @output 'tag schema not supported' at line: '2' column: '2'
// @output 'tag schema not supported' at line: '1' column: '6'
// @output 'tag handle is not defined' at line: '1' column: '6'
// @output 5
// @output 'custom tags not supported' at line: '1' column: '30'
// @output 'duplicate tag handle' at line: '2' column: '9'
// @output {"a":1}
// @output {"a":1}
// @output 'incompatible yaml version for the 1.2 parser' at line: '1' column: '9'
// @output 'YAML document version is already defined' at line: '2' column: '5'
// @output {"a":1}
// @output 'invalid directive document' at line: '2' column: '1'
