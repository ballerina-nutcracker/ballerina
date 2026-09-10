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

import ballerina/io;

type Point record {|
    int x;
    int y;
|};

type Tagged record {|
    "a" kind;
|};

type ReadonlyId record {|
    readonly int id;
    string name;
|};

public function main() {
    var empty = {};
    io:println(empty); // @output {}
    io:println(empty is record {||}); // @output true

    record {||} nothing = {};
    var emptySpread = {...nothing};
    io:println(emptySpread); // @output {}
    io:println(emptySpread is record {||}); // @output true

    // A value written as a specific field is widened to its basic type; a value that came
    // through a spread keeps the type its source holds.
    var literal = {kind: "a"};
    io:println(literal is record {| string kind; |}); // @output true

    Tagged tagged = {kind: "a"};
    var spreadSingleton = {...tagged};
    io:println(spreadSingleton is record {| "a" kind; |}); // @output true

    // An explicitly readonly destination field survives inference next to a spread.
    Point p = {x: 1, y: 2};
    var withReadonly = {readonly tag: "t", ...p};
    io:println(withReadonly); // @output {"tag":"t","x":1,"y":2}
    io:println(withReadonly is record {| readonly string tag; int x; int y; |}); // @output true

    // Field-level readonly on the source is not transferred to the destination field.
    ReadonlyId src = {id: 7, name: "a"};
    var copied = {...src};
    copied.id = 8;
    io:println(copied); // @output {"id":8,"name":"a"}
}
