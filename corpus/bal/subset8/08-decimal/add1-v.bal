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
    io:println(1d + 1d); // @output 2
    io:println(1000d + 1d); // @output 1001
    io:println(1234567890123456789012345678901234d + 1234567890123456789012345678901231d); // @output 2469135780246913578024691357802465
    io:println(12345678901234567890123456789012341d + 12345678901234567890123456789012312d); // @output 2.469135780246913578024691357802465E+34
    io:println(1234567890123456789012345678901234d + 12345678901234567890123456789012312d); // @output 1.358024679135802467913580246791354E+34
    io:println(9.999999999999999999999999999999998E6144d + 0.000000000000000000000000000000001E6144d); // @output 9.999999999999999999999999999999999E+6144
    io:println(9.999999999999999999999999999999995E6144d + 0.000000000000000000000000000000002E6144d); // @output 9.999999999999999999999999999999997E+6144
    io:println(-9.999999999999999999999999999999998E6144d + -0.000000000000000000000000000000001E6144d); // @output -9.999999999999999999999999999999999E+6144
    io:println(-9.999999999999999999999999999999999E6144d + 0.000000000000000000000000000000001E6144d); // @output -9.999999999999999999999999999999998E+6144
    io:println(2E-6143d + 1E-6143d); // @output 3E-6143
    io:println(0.000000000000000000000000000000001E-6110d + 0.000000000000000000000000000000002E-6110d); // @output 3E-6143
    io:println(2E-6143d + -1E-6143d); // @output 1E-6143
    io:println(9E-6143d + 1E-6143d); // @output 1.0E-6142
    io:println(1E-6143d + -1E-6143d); // @output 0
}
