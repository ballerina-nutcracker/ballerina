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
    checkpanic run();
}

function run() returns error? {
    json floatNum = 2.5;
    int rounded = check floatNum.fromJsonWithType(int);
    io:println(rounded); // @output 2

    json intNum = 42;
    float asFloat = check intNum.fromJsonWithType(float);
    io:println(asFloat); // @output 42.0

    json decNum = 7;
    decimal asDecimal = check decNum.fromJsonWithType(decimal);
    io:println(asDecimal); // @output 7

    json maxByte = 255;
    byte b = check maxByte.fromJsonWithType(byte);
    io:println(b); // @output 255

    json floatByte = 200.0;
    byte fromFloat = check floatByte.fromJsonWithType(byte);
    io:println(fromFloat); // @output 200

    json outOfRange = 256;
    io:println(outOfRange.fromJsonWithType(byte) is error); // @output true

    json roundedOutOfRange = 255.6;
    io:println(roundedOutOfRange.fromJsonWithType(byte) is error); // @output true

    json negByte = -1;
    io:println(negByte.fromJsonWithType(byte) is error); // @output true

    float nan = 0.0/0.0;
    json nanJson = nan;
    io:println(nanJson.fromJsonWithType(int) is error); // @output true

    float inf = 1.0/0.0;
    json infJson = inf;
    io:println(infJson.fromJsonWithType(int) is error); // @output true

    json hugeFloat = 1e100;
    io:println(hugeFloat.fromJsonWithType(int) is error); // @output true

    float maxIntOverflow = 9223372036854775808.0;
    json maxIntOverflowJson = maxIntOverflow;
    io:println(maxIntOverflowJson.fromJsonWithType(int) is error); // @output true

    decimal decVal = 42;
    json decJson = decVal;
    int fromDec = check decJson.fromJsonWithType(int);
    io:println(fromDec); // @output 42

    decimal tooBig = 9223372036854775808;
    json tooBigJson = tooBig;
    io:println(tooBigJson.fromJsonWithType(int) is error); // @output true

    json floatToDec = 1.5;
    decimal decFromFloat = check floatToDec.fromJsonWithType(decimal);
    io:println(decFromFloat); // @output 1.5

    io:println(nanJson.fromJsonWithType(decimal) is error); // @output true
    io:println(infJson.fromJsonWithType(decimal) is error); // @output true

    io:println(nanJson.fromJsonWithType(byte) is error); // @output true
    io:println(infJson.fromJsonWithType(byte) is error); // @output true

    decimal decForFloat = 3.5;
    json decForFloatJson = decForFloat;
    float floatFromDec = check decForFloatJson.fromJsonWithType(float);
    io:println(floatFromDec); // @output 3.5
    return;
}
