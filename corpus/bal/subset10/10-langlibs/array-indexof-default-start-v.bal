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

// `array:indexOf` is declared as `indexOf(arr, val, int startIndex = 0)`, so
// omitting startIndex is valid Ballerina. It is skiplisted (see
// test_util/skip.go): a default parameter is evaluated by a default closure
// generated from the defining module's AST, and an opaque lang-library
// function is defined in Go and has no such module. This test documents the
// target behaviour until opaque functions can carry defaults.
public function main() {
    int[] arr = [10, 20, 30, 40, 30];
    int? firstDefault = arr.indexOf(30);
    if firstDefault is int {
        io:println(firstDefault); // @output 2
    }
}
