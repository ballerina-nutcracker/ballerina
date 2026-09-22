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

// Every monomorphic form of `array:indexOf` shares the one untyped signature
// declared in Go, so every omitted `startIndex` here calls the same
// `$default$N` provider in `lang.array` regardless of the container type. The
// BIR golden shows it: all four call sites name `$default$0`, where a provider
// per specialization would have produced `$default$0` through `$default$3`.
public function main() {
    int[] ints = [1, 2, 3, 2];
    string[] strings = ["a", "b", "c", "b"];

    io:println(ints.indexOf(2)); // @output 1
    io:println(ints.indexOf(3)); // @output 2
    io:println(strings.indexOf("b")); // @output 1
    io:println(strings.indexOf("c")); // @output 2
}
