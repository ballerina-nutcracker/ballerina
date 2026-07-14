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
    io:println(1d % 1d); // @output 0
    io:println(1d % 2d); // @output 1
    io:println(-1d % 2d); // @output -1
    io:println(-10d % 3d); // @output -1
    io:println(10d % -3d); // @output 1
    io:println(-16.3d % 4.1d); // @output -4.0
    io:println(17.8d % -4.1d); // @output 1.4
    io:println(-17.8d % -4.1d); // @output -1.4
    io:println(27d % 4d); // @output 3
    io:println(28d % 5d); // @output 3
    io:println(10d % 3d); // @output 1
    io:println(1E-6143d % 1E-6143d); // @output 0
    io:println(1E-6143d % 1E-6142d); // @output 1E-6143
    io:println(1E-6143d % -1E-6142d); // @output 1E-6143
    io:println(1E-3d % 1E-2d); // @output 0.001
    io:println(9.05d % 0.1d); // @output 0.05
    io:println(9.0E31d % 1E-2d); // @output 0
    io:println(9.00005d % 0.0001d); // @output 0.00005
}
