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
    float value = 0x10000000000000000;
    io:println(value == 18446744073709551616.0); // @output true
    float negative = -0x10000000000000000;
    io:println(negative == -18446744073709551616.0); // @output true

    int maximum = 0x7fffffffffffffff;
    int minimum = -0x8000000000000000;
    io:println(maximum == 9223372036854775807 && minimum == -9223372036854775808); // @output true
    float aboveMaximum = 0x8000000000000000;
    float belowMinimum = -0x8000000000000001;
    io:println(aboveMaximum == 9223372036854775808.0 && belowMinimum == -9223372036854775808.0); // @output true

    float tie = 0x10000000000000800;
    float aboveTie = 0x10000000000000801;
    io:println(tie == value && aboveTie == 18446744073709555712.0); // @output true
}
