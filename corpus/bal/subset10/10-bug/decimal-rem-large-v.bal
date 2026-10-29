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

const decimal C1 = 1E+40d % 3d;
const decimal C2 = -1E+40d % 1.1d;

public function main() {
    decimal d1 = 1E+35d;
    decimal d2 = 7d;
    io:println(d1 % d2); // @output 5

    d1 = -1E+40d;
    d2 = 3d;
    io:println(d1 % d2); // @output -1

    d1 = 1E+40d;
    d2 = -3d;
    io:println(d1 % d2); // @output 1

    d1 = -1E+40d;
    d2 = -3d;
    io:println(d1 % d2); // @output -1

    d1 = -1E+40d;
    d2 = 2d;
    io:println(d1 % d2); // @output 0

    d1 = 1E+40d;
    d2 = 1.10d;
    io:println(d1 % d2); // @output 1.00

    d1 = 1E+39d;
    d2 = 3E+2d;
    io:println(d1 % d2); // @output 1E+2

    d1 = 9.0E32d;
    d2 = 1E-2d;
    io:println(d1 % d2); // @output 0

    d1 = 1E+6144d;
    d2 = 9999999999999999999999999999999999d;
    io:println(d1 % d2); // @output 1000000000000000000000000

    d2 = 7.000000000000000000000000000000000E-6143d;
    io:println(d1 % d2); // @output 5.000000000000000000000000000000000E-6143

    d1 = 9.999999999999999999999999999999999E6144d;
    d2 = 1.23E-6100d;
    io:println(d1 % d2); // @output 8.4E-6101

    d1 = 1E+34d;
    d2 = 7d;
    io:println(d1 % d2); // @output 4

    io:println(C1); // @output 1
    io:println(C2); // @output -1.0
}
