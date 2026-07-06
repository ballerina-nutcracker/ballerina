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

public function main() returns error? {
    // RFC 9557: IANA zone annotation "[Zone/Name]" after the fixed offset
    time:Civil c = check time:civilFromString("2021-04-12T23:20:50.520+05:30[Asia/Colombo]");
    io:println(c.year);       // @output 2021
    io:println(c.month);      // @output 4
    io:println(c.day);        // @output 12
    io:println(c.hour);       // @output 23
    io:println(c.minute);     // @output 20
    io:println(c.timeAbbrev); // @output Asia/Colombo
}
