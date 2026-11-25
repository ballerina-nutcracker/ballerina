# Supported ballerina library features

Subset 4 extends the released [subset 3](subset3.md) with the new `ballerina/constraint` package, providing constraint annotations and `constraint:validate` for run-time validation.

## [constraint](https://github.com/ballerina-platform/module-ballerina-constraint/blob/master/docs/spec/spec.md)

Types: `constraint:Error`, `constraint:ValidationError`, `constraint:TypeConversionError`, `constraint:DateOption`, and the constraint record types for each annotation.

| API | Notes |
|---|---|
| `@constraint:Int`, `@constraint:Float`, `@constraint:Number` | `minValue`, `maxValue`, `minValueExclusive`, `maxValueExclusive`, and the digit-count constraints (`maxDigits` for `Int`, `maxIntegerDigits` and `maxFractionDigits` for `Float` and `Number`) |
| `@constraint:String` | `length`, `minLength` and `maxLength`, counted in UTF-16 code units as in jBallerina. The `pattern` constraint is not available until regular expressions are supported |
| `@constraint:Array` | `length`, `minLength` and `maxLength` |
| `@constraint:Date` | Gregorian calendar validation of `year`/`month`/`day` records and the `PAST`, `PAST_OR_PRESENT`, `FUTURE` and `FUTURE_OR_PRESENT` options |
| Custom messages | Any constraint can be written as `{value, message}`; `Date` also accepts a `message` for invalid dates |
| `constraint:validate(value, td)` | Converts the value to the target type and validates the annotations on the target type and on the fields of a target record. The target type must be named explicitly: an inferred target type (`T r = check constraint:validate(v)`) carries no annotations and validates nothing |

Not covered in this subset: validation through annotations on nested types (named field types, array members, union members and nested records), the `pattern` constraint, and the compile-time annotation diagnostics.
