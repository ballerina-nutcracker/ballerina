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
| Field name override annotation | Partially Supported | `@jsondata:Name` and `NameConfig`. Applied only when the target record type is passed explicitly, as in `parseString(s, {}, Book)`, and only to that record's own fields: annotations are not visible through an inferred typedesc or a nested type (#1136). |
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
