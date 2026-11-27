# Supported ballerina library features

Subset 4 extends the released [subset 3](subset3.md) with the new `ballerina/data.jsondata` package, which converts JSON values, text, bytes and byte-block streams to a user-specified type with data projection, and converts `anydata` to `json`.

## [data.jsondata](https://github.com/ballerina-platform/module-ballerina-data.jsondata)

Type-directed JSON conversion. The parse functions take an inferred `typedesc<anydata>` target and return `t|jsondata:Error`; the text parsers convert while they read, with jBallerina's error positions. Unlike jBallerina, they reject a number with a leading zero, invalid UTF-8 input and an unpaired surrogate escape (see the package README).

| Function | Notes |
|---|---|
| `parseAsType` | Convert an in-memory `json` value to the target type |
| `parseString` / `parseBytes` | Parse JSON text, or UTF-8 bytes, into the target type |
| `parseStream` | Parse a `stream<byte[], error?>` into the target type; returns errors instead of panicking as jBallerina does |
| `toJson` | Convert `anydata` to `json`; lists become `json[]`, mappings `map<json>` and `xml` its string form; a cyclic value, or one nested more than 10000 levels deep, panics |

Types: `jsondata:Options` (`allowDataProjection` with `nilAsOptionalField` and `absentAsNilableType`; `enableConstraintValidation` is accepted and ignored), `jsondata:NameConfig`, the `@jsondata:Name` field annotation (applied when the target record type is passed explicitly), `jsondata:Error`. `read` (JSONPath) and `prettify` are not included.
