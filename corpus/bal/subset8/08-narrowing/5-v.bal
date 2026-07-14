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

// @productions float is-expr if-else-stmt floating-point-literal return-stmt any function-call-expr int-literal
import ballerina/io;

public function main() {
    io:println(foo(1.5)); // @output 1.5
    io:println(foo(0.5)); // @output 0.5
    io:println(foo(17)); // @output 2.5
}

function foo(any v) returns float {
    if v is float {
        return v;
    }
    else {
        return 2.5;
    }
}
