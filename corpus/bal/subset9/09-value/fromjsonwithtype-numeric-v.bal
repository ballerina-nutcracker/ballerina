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

// Numeric edge cases specific to fromJsonWithType's json-number source values.
// Ordinary int/float/decimal/byte conversions (including NaN/Infinity to int,
// out-of-range/negative byte, and round-to-even) are already covered via
// cloneWithType in clonewithtype-numeric-v.bal, which exercises the same
// shared conversion code; this file only adds cases not covered there.

import ballerina/io;

public function main() returns error? {
    // byte upper boundary
    json maxByte = 255;
    byte b = check maxByte.fromJsonWithType(byte);
    io:println(b); // @output 255

    // rounding pushes an in-range float just over the byte boundary
    json roundedOutOfRange = 255.6;
    io:println(roundedOutOfRange.fromJsonWithType(byte) is error); // @output true

    // float far outside int64 range (not NaN/Inf)
    json hugeFloat = 1e100;
    io:println(hugeFloat.fromJsonWithType(int) is error); // @output true

    // float just above int64 max
    float maxIntOverflow = 9223372036854775808.0;
    json maxIntOverflowJson = maxIntOverflow;
    io:println(maxIntOverflowJson.fromJsonWithType(int) is error); // @output true

    // decimal just above int64 max
    decimal tooBig = 9223372036854775808;
    json tooBigJson = tooBig;
    io:println(tooBigJson.fromJsonWithType(int) is error); // @output true

    float nan = 0.0/0.0;
    json nanJson = nan;
    float inf = 1.0/0.0;
    json infJson = inf;

    io:println(nanJson.fromJsonWithType(decimal) is error); // @output true
    io:println(infJson.fromJsonWithType(decimal) is error); // @output true
    io:println(nanJson.fromJsonWithType(byte) is error); // @output true
    io:println(infJson.fromJsonWithType(byte) is error); // @output true

    return;
}
