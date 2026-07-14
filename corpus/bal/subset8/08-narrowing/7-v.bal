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

// @productions float string-literal equality multiplicative-expr if-else-stmt equality-expr floating-point-literal unary-expr function-call-expr
import ballerina/io;

public function main() {
    foo(0.0); // @output positive zero
    foo(-0.0); // @output negative zero
    foo(1.0); // @output non-zero
}

function foo(float f) {
    if f == 0.0 {
        if 1.0 / f == 2.0 / 0.0 {
            io:println("positive zero");
        }
        else if 3.0 / f == 4.0 / -0.0 {
            io:println("negative zero");
        }
    }
    else {
        io:println("non-zero");
    }
}
