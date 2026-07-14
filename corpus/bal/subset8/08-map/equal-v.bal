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

// @productions map-type-descriptor mapping-constructor-expr exact-equality equality equality-expr local-var-decl-stmt int-literal
import ballerina/io;

public function main() {
    map<int> v1 = {x: 1, y: 2, z: 3};
    map<int> v2 = {z: 3, y: 2, x: 1};
    map<int> v3 = {x: 1, y: 2, z: 4};
    map<int> v4 = {x: 1, y: 2, z: 3, w: 4};

    io:println(v1 == v1); // @output true
    io:println(v1 == v2); // @output true
    io:println(v1 == v3); // @output false
    io:println(v1 == v4); // @output false
    io:println(v4 == v1); // @output false
    io:println(v1 === v1); // @output true
    io:println(v1 === v2); // @output false
}
