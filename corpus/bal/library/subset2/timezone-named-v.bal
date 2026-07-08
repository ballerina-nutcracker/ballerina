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
    // Named IANA timezone
    time:TimeZone tz = check new time:TimeZone("Asia/Colombo");

    // Named timezones do not return a fixed offset
    time:ZoneOffset? offset = tz.fixedOffset();
    io:println(offset is ()); // @output true

    // utcToCivil converts UTC to the Asia/Colombo zone (+05:30)
    time:Utc utc = check time:utcFromString("2007-12-03T10:15:30.00Z");
    time:Civil civil = tz.utcToCivil(utc);
    io:println(civil.year);       // @output 2007
    io:println(civil.month);      // @output 12
    io:println(civil.day);        // @output 3
    io:println(civil.hour);       // @output 15
    io:println(civil.minute);     // @output 45
    io:println(civil.timeAbbrev); // @output Asia/Colombo

    // utcFromCivil round-trips back to the original UTC
    time:Utc utc2 = check tz.utcFromCivil(civil);
    io:println(time:utcToString(utc2)); // @output 2007-12-03T10:15:30Z

    // civilAddDuration adds one month
    time:Civil civil2 = check tz.civilAddDuration(civil, {months: 1});
    io:println(civil2.month);      // @output 1
    io:println(civil2.year);       // @output 2008
    io:println(civil2.timeAbbrev); // @output Asia/Colombo
}
