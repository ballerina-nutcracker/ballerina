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
import ballerina/lang.'decimal as decimal;

public function main() {
    io:println(decimal:fromString("1.") is error); // @output true
    io:println(decimal:fromString("1.e5") is error); // @output true
    io:println(decimal:fromString("00.5") is error); // @output true
    io:println(decimal:fromString("00") is error); // @output true
    io:println(decimal:fromString("-01.5") is error); // @output true
    io:println(decimal:fromString("1e+") is error); // @output true
    io:println(decimal:fromString("NaN") is error); // @output true
    io:println(decimal:fromString("Infinity") is error); // @output true
    io:println(decimal:fromString("1_000") is error); // @output true
    io:println(decimal:fromString(" 1") is error); // @output true

    io:println(decimal:fromString("0")); // @output 0
    io:println(decimal:fromString("+12.30")); // @output 12.30
    io:println(decimal:fromString(".5")); // @output 0.5
    io:println(decimal:fromString("-.5")); // @output -0.5
    io:println(decimal:fromString("1e005")); // @output 1E+5
    io:println(decimal:fromString("1.2E-3")); // @output 0.0012
}
