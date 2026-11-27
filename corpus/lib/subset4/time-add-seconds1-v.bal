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
import ballerina/time;

public function main() {
    io:println(time:utcAddSeconds([0, 0.5], 10000000000.75)); // @output [10000000001,0.25]
    io:println(addSecondsErrorKind([1, 0], 9223372036854775807)); // @output NumberOverflow
    io:println(addSecondsErrorKind([9223372036854775807, 0.5], 0.75)); // @output NumberOverflow
    io:println(addSecondsErrorKind([-9223372036854775807 - 1, 0.5], -0.25)); // @output NumberOverflow
    io:println(addSecondsErrorKind([0, 0], -9223372036854775808.5)); // @output NumberConversionError
    io:println(addSecondsErrorKind([-9223372036854775807 - 1, 0], -0.01)); // @output NumberOverflow
    io:println(time:utcAddSeconds([10, 0], -0.01)); // @output [9,0.99]
    io:println(time:utcAddSeconds([9223372036854775806, 0.5], 0.75)); // @output [9223372036854775807,0.25]
}

function addSecondsErrorKind(time:Utc utc, decimal seconds) returns string {
    time:Utc|error result = trap time:utcAddSeconds(utc, seconds);
    if result is time:Utc {
        return "no error";
    }
    string message = result.message();
    return message.includes("NumberOverflow") ? "NumberOverflow" : message.includes("NumberConversionError") ? "NumberConversionError" : message;
}
