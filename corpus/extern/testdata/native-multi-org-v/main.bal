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

// Integration test: native packages from two different orgs alongside pure
// Ballerina packages. Mirrors samples/nativeTest.

import acmeorg/calcpkg;
import ballerina/io;
import mockorg/greetpkg;
import mockorg/middlepkg;
import mockorg/nativepkg;

public function main() {
    // Native package 1 (mockorg): string extern
    io:println(nativepkg:hello());

    // Native package 2 (acmeorg): arithmetic externs
    io:println(calcpkg:multiply(6, 7));
    io:println(calcpkg:abs(-42));

    // Pure Ballerina (v4 format)
    io:println(greetpkg:greet("Ballerina"));

    // Pure Ballerina with transitive deps
    io:println(middlepkg:getDoubledValue());
    io:println(middlepkg:quadrupleValue());
}
