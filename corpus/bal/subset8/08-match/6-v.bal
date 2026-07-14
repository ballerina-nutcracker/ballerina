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

// @productions is-expr match-stmt string-literal if-else-stmt return-stmt unary-expr any function-call-expr int-literal
import ballerina/io;

public function main() {
    io:println(foo("hello")); // @output 0
    io:println(foo(1)); // @output 1
    io:println(foo(2)); // @output 2
    io:println(foo(3)); // @output 3
    io:println(foo(-4)); // @output 4
}

function foo(any v) returns int {
    if v is int {
        match v {
            1|2|3 => {
                return v;
            }
            _ => {
                return -v;
            }
        }
    }
    return 0;
}
