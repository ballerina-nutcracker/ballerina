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

// @productions is-expr string string-literal boolean if-else-stmt boolean-literal unary-expr additive-expr any function-call-expr int-literal
import ballerina/io;

public function main() {
    foo(2); // @output 3
    foo("hello"); // @output hello, hello
    foo(true); // @output false
}

function foo(any x) {
    if x is int {
        io:println(x + 1);
    }
    else if x is string {
        io:println(x + ", " + x);
    }
    else if x is boolean {
        io:println(!x);
    }
    else {
        io:println(x);
    }
}
