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
import ballerina/lang.'decimal as decimals;

public function main() {
    io:println(decimals:sum()); // @output 0
    io:println(decimals:sum(1.1d, 2.2d, -0.3d)); // @output 3.0
    io:println(decimals:max(1.1d, 2.2d, -3.0d)); // @output 2.2
    io:println(decimals:min(1.1d, 2.2d, -3.0d)); // @output -3.0
    io:println((-1.23d).abs()); // @output 1.23
    io:println(1.235d.round(2)); // @output 1.24
    io:println(1.234d.quantize(0.00d)); // @output 1.23
    io:println((-1.2d).floor()); // @output -2
    io:println((-1.2d).ceiling()); // @output -1

    decimal|error parsed = decimals:fromString("+12.30");
    if parsed is decimal {
        io:println(parsed); // @output 12.30
    }
    io:println(decimals:fromString("bad") is error); // @output true
}
