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

type WithX record {|
    int x;
|};

type WithY record {|
    string y;
|};

type ReadonlyMixed readonly & record {|
    int|string m;
|};

type ReadonlyInt readonly & record {|
    int m;
|};

public function main() {
    WithX|WithY value = {y: "a"};
    if value is WithY {
        var narrowed = {x: 1, ...value};
        io:println(narrowed); // @output {"x":1,"y":"a"}
        io:println(narrowed is record {| int x; string y; |}); // @output true
    }
    ReadonlyMixed mixed = {m: "a"};
    if !(mixed is ReadonlyInt) {
        // Excluding the int alternative leaves 'm' with string values only.
        record {|string m;|} narrowedMember = {...mixed};
        io:println(narrowedMember); // @output {"m":"a"}
    }
    WithX|WithY other = {x: 2};
    if !(other is WithY) {
        var narrowed = {...other, y: "b"};
        io:println(narrowed); // @output {"x":2,"y":"b"}
    }
}
