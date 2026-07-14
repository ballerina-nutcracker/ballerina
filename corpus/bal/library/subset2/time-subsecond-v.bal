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
    // utcToString sub-second grouping (formatInstantNanos): millis -> .NNN
    time:Utc ms = check time:utcFromString("2024-06-15T10:30:45.123Z");
    io:println(time:utcToString(ms)); // @output 2024-06-15T10:30:45.123Z

    // micros -> .NNNNNN
    time:Utc us = check time:utcFromString("2024-06-15T10:30:45.123456Z");
    io:println(time:utcToString(us)); // @output 2024-06-15T10:30:45.123456Z

    // nanos -> .NNNNNNNNN
    time:Utc ns = check time:utcFromString("2024-06-15T10:30:45.123456789Z");
    io:println(time:utcToString(ns)); // @output 2024-06-15T10:30:45.123456789Z

    // A single fractional digit is padded to milliseconds.
    time:Utc half = check time:utcFromString("2024-06-15T10:30:45.5Z");
    io:println(time:utcToString(half)); // @output 2024-06-15T10:30:45.500Z

    // civilToString with a negative numeric offset and fractional seconds
    // (formatRFC3339WithOffset negative-offset branch + formatInstantNanos).
    string neg = check time:civilToString({
        year: 2024, month: 6, day: 15, hour: 10, minute: 30, second: 45.5,
        timeAbbrev: "-08:00"
    });
    io:println(neg); // @output 2024-06-15T10:30:45.500-08:00

    // utcFromCivil with a negative utcOffset record (offsetToZoneName negative path).
    time:Utc fromNeg = check time:utcFromCivil({
        year: 2024, month: 6, day: 15, hour: 2, minute: 30, second: 45,
        utcOffset: {hours: -8, minutes: 0}
    });
    io:println(time:utcToString(fromNeg)); // @output 2024-06-15T10:30:45Z
}
