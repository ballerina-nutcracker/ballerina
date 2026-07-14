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

// @productions type-cast-expr equality multiplicative-expr equality-expr return-stmt unary-expr additive-expr any function-call-expr local-var-decl-stmt int-literal
import ballerina/io;

public function main() {
    int two48 = 65536 * 65536 * 65536;
    testAround(two48);
    // @output true
    // @output true
    // @output true
    // @output true
    int two55 = two48 * 128;
    testAround(two55);
    // @output true
    // @output true
    // @output true
    // @output true
    int two56 = two55 * 2;
    testAround(two56);
    // @output true
    // @output true
    // @output true
    // @output true
    int two62 = two56 * 64;
    testAround(two62);
    // @output true
    // @output true
    // @output true
    // @output true
}

function testAround(int pow2) {
    roundTrip(pow2 - 1);
    roundTrip(-pow2);
    roundTrip(pow2);
    roundTrip(-pow2 - 1);
}

function roundTrip(int n) {
    io:println(n == <int>toAny(n));
}

function toAny(int n) returns any {
    return n;
}
