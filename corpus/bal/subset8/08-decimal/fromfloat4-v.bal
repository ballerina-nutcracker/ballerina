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
    float f = 0.0f;
    io:println(<decimal>f); // @output 0
    f = 1.0f;
    io:println(<decimal>f); // @output 1
    f = 1.00000f;
    io:println(<decimal>f); // @output 1
    f = 1234567891.0f;
    io:println(<decimal>f); // @output 1234567891
    f = 1234567890123456.0f;
    io:println(<decimal>f); // @output 1234567890123456
    f = 1234567890123456.1f;
    io:println(<decimal>f); // @output 1234567890123456
    f = 1234567890123456.9f;
    io:println(<decimal>f); // @output 1234567890123457
    f = 1.7976931348623157e+308f;
    io:println(<decimal>f); // @output 1.7976931348623157E+308
    f = 0.0000000000000009e+308f;
    io:println(<decimal>f); // @output 9E+292
    f = 0.0000000000000000000000000000000001f;
    io:println(<decimal>f); // @output 1E-34
    f = 4.9406564584124654E-324f;
    io:println(<decimal>f); // @output 5E-324
}
