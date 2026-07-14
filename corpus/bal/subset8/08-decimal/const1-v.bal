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
    io:println(1e+123d); // @output 1E+123
    io:println(1E123d); // @output 1E+123
    io:println(1E-123d); // @output 1E-123
    io:println(1E-123D); // @output 1E-123
    io:println(1.2e12d); // @output 1.2E+12
    io:println(.2e12d); // @output 2E+11
    io:println(.2d); // @output 0.2
    io:println(1d); // @output 1

    io:println(1e-6143d); // @output 1E-6143
    io:println(-1e-6143d); // @output -1E-6143
    io:println(9.999999999999999999999999999999999E6144d); // @output 9.999999999999999999999999999999999E+6144
    io:println(-9.999999999999999999999999999999999E6144d); // @output -9.999999999999999999999999999999999E+6144
}
