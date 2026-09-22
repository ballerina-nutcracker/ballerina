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

import ballerina/lang.array;
import ballerina/io;

// An omitted opaque default can appear inside an expression that is itself
// lowered to a default provider. Each default below compiles to an ordinary
// `$default$N` function in this module whose body calls the `$default$N`
// provider generated for `array:indexOf` in `lang.array`, so the two levels of
// default lowering nest.
isolated function letters() returns int[] => [10, 20, 30, 20];

// Record field defaults: positional omission, an explicit override of
// `startIndex`, and omission with the preceding argument named.
type Positions record {|
    int? first = array:indexOf(letters(), 20);
    int? fromThird = array:indexOf(letters(), 20, 2);
    int? named = array:indexOf(letters(), val = 30);
|};

// Function default parameters. The default expression for `pos` reads the
// preceding parameter `arr`, so its provider receives the caller's container
// and calls the shared `startIndex` provider with it.
function firstOf(int[] arr, int val, int? pos = arr.indexOf(val)) returns int? => pos;

function firstNamed(int[] arr, int? pos = arr.indexOf(val = 20)) returns int? => pos;

function firstQualified(int[] arr, int? pos = array:indexOf(arr, 30)) returns int? => pos;

public function main() {
    Positions p = {};
    io:println(p.first); // @output 1
    io:println(p.fromThird); // @output 3
    io:println(p.named); // @output 2

    // Explicitly supplied fields bypass the field defaults entirely.
    Positions q = {first: 7, fromThird: 8, named: 9};
    io:println(q.first); // @output 7
    io:println(q.fromThird); // @output 8
    io:println(q.named); // @output 9

    int[] values = letters();
    io:println(firstOf(values, 20)); // @output 1
    io:println(firstOf(values, 30)); // @output 2

    // Supplying `pos` bypasses the outer default, so the opaque call inside it
    // is never made.
    io:println(firstOf(values, 20, 99)); // @output 99

    io:println(firstNamed(values)); // @output 1
    io:println(firstQualified(values)); // @output 2
    io:println(firstOf(arr = values, val = 20)); // @output 1
}
