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
    // 2.675 is stored as 2.67499999999999982236431605997495353221893310546875
    io:println((2.675).round(2)); // @output 2.67
    io:println((-2.675).round(2)); // @output -2.67
    // 0.15 is stored as 0.1499999999999999944488848768742172978818416595458984375
    io:println((0.15).round(1)); // @output 0.1
    // 0.35 is stored as 0.34999999999999997779553950749686919152736663818359375
    io:println((0.35).round(1)); // @output 0.3
    // 0.45 is stored as 0.450000000000000011102230246251565404236316680908203125
    io:println((0.45).round(1)); // @output 0.5
    io:println((-0.45).round(1)); // @output -0.5
    // 9.995 is stored as 9.9949999999999992184029906638897955417633056640625
    io:println((9.995).round(2)); // @output 9.99
    io:println((9.96).round(1)); // @output 10.0
    // 1.005 is stored as 1.00499999999999989341858963598497211933135986328125
    io:println((1.005).round(2)); // @output 1.0
    // 0.125 and 0.375 are exact ties, so the even final digit is chosen
    io:println((0.125).round(2)); // @output 0.12
    io:println((0.375).round(2)); // @output 0.38
    io:println((-0.125).round(2)); // @output -0.12
    io:println((1250.0).round(-2)); // @output 1200.0
    io:println((1350.0).round(-2)); // @output 1400.0
    io:println((-1350.0).round(-2)); // @output -1400.0
    io:println((6.0e307).round(-308)); // @output 1e308
    io:println((4e307).round(-308)); // @output 0.0
    io:println((1.7e308).round(-308)); // @output Infinity
    io:println((-1.7e308).round(-308)); // @output -Infinity
    // 5e-324 is stored as 4.94065645841246544...e-324
    io:println((5e-324).round(323)); // @output 0.0
    io:println((5e-324).round(324)); // @output 5e-324
    io:println((5e-324).round(1073)); // @output 5e-324
}
