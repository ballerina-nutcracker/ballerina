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
    // utcNow() with no precision -> externUtcNow(-1) -> goTimeToUtc (nanosecond) path
    time:Utc now = time:utcNow();
    io:println(now.length() == 2); // @output true

    // utcFromString with a malformed RFC 3339 string -> FormatError
    time:Utc|error badUtc = time:utcFromString("not-a-timestamp");
    io:println(badUtc is error); // @output true

    // dateValidate with an out-of-range day -> FormatError
    time:Error? badDay = time:dateValidate({year: 2024, month: 2, day: 30});
    io:println(badDay is time:Error); // @output true

    // civilFromString with a malformed string -> FormatError
    time:Civil|error badCivil = time:civilFromString("garbage");
    io:println(badCivil is error); // @output true

    // Standalone utcFromCivil with a fixed utcOffset
    time:Utc fromOffset = check time:utcFromCivil({
        year: 2021,
        month: 4,
        day: 12,
        hour: 23,
        minute: 20,
        second: 50,
        utcOffset: {hours: 5, minutes: 30}
    });
    io:println(time:utcToString(fromOffset)); // @output 2021-04-12T17:50:50Z

    // Standalone utcFromCivil with timeAbbrev "Z" and no utcOffset
    time:Utc fromZ = check time:utcFromCivil({
        year: 2021,
        month: 4,
        day: 12,
        hour: 17,
        minute: 50,
        second: 50,
        timeAbbrev: "Z"
    });
    io:println(time:utcToString(fromZ)); // @output 2021-04-12T17:50:50Z

    // Standalone utcFromCivil with neither utcOffset nor timeAbbrev -> error
    time:Utc|error noZone = time:utcFromCivil({
        year: 2021,
        month: 4,
        day: 12,
        hour: 17,
        minute: 50
    });
    io:println(noZone is error); // @output true

    // Standalone civilAddDuration with a civil missing zone info -> error
    time:Civil|error addErr = time:civilAddDuration(
        {year: 2021, month: 4, day: 12, hour: 10, minute: 0},
        {hours: 1}
    );
    io:println(addErr is error); // @output true
}
