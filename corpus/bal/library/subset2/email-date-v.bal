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
    // utcToEmailString with UTC (covers formatEmailDate, emailOffsetStr "0" -> "+0000")
    time:Utc utc = check time:utcFromString("2024-06-15T10:30:45.00Z");
    string emailStr = time:utcToEmailString(utc);
    io:println(emailStr); // @output Sat, 15 Jun 2024 10:30:45 +0000

    // utcToEmailString with explicit "GMT" handling
    string emailGmt = time:utcToEmailString(utc, "GMT");
    io:println(emailGmt); // @output Sat, 15 Jun 2024 10:30:45 GMT

    // civilFromEmailString with two-digit day (exercises parseEmailDate, normaliseEmailDay)
    time:Civil|error parsed = time:civilFromEmailString("Sat, 15 Jun 2024 10:30:45 +0000");
    io:println(parsed is error); // @output false

    // civilFromEmailString with single-digit day (exercises normaliseEmailDay padding)
    time:Civil|error parsed2 = time:civilFromEmailString("Mon, 3 Jun 2024 08:00:00 +0000");
    io:println(parsed2 is error); // @output false

    // civilFromEmailString error case (covers error path in parseEmailDate)
    time:Civil|error bad = time:civilFromEmailString("not a date");
    io:println(bad is error); // @output true

    // civilToEmailString (covers civilToGoTime, formatEmailDate with zone offset)
    time:TimeZone tz = check new time:TimeZone("+00:00");
    time:Utc utcBase = check time:utcFromString("2024-06-15T10:30:45.00Z");
    time:Civil civil = tz.utcToCivil(utcBase);
    string|error emailCivil = time:civilToEmailString(civil, time:PREFER_ZONE_OFFSET);
    io:println(emailCivil is string); // @output true
    if emailCivil is string {
        io:println(emailCivil); // @output Sat, 15 Jun 2024 10:30:45 +0000
    }
}
