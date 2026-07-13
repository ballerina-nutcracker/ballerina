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

import testorg/crossmoduleatomiccallargs.types;

public function main() {
    types:DefaultInit defaultInit = new ();
    io:println(defaultInit.value); // @output 10

    types:NamedInit namedInit = new (right = 20, left = 10);
    io:println(namedInit.value); // @output 30

    types:IncludedInit includedInit = new (right = 20);
    io:println(includedInit.value); // @output 25

    types:CalculatorImpl classValue = new ();
    io:println(classValue.withDefaults()); // @output 15
    io:println(classValue.withNamed(right = 20, left = 10)); // @output 30
    io:println(classValue.withIncluded(1, right = 20)); // @output 26
}
