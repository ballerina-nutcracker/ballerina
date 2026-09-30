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

// A nested logical RHS keeps the chain the enclosing composition inherited, so
// both outcomes and the ordinary sequencing after them stay correct.
public function main() {
    int|string a = 1;
    int|string b = 2;
    int|string c = "three";
    if a is int && (b is int || c is int) {
        io:println(a + 1); // @output 2
        if b is int {
            io:println(b + 1); // @output 3
        }
    }
    if !(a is int) || !(b is int && c is int) {
        io:println("dual"); // @output dual
    }
}
