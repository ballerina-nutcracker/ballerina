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

// @productions match-stmt string string-literal list-constructor-expr boolean-literal return-stmt any function-call-expr int-literal
import ballerina/io;

public function main() {
    io:println(foo(0)); // @output zero
    io:println(foo(1)); // @output odd
    io:println(foo("hello")); // @output greeting
    io:println(foo(true)); // @output boolean
    io:println(foo(9)); // @output odd
    io:println(foo("hi")); // @output other
    io:println(foo([0])); // @output other
    io:println(foo(false)); // @output other
}

function foo(any v) returns string {
    match v {
        0 => {
            return "zero";
        }
        1|3|5|7|9 => {
            return "odd";
        }
        true => {
            return "boolean";
        }
        "hello" => {
            return "greeting";
        }
        _ => {
            return "other";
        }
    }
}
