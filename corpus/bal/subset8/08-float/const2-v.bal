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

// @productions module-const-decl multiplicative-expr floating-point-literal unary-expr additive-expr
import ballerina/io;

const A = -3f;
const B = -(3f - 1f);
const C = 3f + 10f / 2f;
const D = 1f + 2.5 + 0.1f + 0.1;

public function main() {
    io:println(A); // @output -3.0
    io:println(B); // @output -2.0
    io:println(C); // @output 8.0
    io:println(D); // @output 3.7

}
