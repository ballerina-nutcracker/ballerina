# Ballerina Data CSV Library

## Overview

The `data.csv` module converts between CSV data and Ballerina values. It parses CSV text, bytes and byte-block streams into arrays of records, maps, arrays or tuples whose type is inferred from the call site, converts arrays of records and string lists to those types, and supports header handling, line skipping, data projection, nil representations, locale-aware number parsing, fail-safe error logging and constraint validation. The Go Native Interpreter currently supports the parse, transform and list conversion functions with their options, except constraint validation, field name annotations, parsing to a stream, and non-Latin character encodings.

## Key Functionalities

- Parse a CSV string, byte array or byte-block stream into an array of records, maps, arrays or tuples.
- Convert an array of records, or a list of string lists, to any of those types.
- Configure the delimiter, text enclosure, escape and comment characters, the header row, custom headers and the nil representation.
- Skip data rows, keep the header row in list results, and control data projection.
- Convert values to the field or member type, including unions, finite types, integer subtypes and locale-aware numbers.
- Tolerate per-row errors with fail-safe logging to the console and to a file.

## Examples

```ballerina
import ballerina/data.csv;
import ballerina/io;

type Employee record {|
    int id;
    string name;
    decimal? salary;
|};

public function main() returns error? {
    string csvString = string `id,name,salary
1,"Doe, John",100.50
2,Jane,null`;
    Employee[] employees = check csv:parseString(csvString);
    io:println(employees);

    [int, string][] pairs = check csv:parseString(csvString, {skipLines: [1]});
    io:println(pairs);

    anydata[][] columns = check csv:transform(employees, {headerOrder: ["name", "id", "salary"]});
    io:println(columns);
}
```

## Go Native Interpreter Support Status

This library is currently being migrated to Go to support the Ballerina Native Interpreter. The table below outlines the current support level for various features of this library in the Go implementation.

Support Levels:

- **Supported**: Fully implemented and tested in the Go version.
- **Partially Supported**: Implemented but lacking some edge cases, options, or sub-features. (See comments).
- **Not Yet Supported**: Planned for migration, but not yet implemented.
- **Cannot Support**: Cannot be implemented in the Go version due to technical limitations or architectural differences. (See comments).

| Feature/API | Support Status | Comments / Limitations |
|---|---|---|
| Parse a CSV string | Supported | `parseString` |
| Parse a byte array | Partially Supported | `parseBytes` decodes only the UTF-8, US-ASCII and ISO-8859-1 encodings; any other `encoding` returns the error jBallerina returns for an unknown charset. |
| Parse a byte-block stream | Partially Supported | `parseStream` has the same encoding limitation as `parseBytes`. |
| Parse into a stream of records | Not Yet Supported | `parseToStream` is not declared: a typedesc parameter cannot yet be used inside a `stream` type in a dependently-typed return type (ballerina-nutcracker/ballerina#1389). |
| Convert records | Supported | `transform` |
| Convert string lists | Supported | `parseList` |
| Record targets | Partially Supported | Open, closed, optional, nilable and rest fields are supported. Field default values are not filled in, so a defaulted field without a column is reported as missing (ballerina-nutcracker/ballerina#1378), and declared fields follow the column order rather than the declaration order (ballerina-nutcracker/ballerina#1381). |
| Map, array and tuple targets | Supported | Including closed result arrays, closed inner arrays and tuples with a rest type. |
| Read-only targets | Supported | |
| Union targets | Partially Supported | A union of array types and an array of a union type pick the first member every row converts to, as in jBallerina. Union members are tried in a fixed order (nil, boolean, int, float, decimal, string, then structured types) instead of the declaration order, which native code cannot see (ballerina-nutcracker/ballerina#1380). |
| Field name annotation | Not Yet Supported | `@csv:Name` is declared but ignored: inferred typedesc values carry no field annotations (ballerina-nutcracker/ballerina#1136, ballerina-nutcracker/ballerina#1379). |
| Header row options | Supported | `header`, `customHeadersIfHeadersAbsent` |
| Delimiter, text enclosure, escape and comment characters | Supported | `delimiter`, `textEnclosure`, `escapeChar`, `comment` |
| Line terminator option | Supported | As in jBallerina, `lineTerminator` has no effect: a line feed always ends a line and carriage returns are dropped. |
| Nil value option | Supported | `nilValue` |
| Locale option | Partially Supported | `locale` affects only the decimal separator and the minus sign. A comma separator is applied for common European languages and a U+2212 minus sign for Swedish, Norwegian, Finnish, Estonian, Lithuanian and Slovenian; any other language uses the root symbols. |
| Skip lines option | Supported | `skipLines`, as a list of row numbers or a `"start-end"` range. |
| Output with headers option | Supported | `outputWithHeaders` |
| Data projection options | Supported | `allowDataProjection`, `nilAsOptionalField`, `absentAsNilableType` |
| Header rows and custom headers for lists | Supported | `headerRows`, `customHeaders` |
| Header order for records converted to lists | Supported | `headerOrder` |
| Fail-safe console logging | Partially Supported | `failSafe.enableConsoleLogs` and `includeSourceDataInConsole` log through `log:printError`, which inherits the log module's gaps: the output is always LOGFMT and the `module` field is empty. |
| Fail-safe file logging | Supported | `failSafe.fileOutputMode`, with all three content types and both write options. |
| Constraint validation | Not Yet Supported | `enableConstraintValidation` is accepted and ignored until `ballerina/constraint` is ported (ballerina-nutcracker/ballerina#732). |
| Compile-time validation of target types and options | Not Yet Supported | The jBallerina compiler plugin diagnostics are not emitted; the same problems are reported as runtime errors where jBallerina detects them at run time too. An inferred target type that is not a subtype of `record {}[]\|anydata[][]` is not rejected at compile time (ballerina-nutcracker/ballerina#1388). |
| Type names in error messages | Partially Supported | Types are shown in their structural form, such as `{\| a: int, never... \|}[]`, because declared type names are not available at runtime (ballerina-nutcracker/ballerina#1354). |
| Module error type | Supported | `csv:Error` is a plain `error` alias, as in jBallerina. |

### Notable Behavioural Changes

- **The character U+FFFF is ordinary data.** jBallerina uses U+FFFF as its end-of-input marker, so the character in the input makes the parser loop forever; the Go-native version parses it as any other character.
- **Fail-safe recovery skips exactly the offending line.** jBallerina locates the line to resume at within its current 1024-character read buffer only, so beyond the first buffer it drops the rest of the buffer, logs an empty offending row, or never terminates; the Go-native version always resumes at the line after the one that raised the error, logs that line as the offending row, and reports its physical line number as the row.
- **An `OVERWRITE` fail-safe log keeps every entry of a union parse.** For a union element type, jBallerina parses the rows and then converts them with two separate loggers that each truncate the file on their first write, so the conversion errors erase the parse errors already logged; the Go-native version truncates the file once per call and keeps both.
- **Fail-safe log files contain valid JSON.** jBallerina builds each JSON log line by concatenating the values without escaping them, so a row or message containing a quote produces an invalid line; the Go-native version escapes the values.
- **The options value is not modified.** jBallerina rewrites the options value it is given, so passing the same value to a second call silently resets `nilAsOptionalField` and `absentAsNilableType`; the Go-native version leaves the options value unchanged.
- **An exponent with an explicit plus sign is parsed in full.** jBallerina's number parsing stops at `E+`, so `1E+5` converts to `1.0`; the Go-native version converts it to `100000.0`.
- **Integer subtype ranges are checked on the full value.** jBallerina checks the `byte`, `int:Signed8`, `int:Signed16`, `int:Unsigned8` and `int:Unsigned16` ranges on the value truncated to 32 bits, so `4294967297` converts to the byte `1`; the Go-native version rejects any value outside the range.
- **A tuple whose members share one type behaves as a closed array.** Types are structural in the Go-native version, so `[int, int]` cannot be told apart from `int[2]`; its size errors name an array type instead of a tuple type, and its header row is checked against the array size.
- **Option characters outside the Basic Multilingual Plane match whole characters.** jBallerina compares the delimiter, text enclosure, escape and comment characters with single UTF-16 code units, so such a character never matches; the Go-native version matches the whole character.
- **An unpaired escaped UTF-16 surrogate becomes U+FFFD.** jBallerina keeps a lone surrogate from a `\u` escape in the string; Go strings cannot hold one, so the Go-native version substitutes the replacement character. An escaped surrogate pair forms one character in both.
- **Byte-block streams keep bytes with the value 0xFF.** jBallerina's stream reader returns the byte 0xFF as the end-of-stream marker, so `parseStream` input in an encoding that uses that byte is cut short; the Go-native version reads every byte.
- **A short row cannot leave a required field without a value.** jBallerina fills a required field that a row shorter than the header row does not reach with a default instead of rejecting the row ([ballerina-library#9285](https://github.com/ballerina-platform/ballerina-library/issues/9285)): `0`, `false` or `0.0` for an `int`, `boolean` or `float` field, and `()` for a `string` or `decimal` field even though the type does not allow `()`, producing a value outside the target type; the Go-native version returns the error `no matching header value is found for the required field '<name>'`, which `failSafe` logs and skips like any other row error. A nilable field is still set to `()` and an optional field is still left out.
