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

// @productions mapping-constructor-expr equality equality-expr boolean-literal assign-stmt local-var-decl-stmt int-literal
import ballerina/io;

public function main() {
    any x = {v: true};
    io:println(x == 1); // @output false
    io:println(x == true); // @output false
    io:println(x == ()); // @output false
    x = {};
    io:println(x != 1); // @output true
    x = {x: 1, y: 2, z: 3};
    io:println(x != true); // @output true
    io:println(x != ()); // @output true
}
