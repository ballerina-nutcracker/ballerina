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

type A record {|
    int a;
|};

type B record {|
    int b;
|};

type Ints int[];

type Fixed int[2];

type Pair [int, string];

type PairRest [int, string...];

type Tuple [A, B];

type ReversedTuple [B, A];

type ReversedRest [B, A...];

type IntOrInts int|int[];

type Anydata anydata;

type Strings string[];

type Decimals decimal[];

function show(anydata|error value) {
    if value is error {
        io:println(value.message());
    } else {
        io:println(value);
    }
}

public function main() {
    show(yaml:parseString("[1, 2]", {}, Ints));
    show(yaml:parseString("- 1\n- 2", {}, Ints));
    show(yaml:parseString("5", {}, Ints));
    show(yaml:parseString("a: 1", {}, Ints));
    // Closed arrays and tuples drop extra members unless projection is disabled, and pad missing ones.
    show(yaml:parseString("[1, 2, 3]", {}, Fixed));
    show(yaml:parseString("[1, 2, 3]", {allowDataProjection: false}, Fixed));
    show(yaml:parseString("[1]", {}, Fixed));
    show(yaml:parseString("[1, a, b]", {}, Pair));
    show(yaml:parseString("[1, a, b]", {allowDataProjection: false}, Pair));
    show(yaml:parseString("[1, a, b]", {}, PairRest));
    show(yaml:parseString("[1]", {}, Pair));
    // Values in lists are converted from their own text.
    show(yaml:parseString("[007, 1.50, x]", {}, Strings));
    show(yaml:parseString("[1.50, 12345678901234567890.123456789]", {}, Decimals));
    // Several documents bind to a list one document per member.
    show(yaml:parseString("1\n---\n2\n---\n3", {}, Ints));
    show(yaml:parseString("--- 1\n--- 2", {}, Ints));
    show(yaml:parseString("a: 1\n---\nb: 2", {}, Tuple));
    show(yaml:parseString("a: 1\n---\nb: 2", {}, ReversedTuple));
    show(yaml:parseString("a: 1\n---\nb: 2", {allowDataProjection: {enableYamlStreamReorder: true}}, ReversedTuple));
    show(yaml:parseString("a: 1\n---\nb: 2\n---\na: 3", {allowDataProjection: {enableYamlStreamReorder: true}}, ReversedRest));
    show(yaml:parseString("a: 1\n---\na: 2", {allowDataProjection: {enableYamlStreamReorder: true}}, ReversedTuple));
    show(yaml:parseString("5", {}, IntOrInts));
    show(yaml:parseString("[5]", {}, IntOrInts));
    show(yaml:parseString("5\n---\n6", {}, IntOrInts));
    // json and anydata take one document as it is and several as a list.
    show(yaml:parseString("a: 1\n---\nb: 2", {}, Anydata));
    show(yaml:parseString("---\na: 1", {}, Anydata));
    show(yaml:parseString("[1, 2]\n---\n[3]", {}, Anydata));
    show(yaml:parseString("---", {}, Anydata));
    show(yaml:parseString("...", {}, Anydata));
    show(yaml:parseString("---\n---", {}, Anydata));
    show(yaml:parseString("# only a comment", {}, Anydata));
    show(yaml:parseString("", {}, Anydata));
    json value = checkpanic yaml:parseString("a: [1, {b: c}]");
    show(value);
}

// @output [1,2]
// @output [1,2]
// @output incompatible expected type 'int[]' for value '5'
// @output invalid type 'int[]' expected 'map type'
// @output [1,2]
// @output array size is not compatible with the expected size
// @output [1,0]
// @output [1,"a"]
// @output array size is not compatible with the expected size
// @output [1,"a","b"]
// @output [1,""]
// @output ["007","1.50","x"]
// @output [1.50,12345678901234567890.123456789]
// @output [1,2,3]
// @output [1,2]
// @output [{"a":1},{"b":2}]
// @output required field 'b' not present in YAML
// @output [{"b":2},{"a":1}]
// @output [{"b":2},{"a":1},{"a":3}]
// @output array size is not compatible with the expected size
// @output 5
// @output [5]
// @output [5,6]
// @output [{"a":1},{"b":2}]
// @output {"a":1}
// @output [[1,2],[3]]
// @output 
// @output 
// @output [null,null]
// @output []
// @output 'invalid yaml document' at line: '1' column: '0'
// @output {"a":[1,{"b":"c"}]}
