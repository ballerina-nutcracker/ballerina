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

// KNOWN LIMITATION: per the lang.value spec, cloneWithType/fromJsonWithType must pick the
// leftmost union member that matches ("the leftmost matching type descriptor is used as the
// inherent type"), including recursively inside nested unions -- so `int|float` and `float|int`
// must give different results when the source is ambiguous (e.g. a decimal with no exact-type
// member). This implementation does not yet track declared union member order through
// semtypes.SemType/TypeDesc, so it always resolves ambiguous numeric candidates via a fixed
// internal order regardless of how the union was declared. This test locks in the CURRENT
// (order-insensitive) behaviour so a future fix is a deliberate, visible change to this file
// rather than a silent regression. See issue #657:
// https://github.com/ballerina-nutcracker/ballerina/issues/657
import ballerina/io;

type R3 record {|
    int f3;
|};
type R4 record {|
    float f3;
|};
type R5 record {|
    int f3;
|};
type R6 record {|
    string f3;
|};

type R1 record {|
    R3|R4 f1;
|};
type R2 record {|
    R5|R6 f2;
|};
type R R1|R2;

// Same shape as R1, but with the nested union's declared order reversed.
type R1b record {|
    R4|R3 f1;
|};
type Rb R1b|R2;

public function main() returns error? {
    decimal d = 1.5;

    // Value only has key f1, so R2 (which requires f2) cannot match; must resolve via R1.
    // Within R1, f1's target is R3|R4 (int f3 | float f3): f3 = decimal 1.5 matches neither
    // exactly, but numeric-converts to both, so the choice depends on leftmost member order.
    anydata v1 = {f1: {f3: d}};
    R r1 = check v1.cloneWithType();
    io:println(r1); // @output {"f1":{"f3":2}}

    // Same value, but the nested union is declared R4|R3 instead of R3|R4. Per spec this must
    // give {"f1":{"f3":1.5}} (see the known-limitation note above); this implementation
    // doesn't diverge yet.
    Rb r1b = check v1.cloneWithType();
    io:println(r1b); // @output {"f1":{"f3":2}}

    // Value only has key f2, so R1 cannot match; must resolve via R2. Within R2, f2's target
    // is R5|R6 (int f3 | string f3): decimal cannot numeric-convert to string at all, so R6
    // fails structurally regardless of order -- this case is unambiguous either way.
    anydata v2 = {f2: {f3: d}};
    R r2 = check v2.cloneWithType();
    io:println(r2); // @output {"f2":{"f3":2}}

    // Value only has key f1 (so only R1 is a plausible top-level shape) but f3 is boolean,
    // which fails numeric conversion against both R3 (int) and R4 (float): the nested union
    // fails entirely, and there is no sibling top-level candidate left to back off to
    // (v3 has no f2 key), so the whole conversion is expected to fail.
    anydata v3 = {f1: {f3: true}};
    R|error r3 = v3.cloneWithType();
    if r3 is error {
        io:println("error");
    } else {
        io:println(r3);
    }
    // @output error

    return;
}
