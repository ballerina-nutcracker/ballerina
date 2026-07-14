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

public function main() {
    io:println(1d * 1d); // @output 1
    io:println(-1d * 1d); // @output -1
    io:println(-1d * 0d); // @output 0
    io:println(1E-6000d * 1E-143d); // @output 1E-6143
    io:println(1E6000d * 1E144d); // @output 1.000000000000000000000000000000000E+6144
    io:println(9.999999999999999999999999999999999E6000d * 1E144d); // @output 9.999999999999999999999999999999999E+6144
    io:println(9.999999999999999999999999999999999E6000d * -1E144d); // @output -9.999999999999999999999999999999999E+6144
    io:println(9.999999999999999999999999999999999E6000d * -2E143d); // @output -2.000000000000000000000000000000000E+6144
    io:println(-1E-6143d * 0d); // @output 0
    io:println(1E-6143d * 0d); // @output 0
    io:println(0d * 0d); // @output 0
    io:println(9.999999999999999999999999999999999E-6001d * 2E-143d); // @output 2.000000000000000000000000000000000E-6143
    io:println(1E-6143d * 1E6143d); // @output 1
}
