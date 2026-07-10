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

// Error is the generic module level error.
public type Error error;

// FormatError is returned when arguments are invalid or a string does not match the expected format.
// Note: distinct error types are not yet supported; FormatError is currently an alias for error.
public type FormatError error;

// Seconds holds a decimal value representing seconds.
public type Seconds decimal;

// Utc is a point on the UTC time-scale represented as [int, decimal].
// The first member is integral seconds from the UNIX epoch; the second is the fractional seconds.
public type Utc [int, decimal];

public const int SUNDAY = 0;
public const int MONDAY = 1;
public const int TUESDAY = 2;
public const int WEDNESDAY = 3;
public const int THURSDAY = 4;
public const int FRIDAY = 5;
public const int SATURDAY = 6;

// DayOfWeek represents a day of the week (0=Sunday .. 6=Saturday).
public type DayOfWeek SUNDAY|MONDAY|TUESDAY|WEDNESDAY|THURSDAY|FRIDAY|SATURDAY;

type DateFields record {
    int year;
    int month;
    int day;
};

type TimeOfDayFields record {
    int hour;
    int minute;
    Seconds second?;
};

type OptionalDateFields record {
    int year?;
    int month?;
    int day?;
};

type OptionalTimeOfDayFields record {
    int hour?;
    int minute?;
    Seconds second?;
};

// Date is a date in the proleptic Gregorian calendar.
public type Date record {
    *DateFields;
    *OptionalTimeOfDayFields;
    ZoneOffset utcOffset?;
};

// TimeOfDay is a time within a day.
public type TimeOfDay record {
    *OptionalDateFields;
    *TimeOfDayFields;
    ZoneOffset utcOffset?;
};

// ZoneOffset is a fixed UTC zone offset.
public type ZoneOffset record {|
    int hours;
    int minutes = 0;
    decimal seconds?;
|};

type ReadWriteZoneOffset record {|
    int hours;
    int minutes = 0;
    decimal seconds?;
|};

// Z represents the UTC zone offset (hours=0, minutes=0).
public final ZoneOffset Z = {hours: 0};

// ZERO_OR_ONE is either 0 or 1.
public type ZERO_OR_ONE 0|1;

// Civil is a date-time in a civil time zone.
public type Civil record {
    *DateFields;
    *TimeOfDayFields;
    ZoneOffset utcOffset?;
    string timeAbbrev?;
    ZERO_OR_ONE which?;
    DayOfWeek dayOfWeek?;
};

// UtcZoneHandling controls the zone string used in utcToEmailString.
public type UtcZoneHandling "0"|"GMT"|"UT"|"Z";

// Duration represents a time duration for adjusting civil date-time values.
public type Duration record {|
    int years = 0;
    int months = 0;
    int weeks = 0;
    int days = 0;
    int hours = 0;
    int minutes = 0;
    Seconds seconds = 0.0;
|};

// HeaderZoneHandling indicates how to handle zone offset vs time abbreviation in header formats.
public enum HeaderZoneHandling {
    PREFER_TIME_ABBREV,
    PREFER_ZONE_OFFSET,
    ZONE_OFFSET_WITH_TIME_ABBREV_COMMENT
}

// Zone is an abstract object type for handling time zones.
// Note: jBallerina declares this as `readonly & object`; Ballerina Nutcracker uses a plain
// object type because `readonly & object` type descriptors are not yet supported.
public type Zone object {
    # If always at a fixed offset from UTC, returns it; otherwise nil.
    # + return - The fixed zone offset or nil
    public isolated function fixedOffset() returns ZoneOffset?;

    # Converts a given `time:Civil` value to a `time:Utc` timestamp based on the time zone value.
    # + civil - The `time:Civil` value to be converted
    # + return - The corresponding `time:Utc` value or an error
    public isolated function utcFromCivil(Civil civil) returns Utc|Error;

    # Converts a given `time:Utc` timestamp to a `time:Civil` value based on the time zone value.
    # + utc - The `time:Utc` timestamp value to be converted
    # + return - The corresponding `time:Civil` value
    public isolated function utcToCivil(Utc utc) returns Civil;

    # Adds the given time duration to the specified civil date-time based on the time zone.
    # + civil - The civil time to which the duration should be added
    # + duration - The date-time duration to be added
    # + return - The civil time after adding the duration
    public isolated function civilAddDuration(Civil civil, Duration duration) returns Civil|Error;
};

# Localized time zone implementation for IANA zone IDs and fixed zone offsets.
# Note: jBallerina declares this as `readonly class`; Ballerina Nutcracker uses a plain class
# because readonly classes are not yet supported.
public class TimeZone {
    *Zone;

    # Initializes a TimeZone from a zone ID (e.g. "Asia/Colombo", "+05:30") or the system default.
    #
    # + zoneId - Zone ID as a string, or nil to use the system default timezone
    # + return - A `time:Error` if the zone ID is invalid, otherwise nil
    public isolated function init(string? zoneId = ()) returns Error? {
        return self.initNative(zoneId);
    }

    private isolated function initNative(string? zoneId) returns Error? = external;

    # If always at a fixed offset from UTC, returns it; otherwise nil.
    #
    # + return - The fixed zone offset or nil
    public isolated function fixedOffset() returns ZoneOffset? = external;

    # Converts a given `time:Civil` value to a `time:Utc` timestamp based on the time zone value.
    #
    # + civil - The `time:Civil` value to be converted
    # + return - The corresponding `time:Utc` value or an error
    public isolated function utcFromCivil(Civil civil) returns Utc|Error = external;

    # Converts a given `time:Utc` timestamp to a `time:Civil` value based on the time zone value.
    #
    # + utc - The `time:Utc` timestamp value to be converted
    # + return - The corresponding `time:Civil` value
    public isolated function utcToCivil(Utc utc) returns Civil = external;

    # Adds the given time duration to the specified civil date-time based on the time zone.
    # The operation assumes that all days have exactly 86,400 seconds.
    #
    # + civil - The civil time to which the duration should be added
    # + duration - The date-time duration to be added
    # + return - The civil time after adding the duration
    public isolated function civilAddDuration(Civil civil, Duration duration) returns Civil|Error = external;
}

# Loads the default time zone of the system.
#
# + return - Zone value or an error if the system zone ID is in invalid format
public isolated function loadSystemZone() returns Zone|Error {
    return check new TimeZone();
}

# Returns the time zone object for the given zone ID.
#
# + id - Time zone ID (e.g. "Asia/Colombo")
# + return - Corresponding time zone object or nil if the ID is invalid
public isolated function getZone(string id) returns Zone? {
    TimeZone|Error timeZone = new TimeZone(id);
    if timeZone is error {
        return;
    }
    return timeZone;
}

# Returns the UTC representing the current time.
#
# + precision - Number of decimal places in the fractional seconds (nil = nanosecond precision)
# + return - The `time:Utc` value corresponding to the current UTC time
public isolated function utcNow(int? precision = ()) returns Utc {
    if precision is int {
        return externUtcNow(precision);
    }
    return externUtcNow(-1);
}

# Returns seconds from an unspecified epoch with monotonic guarantee.
#
# + return - Number of seconds from an unspecified epoch
public isolated function monotonicNow() returns decimal {
    return externMonotonicNow();
}

# Converts from RFC 3339 timestamp to Utc.
#
# + timestamp - RFC 3339 timestamp string (e.g., `2007-12-03T10:15:30.00Z`)
# + return - The corresponding `time:Utc` or a `time:Error`
public isolated function utcFromString(string timestamp) returns Utc|Error {
    return externUtcFromString(timestamp);
}

# Converts a given `time:Utc` time to a RFC 3339 timestamp string.
#
# + utc - Utc time as a tuple `[int, decimal]`
# + return - The corresponding RFC 3339 timestamp string
public isolated function utcToString(Utc utc) returns string {
    return externUtcToString(utc);
}

# Returns UTC time that occurs seconds after `utc`.
#
# + utc - Utc time as a tuple `[int, decimal]`
# + seconds - Number of seconds to be added
# + return - The resulted `time:Utc` value after the summation
public isolated function utcAddSeconds(Utc utc, Seconds seconds) returns Utc {
    return externUtcAddSeconds(utc, seconds);
}

# Returns the difference in seconds between two UTC times.
#
# + utc1 - 1st Utc time
# + utc2 - 2nd Utc time
# + return - The difference between `utc1` and `utc2` in seconds
public isolated function utcDiffSeconds(Utc utc1, Utc utc2) returns Seconds {
    return externUtcDiffSeconds(utc1, utc2);
}

# Check that days and months are within range as per Gregorian calendar rules.
#
# + date - The date to be validated
# + return - `()` if valid or else `time:Error`
public isolated function dateValidate(Date date) returns Error? {
    return externDateValidate(date);
}

# Get the day of week for a specified date.
#
# + date - The date for which the day of the week is to be calculated
# + return - The `time:DayOfWeek` or panic if date is invalid
public isolated function dayOfWeek(Date date) returns DayOfWeek {
    return checkpanic externDayOfWeek(date);
}

# Converts a given `time:Utc` timestamp to a `time:Civil` value.
#
# + utc - The `time:Utc` timestamp value to be converted
# + return - The corresponding `time:Civil` value
public isolated function utcToCivil(Utc utc) returns Civil {
    return externUtcToCivil(utc);
}

# Converts a given `time:Civil` value to a `time:Utc` timestamp.
#
# + civilTime - The `time:Civil` value to be converted
# + return - The corresponding `time:Utc` value or an error if `utcOffset` is missing
public isolated function utcFromCivil(Civil civilTime) returns Utc|Error {
    return externUtcFromCivil(civilTime);
}

# Converts a given RFC 3339 timestamp to `time:Civil`.
#
# + dateTimeString - RFC 3339 timestamp string
# + return - The corresponding `time:Civil` value or an error
public isolated function civilFromString(string dateTimeString) returns Civil|Error {
    return check externCivilFromString(dateTimeString);
}

# Obtain a RFC 3339 timestamp string from a given `time:Civil`.
#
# + civil - The `time:Civil` value to be converted
# + return - The corresponding string value or an error
public isolated function civilToString(Civil civil) returns string|Error {
    return externCivilToString(civil);
}

# Converts a given UTC to an email formatted string (e.g., `Mon, 3 Dec 2007 10:15:30 +0000`).
#
# + utc - The `time:Utc` value to be formatted
# + zh - Type of the zone value to be added
# + return - The corresponding formatted string value
public isolated function utcToEmailString(Utc utc, UtcZoneHandling zh = "0") returns string {
    return externUtcToEmailString(utc, zh);
}

# Converts a given RFC 5322 formatted string to a civil record.
#
# + dateTimeString - RFC 5322 formatted string
# + return - The corresponding `time:Civil` record or an error
public isolated function civilFromEmailString(string dateTimeString) returns Civil|Error {
    return check externCivilFromEmailString(dateTimeString);
}

# Converts a given `time:Civil` record to RFC 5322 format.
#
# + civil - The `time:Civil` record to be converted
# + zoneHandling - Indicate how to handle the zone
# + return - The RFC 5322 formatted string or an error
public isolated function civilToEmailString(Civil civil, HeaderZoneHandling zoneHandling) returns string|Error {
    return externCivilToEmailString(civil, zoneHandling);
}

# Adds the given time duration to the specified civil date-time (timezone-agnostic).
#
# + civil - The civil time to which the duration should be added
# + duration - The time duration to be added
# + return - The civil time after adding the duration or an error
public isolated function civilAddDuration(Civil civil, Duration duration) returns Civil|Error {
    return externCivilAddDuration(civil, duration);
}

isolated function externUtcNow(int precision) returns Utc = external;
isolated function externMonotonicNow() returns Seconds = external;
isolated function externUtcFromString(string str) returns Utc|Error = external;
isolated function externUtcToString(Utc utc) returns string = external;
isolated function externUtcAddSeconds(Utc utc, Seconds seconds) returns Utc = external;
isolated function externUtcDiffSeconds(Utc utc1, Utc utc2) returns Seconds = external;
isolated function externDateValidate(Date date) returns Error? = external;
isolated function externDayOfWeek(Date date) returns DayOfWeek|Error = external;
isolated function externUtcToCivil(Utc utc) returns Civil = external;
isolated function externUtcFromCivil(Civil civil) returns Utc|Error = external;
isolated function externCivilFromString(string dateTimeString) returns Civil|Error = external;
isolated function externCivilToString(Civil civil) returns string|Error = external;
isolated function externUtcToEmailString(Utc utc, string zh) returns string = external;
isolated function externCivilFromEmailString(string dateTimeString) returns Civil|Error = external;
isolated function externCivilToEmailString(Civil civil, HeaderZoneHandling zoneHandling) returns string|Error = external;
isolated function externCivilAddDuration(Civil civil, Duration duration) returns Civil|Error = external;
