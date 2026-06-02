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

    json outOfRange = 256;
    io:println(outOfRange.fromJsonWithType(byte) is error); // @output true

    json roundedOutOfRange = 255.6;
    io:println(roundedOutOfRange.fromJsonWithType(byte) is error); // @output true
    return;
}
