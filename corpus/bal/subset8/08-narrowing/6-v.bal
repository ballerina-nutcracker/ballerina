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

// @productions is-expr string-literal type-cast-expr if-else-stmt boolean-literal return-stmt unary-expr additive-expr any function-call-expr assign-stmt local-var-decl-stmt int-literal
import ballerina/io;

public function main() {
    io:println(add(1, 2)); // @output 3
    io:println(add(2, false)); // @output 2
    io:println(add("hello", false)); // @output -1
}

function add(any x, any y) returns int {
    any n = x;
    if n is int {
        if y is int {
            n = n + y;
            return <int>n;
        }
        return n;
    }
    return -1;
}
