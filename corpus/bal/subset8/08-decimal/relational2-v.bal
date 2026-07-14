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

import ballerina/io;

const decimal x = 1.5d;
const decimal y = 3d;
const z = 3d;

public function main() {
    io:println(x <= y); // @output true
    io:println(x >= y); // @output false
    io:println(y < x); // @output false
    io:println(y > x); // @output true
    io:println(y >= z); // @output true
    io:println(y <= z); // @output true
}
