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

// @productions error-constructor-expr is-expr string-literal if-else-stmt relational-expr return-stmt unary-expr additive-expr function-call-expr assign-stmt local-var-decl-stmt int-literal
import ballerina/io;

public function main() {
    int|error result = positive(-1);
    display(result); // @output error("negative")
    result = positive(4);
    display(result); // @output 8
}

function display(int|error result) {
    if result is int {
        io:println(result + result);
    }
    else {
        error e = result;
        io:println(e);
    }
}

function positive(int n) returns int|error {
    if n >= 0 {
        return n;
    }
    else {
        return error("negative");
    }
}
