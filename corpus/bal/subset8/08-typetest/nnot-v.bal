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

// @productions is-not-expr is-expr string return-stmt additive-expr any function-call-expr local-var-decl-stmt int-literal
import ballerina/io;

public function main() {
    any x = foo();
    int z = 4;
    io:println(z !is int); // @output false
    io:println(z !is string); // @output true
    io:println(x !is int); // @output false
    io:println(x !is string); // @output true
}

public function foo() returns any {
    int a = 1;
    int b = 2;
    any y = a + b;
    return y;
}
