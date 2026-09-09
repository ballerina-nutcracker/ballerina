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

// A langlib method call is fine inside a default value expression when its
// receiver is a preceding parameter or a call result, because those have a
// resolved type. A module-level variable receiver does not; see
// langlib-method-default-module-var-ref-v.bal and
// https://github.com/ballerina-nutcracker/ballerina/issues/915.
isolated function letters() returns int[] => [10, 20, 30, 20];

// Receiver is the preceding parameter.
function lengthOf(int[] arr, int n = arr.length()) returns int => n;

// Receiver is a call result, including an opaque function with its own
// omitted default.
function firstOf(int? pos = letters().indexOf(30)) returns int? => pos;

class Holder {
    int? pos;

    function init(int? pos = letters().indexOf(20)) {
        self.pos = pos;
    }
}

type Positions record {|
    int? found = letters().indexOf(30);
|};

public function main() {
    io:println(lengthOf([1, 2, 3])); // @output 3
    io:println(lengthOf([1, 2, 3], 99)); // @output 99

    io:println(firstOf()); // @output 2
    io:println(firstOf(7)); // @output 7

    Holder h = new;
    io:println(h.pos); // @output 1

    Positions p = {};
    io:println(p.found); // @output 2

    // The same call in a function body, for contrast.
    io:println(letters().length()); // @output 4
}
