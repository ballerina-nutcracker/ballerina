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
    // UTC timezone via "UTC" zone ID
    time:TimeZone utcZone = check new time:TimeZone("UTC");

    // fixedOffset returns {hours: 0} for UTC
    time:ZoneOffset? offset = utcZone.fixedOffset();
    if !(offset is ()) {
        io:println(offset.hours);   // @output 0
        io:println(offset.minutes); // @output 0
    }

    // utcToCivil round-trips through a known UTC timestamp
    time:Utc utc = check time:utcFromString("2007-12-03T10:15:30.00Z");
    time:Civil civil = utcZone.utcToCivil(utc);
    io:println(civil.year);   // @output 2007
    io:println(civil.month);  // @output 12
    io:println(civil.day);    // @output 3
    io:println(civil.hour);   // @output 10
    io:println(civil.minute); // @output 15

    // utcFromCivil requires timeAbbrev
    time:Utc utc2 = check utcZone.utcFromCivil(civil);
    io:println(time:utcToString(utc2)); // @output 2007-12-03T10:15:30Z

    // civilAddDuration adds one day
    time:Civil civil2 = check utcZone.civilAddDuration(civil, {days: 1});
    io:println(civil2.day); // @output 4
}
