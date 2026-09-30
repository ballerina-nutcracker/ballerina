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

// A record field default is an isolated closure, so it cannot capture a mutable
// local at all. Every default that references one is rejected, even when an
// earlier default has already captured the same variable. See default-final-v
// for the final variant, where both defaults keep the enclosing refinement.
public function main() {
    int|string x = 1;
    if x is int {
        record {| int a = x; int b = x; |} r = {}; // @error both defaults capture mutable x
        io:println(r.a + r.b);
    }
}
