# Ballerina Data.jsondata Library

## Overview

This module converts between JSON and Ballerina's `anydata` values. The full jBallerina `data.jsondata` module parses JSON from an in-memory `json` value, a string, a byte array or a byte-block stream into a user-specified target type, with data projection options and constraint validation; converts `anydata` to `json`; renames record fields with the `@jsondata:Name` annotation; queries JSON with JSONPath expressions; and prettifies JSON for printing. A compiler plugin also checks the target types at compile time. The Go Native Interpreter currently supports parsing from all four sources with data projection, conversion to `json`, and field renaming for the target record's own fields.

## Key Functionalities

- Convert an in-memory `json` value to a target type with `parseAsType`.
- Parse JSON text into a target type with `parseString`, `parseBytes` and `parseStream`, converting while reading as jBallerina does.
- Control projection with `Options`: drop or reject fields and list members that the target type does not have, read `null` as an absent optional field, and read an absent nilable field as `null`.
- Convert any `anydata` value, including `xml`, to `json` with `toJson`.
- Rename record fields with `@jsondata:Name` when the target record type is passed explicitly.

## Examples

```ballerina
import ballerina/data.jsondata;
import ballerina/io;

type Book record {|
    @jsondata:Name {value: "book-title"}
    string title;
    int year;
|};

type Summary record {|
    string title;
|};

public function main() returns error? {
    Book book = check jsondata:parseString(string `{"book-title":"Dune","year":1965}`, {}, Book);
    io:println(book.title); // Dune

    Summary summary = check jsondata:parseAsType({title: "Emma", year: 1815});
    io:println(summary); // {"title":"Emma"}

    json j = jsondata:toJson({title: "Emma", tags: ["classic"]});
    io:println(j); // {"title":"Emma","tags":["classic"]}
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
| Converting a JSON value to a target type | Partially Supported | `parseAsType`. A record field with a default value is still required, because native code cannot read field defaults (ballerina-nutcracker/ballerina#1378). Union members are tried with a member the value already belongs to first, not in declaration order (#1380), so `float\|int` given `1` yields an `int` where jBallerina yields a `float`. Record values keep JSON key order rather than declaration order (#1381). |
| Parsing JSON text into a target type | Partially Supported | `parseString`. Same record-default, union-order and field-order gaps as `parseAsType`; a mapping or list read for a union target is converted with the same union order. |
| Parsing a JSON byte array | Partially Supported | `parseBytes`. Same gaps as `parseString`. |
| Parsing a JSON byte-block stream | Partially Supported | `parseStream`. Same gaps as `parseString`. The inferred target type cannot be used under `trap` (#1385); pass it explicitly. |
| Data projection options | Supported | `Options.allowDataProjection` with `nilAsOptionalField` and `absentAsNilableType`. |
| Read-only target types | Supported | |
| Error messages | Partially Supported | Message texts, field paths, numeric formatting and syntax-error line and column numbers match jBallerina. A target type is printed structurally instead of by its declared name (#1354), anonymous types do not use Ballerina type syntax (#1386), and floats print in Go exponent form (#1384). |
| Converting anydata to JSON | Partially Supported | `toJson`. `@jsondata:Name` overrides are not applied, because a record value does not keep its record type at runtime (#1379). |
| Field name override annotation | Partially Supported | `@jsondata:Name` and `NameConfig`. Applied only when the target record type is passed explicitly, as in `parseString(s, {}, Book)`, and only to records of that record's type, including a recursive reference to it: annotations are not visible through an inferred typedesc or a nested type (#1136). A target type is matched by its shape, so a distinct nested record type with the same fields as the target is renamed too. |
| Module error type | Supported | `jsondata:Error` is a `distinct` error. An inferred target type that wrongly includes it (#1383) is ignored. |
| Compile-time target type validation | Partially Supported | The interpreter has no compiler-plugin support (#453), so the jBallerina diagnostics `JSON_ERROR_202` (duplicate field after a `@jsondata:Name` override) and `JSON_ERROR_203` (`xml` or `table` target) are reported at runtime, and only for the target record's own fields and an `xml` or `table` type a value is parsed into; see Notable Behavioural Changes. Arguments of the parse functions are not type-checked at compile time (#1388). |
| Constraint validation | Not Yet Supported | `Options.enableConstraintValidation` is accepted and ignored: `ballerina/constraint` is not ported yet, so no constraint annotations can be declared. |
| JSONPath queries | Not Yet Supported | `read`, `JsonPathValue` and `JsonPathRawTemplate` are not declared. jBallerina evaluates them with Jayway JsonPath, whose semantics differ from RFC 9535. |
| Prettifying JSON | Not Yet Supported | `prettify` is not declared: it is written in Ballerina on top of `value:toJsonString`, which the interpreter does not provide yet (#1368). |

### Notable Behavioural Changes

- **Parsing a stream returns errors instead of panicking.** jBallerina's `parseStream` lets every conversion and syntax error escape as a panic even though it declares `returns t|Error`; the Go-native version returns a `jsondata:Error` with the same message, as the signature describes.
- **An incomplete document is reported as a module error.** jBallerina reports a document that ends right after a field's `:` as a plain `error` with the message `invalid JSON document`, which then fails its own `t|Error` return type and panics with a `TypeCastError`; the Go-native version returns a `jsondata:Error` with the same message. A decimal token out of the `decimal` range is likewise returned as a `jsondata:Error` instead of panicking.
- **An empty list member is reported as an error.** For input such as `[1,,2]`, jBallerina throws a Java `StringIndexOutOfBoundsException`; the Go-native version returns a `jsondata:Error` naming the expected type and the empty value.
- **A failed parse does not affect the next one.** jBallerina reuses one parser per thread and keeps the characters of a token it failed in the middle of, so the first token of the next parse on that thread is read with them prepended; the Go-native version starts every parse from an empty state.
- **The first missing required field is reported in field-name order.** When several required fields are absent, jBallerina reports the first one in Java `HashMap` order; the Go-native version reports the first one in code-point order of the field names, so the message is deterministic.
- **Compiler-plugin checks are partly replaced by runtime errors.** jBallerina's compiler plugin rejects a record whose `@jsondata:Name` overrides clash, and an `xml` or `table` type in the target of `parseString`, `parseBytes` or `parseStream`, as compile errors. This port has no compiler plugin, so those programs compile, and a parse call returns a `jsondata:Error` only for what it reaches: a clash among the target record's own fields (`duplicate field '<name>'`), and an `xml` or `table` type that a value is parsed into (`unsupported type '<type>'` for the target itself, an incompatible-value error for a member or field). A clashing record nested in the target, or an `int|xml` target given an `int`, parses without error.
- **An unpaired surrogate escape is rejected.** jBallerina keeps a lone UTF-16 surrogate from a `\u` escape, such as `"\uD800"`, `"\uDC00"` or `"\uD800\u0041"`, in the string without an error, producing a string that is not a sequence of Unicode scalar values (its `toBytes()` encodes the surrogate as `?`) ([ballerina-library#9427](https://github.com/ballerina-platform/ballerina-library/issues/9427)); the Go-native version returns a `jsondata:Error` naming the escape and its position, such as `'unpaired surrogate escape '\uD800'' at line: '1' column: '8'`. An escaped surrogate pair such as `"\uD83D\uDE00"` forms one character in both.
- **Invalid UTF-8 input is rejected.** jBallerina's `parseBytes` replaces a byte that is not valid UTF-8 with U+FFFD and its `parseStream` drops it, so `"a\xFFb"` silently parses as a different string ([ballerina-library#9415](https://github.com/ballerina-platform/ballerina-library/issues/9415)); the Go-native version returns a `jsondata:Error` naming the byte offset, such as `invalid UTF-8 byte at offset 2`.
- **A number with a leading zero is rejected.** jBallerina's `parseString`, `parseBytes` and `parseStream` accept `01`, `-01`, `00` and `01.5`, at the top level or inside a list or mapping, which RFC 8259 does not allow ([ballerina-library#9416](https://github.com/ballerina-platform/ballerina-library/issues/9416)); the Go-native version returns a `jsondata:Error` such as `'leading zero in number '01'' at line: '1' column: '4'`. `0`, `-0` and `0.5` stay valid.
- **An error from a parsed stream is returned.** jBallerina's `parseStream` treats an error from the byte-block stream as the end of the input and parses the bytes read so far, so a truncated stream can produce a wrong value; the Go-native version returns a `jsondata:Error` with the stream error's message and that error as its cause.
- **Union-typed structures inside a list keep every member.** When a list member's type is a union that contains a list or mapping type, as in `(int[]|string)[]` or `(A|int)[2]`, jBallerina's parser loses track of the member index and the parent's expected type, so later members overwrite earlier ones or are checked against the union; for example `[[1],[2],[3]]` parses as `[[3]]`. The Go-native version keeps every member at its index and checks each against the list's member type.
- **A JSON key naming a renamed field is rejected.** For an open record whose field `title` is read from the key `t` by `@jsondata:Name`, a key `title` in the input is neither that field nor a valid rest field. jBallerina's `parseString` writes it over the field and `parseAsType` panics with a `ClassCastException`; the Go-native version returns `duplicate field 'title'` from both.
