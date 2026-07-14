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

const D1 = 2.1d;
const D2 = -D1;

const decimal D3 = 2.1e2d;
const decimal D4 = -D3;

public function main() {
    decimal d = 0d;
    io:println(-d); // @output 0

    d = 1d;
    io:println(-d); // @output -1

    d = -1d;
    io:println(-d); // @output 1

    d = 1E-6143d;
    io:println(-d); // @output -1E-6143

    d = 9.999999999999999999999999999999998E6144d;
    io:println(-d); // @output -9.999999999999999999999999999999998E+6144

    io:println(D2); // @output -2.1
    io:println(D4); // @output -2.1E+2
}
