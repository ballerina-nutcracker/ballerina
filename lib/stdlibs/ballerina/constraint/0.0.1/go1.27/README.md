# Ballerina Constraint Library

## Overview

The `ballerina/constraint` library provides annotations for attaching constraints to Ballerina types and record fields, and a `validate` function that checks a value against those constraints. It covers numeric, string, array and date constraints, custom error messages, and compile-time checking of the annotations. The Go Native Interpreter currently supports run-time validation of the annotations on the target type and on the fields of a target record type.

## Key Functionalities

- Constrain `int`, `float` and `decimal` values with `minValue`, `maxValue`, `minValueExclusive`, `maxValueExclusive` and the digit-count constraints (`@constraint:Int`, `@constraint:Float`, `@constraint:Number`)
- Constrain `string` lengths with `length`, `minLength` and `maxLength` (`@constraint:String`)
- Constrain array lengths with `length`, `minLength` and `maxLength` (`@constraint:Array`)
- Validate `year`/`month`/`day` records against the Gregorian calendar and against the past, present and future (`@constraint:Date`)
- Attach custom messages to individual constraints
- Validate a value against an explicitly named target type with `constraint:validate`

## Examples

```ballerina
import ballerina/constraint;
import ballerina/io;

@constraint:Int {minValue: 18, maxValue: 100}
type Age int;

type User record {|
    @constraint:String {minLength: 3, maxLength: {value: 8, message: "name is too long"}}
    string name;
    @constraint:Int {minValue: 18}
    int age;
|};

public function main() {
    io:println(constraint:validate(30, Age)); // 30
    User|constraint:Error result = constraint:validate({name: "al", age: 10}, User);
    if result is constraint:Error {
        io:println(result.message()); // Validation failed for '$.age:minValue','$.name:minLength' constraint(s).
    }
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
| Integer constraints | Supported | |
| Float constraints | Supported | |
| Number constraints | Supported | |
| String length constraints | Supported | |
| String pattern constraint | Not Yet Supported | The `pattern` field of `StringConstraints` is omitted because `string:RegExp` and regular expression literals are not supported by the interpreter. |
| Array constraints | Supported | |
| Date constraints | Supported | |
| Custom error messages | Supported | |
| Validation against an explicit target type | Supported | `constraint:validate(value, T)`. |
| Validation against an inferred target type | Not Yet Supported | `T result = check constraint:validate(value)` validates nothing because the typedesc the compiler synthesizes for `<>` carries no annotations. Pass the type explicitly. |
| Constraints on nested types | Not Yet Supported | Only annotations on the target type itself and on the direct fields of a target record are validated. Annotations on named types used as field types, array members, union members or nested records are not visible at run time. |
| Type conversion before validation | Partially Supported | Values are converted with the same routine as `value:cloneWithType`, which does not yet fill in default values of record fields, so a value missing a defaulted field fails with `TypeConversionError`. |
| Module error types | Supported | `Error`, `ValidationError` and `TypeConversionError`. |
| Compile-time annotation validation | Not Yet Supported | The jBallerina compiler plugin diagnostics (`CONSTRAINT_101` to `CONSTRAINT_104`) are not reported, so an invalid annotation is accepted at compile time. |

### Notable Behavioural Changes

- **Digit counts are computed per value.** jBallerina caches the digit counts of the first float or number value it validates and reuses them for later fields in the same call, so a second `maxIntegerDigits` or `maxFractionDigits` field can be checked against the wrong counts; the Go-native version counts the digits of every value separately.
- **Custom messages from different fields are ordered by field name.** jBallerina joins the custom messages of failed constraints in the order the record fields are declared; the Go-native version joins them in field-name order, because record fields have no semantic order and the runtime exposes them unordered. The default `Validation failed for ...` part is sorted in both.
