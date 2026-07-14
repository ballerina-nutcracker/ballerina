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
    // utcNow with precision (covers goTimeToUtcWithPrecision, getInt64Arg)
    time:Utc utcWithPrecision = time:utcNow(3);
    io:println(utcWithPrecision.length() == 2); // @output true

    // utcAddSeconds and utcDiffSeconds
    time:Utc base = check time:utcFromString("2024-06-15T10:30:45.00Z");
    time:Utc later = time:utcAddSeconds(base, 3600.0d);
    time:Seconds diff = time:utcDiffSeconds(later, base);
    io:println(diff == 3600.0d); // @output true

    // dateValidate: valid date
    time:Error? valid = time:dateValidate({year: 2024, month: 6, day: 15});
    io:println(valid is ()); // @output true

    // dateValidate: invalid month
    time:Error? invalid = time:dateValidate({year: 2024, month: 13, day: 1});
    io:println(invalid is time:Error); // @output true

    // dayOfWeek
    time:DayOfWeek dow = time:dayOfWeek({year: 2024, month: 6, day: 15});
    io:println(dow == time:SATURDAY); // @output true

    // utcToCivil (covers buildCivil)
    time:Civil civil = time:utcToCivil(base);
    io:println(civil.year);   // @output 2024
    io:println(civil.month);  // @output 6
    io:println(civil.day);    // @output 15

    // utcFromCivil round-trip via TimeZone (covers civilMapToGoTimeInLocation path)
    time:TimeZone tz = check new time:TimeZone("+05:30");
    time:Utc utcBase = check time:utcFromString("2021-04-12T17:50:50.00Z");
    time:Civil civilOffset = tz.utcToCivil(utcBase);
    time:Utc roundTripped = check tz.utcFromCivil(civilOffset);
    io:println(time:utcToString(roundTripped)); // @output 2021-04-12T17:50:50Z

    // civilToString via TimeZone civil (covers civilToGoTime, formatRFC3339WithOffset)
    string civilStr = check time:civilToString(civilOffset);
    io:println(civilStr); // @output 2021-04-12T23:20:50+05:30

    // civilFromString (covers the RFC3339 parse path in initTimeModule)
    time:Civil parsed = check time:civilFromString("2021-04-12T23:20:50.520+05:30[Asia/Colombo]");
    io:println(parsed.year);  // @output 2021
    io:println(parsed.month); // @output 4
    io:println(parsed.day);   // @output 12
}
