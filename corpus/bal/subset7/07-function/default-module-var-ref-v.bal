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

// A module-level variable is usable in a default value expression as long as
// its type comes from the surrounding context rather than having to be
// inferred. A bare reference takes the parameter's declared type, so it works.
// Passing the same variable to a call does not; see
// default-module-var-ref-arg-v.bal and
// https://github.com/ballerina-nutcracker/ballerina/issues/915.
final int base = 5;

isolated function twice(int x) returns int => x * 2;

// Bare module-variable reference as a default.
function bare(int n = base) returns int => n;

// A module-level initializer is not a default expression, so the same call
// that fails inside one is fine here.
final int doubled = twice(base);

// Defaults that read a preceding parameter, with no module variable involved.
function fromPreceding(int x, int y = x + 1) returns int => x + y;

public function main() {
    io:println(bare()); // @output 5
    io:println(bare(9)); // @output 9
    io:println(doubled); // @output 10
    io:println(fromPreceding(1)); // @output 3
    io:println(fromPreceding(1, 5)); // @output 6
}
