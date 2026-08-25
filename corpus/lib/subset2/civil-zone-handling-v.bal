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
    // civilToString with a named IANA zone in timeAbbrev (no utcOffset)
    // -> civilNamedZoneTime LoadLocation path
    string named = check time:civilToString({
        year: 2021,
        month: 4,
        day: 12,
        hour: 23,
        minute: 20,
        second: 50,
        timeAbbrev: "Asia/Colombo"
    });
    io:println(named); // @output 2021-04-12T23:20:50+05:30

    // civilToString with a numeric offset in timeAbbrev (no utcOffset)
    // -> civilNamedZoneTime isNumericOffset path
    string numeric = check time:civilToString({
        year: 2021,
        month: 4,
        day: 12,
        hour: 23,
        minute: 20,
        second: 50,
        timeAbbrev: "+05:30"
    });
    io:println(numeric); // @output 2021-04-12T23:20:50+05:30

    // civilToString with "Z" in timeAbbrev (no utcOffset)
    // -> civilNamedZoneTime UTC path
    string utc = check time:civilToString({
        year: 2021,
        month: 4,
        day: 12,
        hour: 17,
        minute: 50,
        second: 50,
        timeAbbrev: "Z"
    });
    io:println(utc); // @output 2021-04-12T17:50:50Z

    // civilToString with neither utcOffset nor timeAbbrev -> error (zoneHandlingFor)
    string|error noZone = time:civilToString({
        year: 2021,
        month: 4,
        day: 12,
        hour: 17,
        minute: 50
    });
    io:println(noZone is error); // @output true

    // civilToString with a utcOffset that carries a seconds field
    // -> zoneOffsetFields reads the optional seconds
    string withSeconds = check time:civilToString({
        year: 2021,
        month: 4,
        day: 12,
        hour: 23,
        minute: 20,
        second: 50,
        utcOffset: {hours: 5, minutes: 30, seconds: 30}
    });
    io:println(withSeconds); // @output 2021-04-12T23:20:50+05:30

    // TimeZone with a negative fixed offset -> buildZoneOffset negative path
    time:TimeZone west = check new time:TimeZone("-05:00");
    time:ZoneOffset? westOffset = west.fixedOffset();
    if westOffset !is () {
        io:println(westOffset.hours);   // @output -5
        io:println(westOffset.minutes); // @output 0
    }

    // TimeZone with an out-of-range numeric offset -> initNative parseNumericOffset error
    time:TimeZone|error tooBig = new time:TimeZone("+25:00");
    io:println(tooBig is error); // @output true

    // TimeZone.utcFromCivil with an invalid calendar date -> civilMapToGoTimeInLocation error
    time:TimeZone utcZone = check new time:TimeZone("UTC");
    time:Utc|error invalidDate = utcZone.utcFromCivil({
        year: 2021,
        month: 2,
        day: 30,
        hour: 10,
        minute: 0,
        timeAbbrev: "Z"
    });
    io:println(invalidDate is error); // @output true

    // civilToEmailString with the time-abbrev comment handling
    // -> appends the "(abbrev)" comment, exercising mapString
    string emailComment = check time:civilToEmailString({
        year: 2024,
        month: 6,
        day: 15,
        hour: 10,
        minute: 30,
        second: 45,
        utcOffset: {hours: 0, minutes: 0},
        timeAbbrev: "GMT"
    }, time:ZONE_OFFSET_WITH_TIME_ABBREV_COMMENT);
    io:println(emailComment); // @output Sat, 15 Jun 2024 10:30:45 +0000 (GMT)

    // civilToEmailString with the comment handling but no timeAbbrev field
    // -> mapString falls back to "" and no comment is appended
    string noAbbrev = check time:civilToEmailString({
        year: 2024,
        month: 6,
        day: 15,
        hour: 10,
        minute: 30,
        second: 45,
        utcOffset: {hours: 0, minutes: 0}
    }, time:ZONE_OFFSET_WITH_TIME_ABBREV_COMMENT);
    io:println(noAbbrev); // @output Sat, 15 Jun 2024 10:30:45 +0000
}
