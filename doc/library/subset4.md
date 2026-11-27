# Supported ballerina library features

Subset 4 extends the released [subset 3](subset3.md) with the new `ballerina/data.xmldata` package, which converts XML values, text, bytes and byte-block streams to record types and validates XML against a record type.

## [data.xmldata](https://github.com/ballerina-platform/module-ballerina-data.xmldata)

Conversion from XML to record types. The parse functions take an inferred `typedesc<record {}>` target and return `t|xmldata:Error`; the text parsers convert while they read, and a target with a union of records is parsed into an XML value and traversed instead, as in jBallerina.

| Function | Notes |
|---|---|
| `parseAsType` | Convert an `xml` value to the target record type |
| `parseString` / `parseBytes` / `parseStream` | Parse XML text, UTF-8 bytes or a `stream<byte[], error?>` into the target record type |
| `validate` | Check an `xml` value against a record type; an XSD file path is not supported and returns an `Error` |

Elements, attributes and text content map to record fields, nested records, arrays, maps, `json`, `anydata` and `xml` fields, and open-record rest fields. Types: `xmldata:Options` and `xmldata:SourceOptions` (`attributePrefix`, `textFieldName`, `allowDataProjection`, `useSemanticEquality`; `enableConstraintValidation` is accepted and ignored), `xmldata:NameConfig`, `xmldata:NamespaceConfig`, the `@xmldata:Name`, `@xmldata:Namespace` and `@xmldata:Attribute` annotations (applied when the target record type is passed explicitly), `xmldata:Error`. `toXml`, `fromJson`, `transform` (XPath), the XSD annotations and `@xmldata:Any` are not included.
