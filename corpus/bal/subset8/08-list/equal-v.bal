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

// @productions list-type-descriptor list-constructor-expr exact-equality equality equality-expr local-var-decl-stmt int-literal
import ballerina/io;

public function main() {
    int[] v1 = [1, 2, 3];
    int[] v2 = [1, 2, 3];
    int[] v3 = [1, 2, 4];
    int[] v4 = [1, 2, 3, 4];

    io:println(v1 == v1); // @output true
    io:println(v1 == v2); // @output true
    io:println(v1 == v3); // @output false
    io:println(v1 == v4); // @output false
    io:println(v4 == v1); // @output false
    io:println(v1 === v1); // @output true
    io:println(v1 === v2); // @output false
}
