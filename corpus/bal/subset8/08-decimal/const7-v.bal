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

const decimal D1 = 1.1e2d;
const D2 = 1.1e2d;
const D3 = 9.9e23d;
const D4 = D2 + D3;
const decimal D5 = D2 + D3;
const D6 = D2 - D3;
const decimal D7 = D2 - D3;
const D8 = D2 * D3;
const D9 = D3 / D2;
const D10 = D3 % D2;

public function main() {
    io:println(D1); // @output 1.1E+2
    io:println(D2); // @output 1.1E+2
    io:println(D3); // @output 9.9E+23
    io:println(D4); // @output 9.9000000000000000000011E+23
    io:println(D5); // @output 9.9000000000000000000011E+23
    io:println(D6); // @output -9.8999999999999999999989E+23
    io:println(D7); // @output -9.8999999999999999999989E+23
    io:println(D8); // @output 1.089E+26
    io:println(D9); // @output 9E+21
    io:println(D10); // @output 0
}
