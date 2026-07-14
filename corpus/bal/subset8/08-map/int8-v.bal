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

// @productions map-type-descriptor mapping-constructor-expr type-cast-expr exact-equality equality-expr local-var-decl-stmt int-literal
import ballerina/io;

public function main() {
    map<int> im = {x: 1, y: 2, z: 3};
    any m = im;
    _ = m;
    map<int> im2 = <map<int>>im;
    io:println(im2); // @output {"x":1,"y":2,"z":3}
    io:println(im === im2); // @output true
}
