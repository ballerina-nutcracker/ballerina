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
    // No-colon "+HHMM" offset form (isNumericOffset / parseNumericOffset HHMM branch).
    time:TimeZone east = check new time:TimeZone("+0530");
    time:ZoneOffset? eastOffset = east.fixedOffset();
    if eastOffset !is () {
        io:println(eastOffset.hours);   // @output 5
        io:println(eastOffset.minutes); // @output 30
    }

    // No-colon negative "+HHMM" form.
    time:TimeZone west = check new time:TimeZone("-0800");
    time:ZoneOffset? westOffset = west.fixedOffset();
    if westOffset !is () {
        io:println(westOffset.hours); // @output -8
    }

    // Out-of-range minutes in the colon form (parseNumericOffset range check).
    time:TimeZone|error badMin = new time:TimeZone("+12:99");
    io:println(badMin is error); // @output true

    // Out-of-range hours in the no-colon form.
    time:TimeZone|error badHour = new time:TimeZone("+2400");
    io:println(badHour is error); // @output true

    // Strings that are not numeric offsets fall through to IANA zone lookup,
    // which fails for these malformed inputs (isNumericOffset false branches).
    time:TimeZone|error bad1 = new time:TimeZone("+1a:30");
    io:println(bad1 is error); // @output true
    time:TimeZone|error bad2 = new time:TimeZone("+12:a0");
    io:println(bad2 is error); // @output true
    time:TimeZone|error bad3 = new time:TimeZone("+12a0");
    io:println(bad3 is error); // @output true

    // A numeric timeAbbrev that is out of range fails inside the named-zone path
    // (civilNamedZoneTime parseNumericOffset error branch).
    string|error badAbbrev = time:civilToString({
        year: 2021, month: 4, day: 12, hour: 23, minute: 20, second: 50,
        timeAbbrev: "+12:99"
    });
    io:println(badAbbrev is error); // @output true

    // An unknown IANA zone name in timeAbbrev fails the named-zone lookup
    // (civilNamedZoneTime LoadLocation error branch).
    string|error unknownZone = time:civilToString({
        year: 2021, month: 4, day: 12, hour: 23, minute: 20, second: 50,
        timeAbbrev: "Bogus/Zone"
    });
    io:println(unknownZone is error); // @output true

    // utcFromCivil with a fixed utcOffset and an invalid calendar date fails in
    // the fixed-offset path (civilFixedOffsetTime invalid-date branch).
    time:Utc|error badDate = time:utcFromCivil({
        year: 2021, month: 2, day: 30, hour: 10, minute: 0, second: 0,
        utcOffset: {hours: 5, minutes: 30}
    });
    io:println(badDate is error); // @output true

    // A civil record without the optional `second` field is formatted by reading
    // the field via the mapDecimal zero fallback.
    string noSec = check time:civilToString({
        year: 2021, month: 4, day: 12, hour: 23, minute: 20,
        timeAbbrev: "+05:30"
    });
    io:println(noSec); // @output 2021-04-12T23:20:00+05:30
}
