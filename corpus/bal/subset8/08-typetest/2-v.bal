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

// @productions is-expr map-type-descriptor mapping-constructor-expr string string-literal list-type-descriptor list-constructor-expr boolean if-else-stmt boolean-literal unary-expr any function-call-expr local-var-decl-stmt int-literal
import ballerina/io;

public function main() {
    p(1); // @output int
    p(false); // @output boolean
    p("hello"); // @output string
    p("this is a long string"); // @output string
    p(0x7fffffffffffffff); // @output int
    p(-0x7fffffffffffffff); // @output int
    any[] list = [1, 2];
    p(list); // @output array
    map<any> mapping = {};
    p(mapping); // @output map
}

function p(any v) {
    if v is boolean {
        io:println("boolean");
    }
    if v is int {
        io:println("int");
    }
    if v is string {
        io:println("string");
    }
    if v is any[] {
        io:println("array");
    }
    if v is map<any> {
        io:println("map");
    }
}
