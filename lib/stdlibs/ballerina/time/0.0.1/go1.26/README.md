# Ballerina Time Library

## Overview

The `ballerina/time` library provides types and functions for working with UTC time, civil (local) time, time zones, duration-based date arithmetic, the full UTC/Civil conversion surface, RFC 3339 and RFC 5322 formatting, and timezone-aware operations via `TimeZone` and related APIs.

## Key Functionalities

- Parse and format RFC 3339 timestamps (`utcFromString`, `utcToString`)
- Get the current UTC time (`utcNow`) and monotonic elapsed time (`monotonicNow`)
- Add/subtract seconds from a UTC time (`utcAddSeconds`) and compute the difference (`utcDiffSeconds`)
- Convert between UTC and civil date-time values (`utcToCivil`, `utcFromCivil`)
- Parse and format RFC 3339 civil strings with timezone offsets (`civilFromString`, `civilToString`)
- Parse and format RFC 5322 / email-format date strings (`civilFromEmailString`, `civilToEmailString`, `utcToEmailString`)
- Add calendar durations to civil times (`civilAddDuration`)
- Validate dates and determine the day of week (`dateValidate`, `dayOfWeek`)
- Create timezone objects via IANA name, fixed offset, or system default (`TimeZone`, `loadSystemZone`, `getZone`)
- Convert between UTC and civil time in a specific timezone (`TimeZone.utcToCivil`, `TimeZone.utcFromCivil`)
- Add durations to civil times with timezone awareness (`TimeZone.civilAddDuration`)
- Query whether a zone always has a constant UTC offset (`TimeZone.fixedOffset`)

## Examples

```ballerina
import ballerina/io;
import ballerina/time;

public function main() returns error? {
    // Parse and round-trip a UTC timestamp
    time:Utc utc = check time:utcFromString("2007-12-03T10:15:30.00Z");
    io:println(time:utcToString(utc));           // 2007-12-03T10:15:30Z

    // Add 20.9 seconds
    time:Utc utc2 = time:utcAddSeconds(utc, 20.9);
    io:println(time:utcToString(utc2));           // 2007-12-03T10:15:50.900Z

    // Convert to Civil and inspect fields
    time:Civil civil = time:utcToCivil(utc);
    io:println(civil.year);                       // 2007
    io:println(time:dayOfWeek({year: 2007, month: 12, day: 3})); // 1 (Monday)

    // Civil with fixed offset
    time:Civil civil2 = check time:civilFromString("2021-04-12T23:20:50.520+05:30");
    io:println(check time:civilToString(civil2)); // 2021-04-12T23:20:50.520+05:30

    // Email format
    io:println(time:utcToEmailString(utc));       // Mon, 3 Dec 2007 10:15:30 +0000

    // Add a duration
    time:Civil updated = check time:civilAddDuration(civil2, {years: 1, days: 3, hours: 4});
    io:println(check time:civilToString(updated));

    // Timezone-aware conversion
    time:TimeZone tz = check new time:TimeZone("Asia/Colombo");
    time:Civil colombo = tz.utcToCivil(utc);
    io:println(colombo.hour);                     // 15 (+05:30 ahead of UTC)

    // Round-trip via TimeZone.utcFromCivil
    time:Utc back = check tz.utcFromCivil(colombo);
    io:println(time:utcToString(back));           // 2007-12-03T10:15:30Z
}
```

## Ballerina Nutcracker Support Status

This library is currently being migrated to Go to support Ballerina Nutcracker. The table below outlines the current support level for various features of this library in the Go implementation.

Support Levels:

- **Supported**: Fully implemented and tested in the Go version.
- **Partially Supported**: Implemented but lacking some edge cases, options, or sub-features. (See comments).
- **Not Yet Supported**: Planned for migration, but not yet implemented.
- **Cannot Support**: Cannot be implemented in the Go version due to technical limitations or architectural differences. (See comments).

| Feature/API | Support Status | Comments / Limitations |
|---|---|---|
| Seconds type | Supported | |
| UTC type | Supported | `readonly &` intersection dropped — equivalent mutable tuple type used (see Notable Behavioural Changes) |
| ZoneOffset record | Supported | `readonly &` intersection dropped — equivalent mutable record used (see Notable Behavioural Changes) |
| Day-of-week constants and type | Supported | |
| Date record | Supported | |
| TimeOfDay record | Supported | |
| Civil record | Supported | |
| Duration record | Supported | |
| Z constant | Supported | |
| UtcZoneHandling type | Supported | |
| HeaderZoneHandling enum | Supported | |
| Current UTC time | Supported | Uses `pal.Time.Now` |
| Monotonic elapsed time | Supported | Uses `pal.Time.MonotonicNow`; epoch is unspecified (see Notable Behavioural Changes) |
| Parse RFC 3339 string to UTC | Supported | Uses `time.RFC3339Nano` |
| Format UTC to RFC 3339 string | Supported | 0/3/6/9-digit sub-second grouping matches Java's `Instant.toString()` |
| Add seconds to UTC | Supported | Implemented as Go extern for correct negative-second handling |
| Compute difference between two UTC values | Supported | |
| Validate a Date | Supported | Error message wording differs from jBallerina (see Notable Behavioural Changes) |
| Day of week from a Date | Supported | |
| Convert UTC to Civil | Supported | |
| Convert Civil to UTC | Supported | Handles `timeAbbrev = "Z"` case |
| Parse RFC 3339 string to Civil | Supported | Correctly omits `utcOffset` for Z-terminated strings; sets `timeAbbrev = "Z"`; supports RFC 9557 IANA zone annotation (`[Zone/Name]` suffix) |
| Format Civil to RFC 3339 string | Supported | Fixed-offset and UTC zones supported; named IANA zones via `PREFER_TIME_ABBREV` require the IANA database on the host |
| Format UTC to email string | Supported | Non-zero-padded day matches Java's `DateTimeFormatter.RFC_1123_DATE_TIME` |
| Parse email string to Civil | Supported | Handles optional `(comment)` for time abbreviation |
| Format Civil to email string | Supported | Supports all three `HeaderZoneHandling` modes |
| Add duration to Civil | Supported | Timezone-agnostic; `weeks` field is normalised to days |
| Zone abstract object type | Supported | Declared as plain `object` type; `readonly &` prefix dropped because `readonly & object` type descriptors are not yet supported |
| TimeZone class | Supported | Declared as plain `class`; `readonly` qualifier dropped because readonly classes are not yet supported |
| Load system timezone | Supported | Uses `time.Local`; delegates to the host OS timezone database |
| Get named timezone | Supported | `getZone` returns nil for any invalid zone ID rather than an error |
| distinct error types | Partially Supported | `FormatError` is currently an alias for `error`; `distinct` type descriptors not yet supported in the interpreter |

### Notable Behavioural Changes

- **`Utc` type mutability.** jBallerina declares `Utc` as `readonly & [int, decimal]` (immutable tuple). The Go-native version uses a plain mutable tuple type because `readonly &` intersection types on tuples are not yet supported by the interpreter's AST transformation. Programs should treat `Utc` values as immutable by convention; mutation is not guarded at runtime.
- **`ZoneOffset` type mutability.** Same as above — `ZoneOffset` is declared as a plain open record instead of `readonly & record {| ... |}`. Programs should not mutate `ZoneOffset` values.
- **`FormatError` is not distinct.** jBallerina's `FormatError` is a `distinct Error` subtype, allowing `error is time:FormatError` checks to distinguish it from other errors. The Go-native version declares `FormatError` as a plain `error` alias because `distinct` type descriptors are not yet supported. `error is time:FormatError` will not narrow correctly in the Go version.
- **Error message wording for `dateValidate`, `dayOfWeek`, `utcFromCivil`, `TimeZone.init`, `TimeZone.utcFromCivil`.** These functions return errors whose message text is produced by Go's standard `time` package or the Go-native implementation rather than Java's `DateTimeException.getMessage()`. The message content differs (e.g., "invalid date: 2021-02-30" vs. "Invalid value for DayOfMonth..."). Programs must not depend on the exact error message text.
- **`monotonicNow()` epoch.** The specification states the epoch is "unspecified". jBallerina uses the JVM process start (`System.nanoTime()`); the Go-native version uses the time at which the PAL was constructed. The two values are not comparable across processes and will differ between implementations. This is expected behavior.
- **Named IANA timezones in `civilToString`, `civilToEmailString`, and `TimeZone`.** When a `Civil` record carries a `timeAbbrev` containing an IANA zone name (e.g., `"Asia/Colombo"`), or when a `TimeZone` object is constructed from an IANA name, the Go-native version resolves the zone using the host operating system's timezone database via `time.LoadLocation`. If the host has an incomplete or missing IANA database, an error is returned. jBallerina ships its own bundled IANA data.
- **DST disambiguation in `TimeZone.utcFromCivil`.** When a civil time falls in an ambiguous DST window (clocks are set back), Go's `time.Date` resolves to the first (standard-time) occurrence. jBallerina honours the `which` field in the `Civil` record to select the correct occurrence. The `which` field is silently ignored in the Go-native version.
