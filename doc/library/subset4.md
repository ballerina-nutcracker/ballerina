# Supported ballerina library features

Subset 4 extends the released [subset 3](subset3.md) with the new `ballerina/data.csv` package, which parses CSV data into, and converts lists of records and strings to, arrays of records, maps, arrays and tuples typed from the call site.

## [data.csv](https://github.com/ballerina-platform/module-ballerina-data.csv)

Types: `csv:Options`, `csv:ParseOptions`, `csv:ParseListOptions`, `csv:TransformOptions`, `csv:FailSafeOptions`, `csv:FileOutputMode`, `csv:LogOutput`, `csv:Location`, `csv:NameConfig`, the `csv:LineTerminator`, `csv:NilValue`, `csv:ErrorLogContentType` and `csv:FileWriteOption` enums, and `csv:Error`.

| Function | Notes |
|---|---|
| `csv:parseString(csvString, options?, t?)` | Parse CSV text into `t`, a subtype of `record {}[]\|anydata[][]` inferred from the call site; returns `t\|csv:Error` |
| `csv:parseBytes(csvBytes, options?, t?)` | As `parseString`, decoding the bytes with `options.encoding`: UTF-8, US-ASCII or ISO-8859-1 |
| `csv:parseStream(csvByteStream, options?, t?)` | As `parseBytes`, reading a `stream<byte[], error?>` to its end; an error the stream completes with is returned as it is |
| `csv:transform(csvRecords, options?, t?)` | Convert a `record {}[]` to `t`; `headerOrder` sets the column order for list targets |
| `csv:parseList(csvList, options?, t?)` | Convert a `string[][]` to `t`; `headerRows` and `customHeaders` name the columns for record and map targets |

These options are supported: `delimiter`, `textEnclosure`, `escapeChar`, `comment`, `header`, `customHeadersIfHeadersAbsent`, `nilValue`, `locale` (decimal separator and minus sign), `skipLines`, `outputWithHeaders`, `allowDataProjection` with `nilAsOptionalField` and `absentAsNilableType`, and `failSafe` with console and file logging. `lineTerminator` is accepted and, as in jBallerina, has no effect. `enableConstraintValidation` validates no constraints; as in jBallerina, it only decides the error `parseString`, `parseBytes` and `parseStream` return when no member of a union target type fits. Union target types are resolved by trying their members in a fixed order rather than the declaration order, record fields with default values are reported as missing when no column provides them, and declared record fields follow the column order.

Not covered in this subset: `csv:parseToStream`, the `@csv:Name` field annotation, constraint validation, the compiler plugin diagnostics, and encodings other than UTF-8, US-ASCII and ISO-8859-1.
