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
    int|error n = int:fromString("12345");
    io:println(n); // @output 12345

    int|error neg = int:fromString("-42");
    io:println(neg); // @output -42

    int|error invalid = int:fromString("not-a-number");
    io:println(invalid is error); // @output true

    int|error h = int:fromHexString("ff");
    io:println(h); // @output 255

    int|error negH = int:fromHexString("-1a");
    io:println(negH); // @output -26

    int|error invalidHex = int:fromHexString("zz");
    io:println(invalidHex is error); // @output true

    int|error plusH = int:fromHexString("+1a");
    io:println(plusH); // @output 26

    int|error minH = int:fromHexString("-8000000000000000");
    io:println(minH); // @output -9223372036854775808

    int|error maxH = int:fromHexString("7fffffffffffffff");
    io:println(maxH); // @output 9223372036854775807

    int|error overflowH = int:fromHexString("ffffffffffffffff");
    io:println(overflowH is error); // @output true
}
