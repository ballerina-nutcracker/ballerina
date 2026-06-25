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
import ballerina/lang.'float as floats;

public function main() {
    io:println(floats:sum()); // @output 0.0
    io:println(floats:sum(1.5, 2.25, -0.75)); // @output 3.0
    io:println(floats:max(1.5, 2.25, -0.75)); // @output 2.25
    io:println(floats:min(1.5, 2.25, -0.75)); // @output -0.75
    io:println((-1.25).abs()); // @output 1.25
    io:println((2.5).round()); // @output 2.0
    io:println((3.5).round()); // @output 4.0
    io:println((4.55555).round(3)); // @output 4.556
    io:println((1.2345).round(400)); // @output 1.2345
    io:println((1.2345).round(-400)); // @output 0.0
    io:println((-1.2345).round(-309)); // @output 0.0
    io:println((-1.2).floor()); // @output -2.0
    io:println((-1.2).ceiling()); // @output -1.0
    io:println((4.0).sqrt()); // @output 2.0
    io:println((0.125).cbrt()); // @output 0.5
    io:println((2.0).pow(3.0)); // @output 8.0
    io:println((1.0).log()); // @output 0.0
    io:println((10.0).log10()); // @output 1.0
    io:println((0.0).exp()); // @output 1.0
    io:println((0.0).sin()); // @output 0.0
    io:println((0.0).cos()); // @output 1.0
    io:println((0.0).tan()); // @output 0.0
    io:println((1.0).acos()); // @output 0.0
    io:println((0.0).atan()); // @output 0.0
    io:println((0.0).asin()); // @output 0.0
    io:println(floats:atan2(0.0, 1.0)); // @output 0.0
    io:println((0.0).sinh()); // @output 0.0
    io:println((0.0).cosh()); // @output 1.0
    io:println((0.0).tanh()); // @output 0.0
    io:println(floats:isFinite(1.0)); // @output true
    io:println(floats:isInfinite(floats:Infinity)); // @output true
    io:println(floats:isNaN(floats:NaN)); // @output true

    float|error parsed = floats:fromString("+12.5");
    if parsed is float {
        io:println(parsed); // @output 12.5
    }
    io:println(floats:fromString("bad") is error); // @output true
    io:println((-10.2453).toHexString()); // @output -0x1.47d97f62b6ae8p3
    io:println(floats:Infinity.toHexString()); // @output Infinity
    io:println(floats:fromHexString("0x1.0a3d70a3d70a4p4")); // @output 16.64
    io:println(floats:fromHexString("0x1J") is error); // @output true
    io:println((4.16).toBitsInt()); // @output 4616369762039853220
    io:println(floats:fromBitsInt(4)); // @output 2e-323
    io:println((12.456).toFixedString(2)); // @output 12.46
    io:println((12.456).toFixedString(())); // @output 12.456
    io:println((12.456).toExpString(2)); // @output 1.25e+01
    io:println((12.456).toExpString(())); // @output 1.2456e+01
    io:println(floats:avg(2.0, 4.0)); // @output 3.0
    io:println(floats:avg().isNaN()); // @output true
}
