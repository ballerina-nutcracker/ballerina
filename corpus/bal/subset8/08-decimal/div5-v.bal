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
    decimal d1 = 1d;
    decimal d2 = 1d;
    io:println(d1 / d2); // @output 1

    d1 = -1d;
    d2 = 1d;
    io:println(d1 / d2); // @output -1

    d1 = 0d;
    d2 = -1d;
    io:println(d1 / d2); // @output 0

    d1 = 9.999999999999999999999999999999999E6144d;
    d2 = 9.999999999999999999999999999999999E6144d;
    io:println(d1 / d2); // @output 1

    d1 = 9.999999999999999999999999999999999E-6001d;
    d2 = 0.5E143d;
    io:println(d1 / d2); // @output 2.000000000000000000000000000000000E-6143
}
