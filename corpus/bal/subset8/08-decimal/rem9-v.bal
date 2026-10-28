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

const decimal C = 9.0E32d % 1E-2d;

public function main() {
    decimal d1 = 1E+35d;
    decimal d2 = 7d;
    io:println(d1 % d2); // @output 5

    d1 = 1E+40d;
    d2 = 3d;
    io:println(d1 % d2); // @output 1

    d1 = -1E+40d;
    d2 = 3d;
    io:println(d1 % d2); // @output -1

    d1 = 1E+40d;
    d2 = -3d;
    io:println(d1 % d2); // @output 1

    d1 = -1E+40d;
    d2 = 2d;
    io:println(d1 % d2); // @output 0

    d1 = 1E+40d;
    d2 = 1.1d;
    io:println(d1 % d2); // @output 1.0

    d1 = 9.0E32d;
    d2 = 1E-2d;
    io:println(d1 % d2); // @output 0

    d1 = 9.999999999999999999999999999999998E6144d;
    d2 = 1E-2d;
    io:println(d1 % d2); // @output 0

    d1 = 9.999999999999999999999999999999999E6144d;
    d2 = 1.23E-6100d;
    io:println(d1 % d2); // @output 8.4E-6101

    io:println(9.0E32d % 1E-2d); // @output 0
    io:println(1E+6144d % 0.7d); // @output 0.3
    io:println(C); // @output 0
}
