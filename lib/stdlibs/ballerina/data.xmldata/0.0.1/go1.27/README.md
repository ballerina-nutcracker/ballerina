# Ballerina Data.xmldata Library

## Overview

This module converts between XML and Ballerina record values. The full jBallerina `data.xmldata` module converts an XML value, XML text, a byte array or a byte-block stream into a record type, with options for projection, namespace matching, attribute prefixes and the text field name; renames elements, namespaces and attributes with the `@xmldata:Name`, `@xmldata:Namespace` and `@xmldata:Attribute` annotations; validates XML against a record type or an XSD file, including XSD element, sequence and choice rules declared with annotations; converts maps, records and JSON to XML; and queries XML with XPath. A compiler plugin also checks the target types at compile time. The Go Native Interpreter currently supports converting XML from all four sources into record types, the projection and namespace options, the name, namespace and attribute annotations on the target record's own fields, and validating XML against a record type.

## Key Functionalities

- Convert an XML value to a record type with `parseAsType`.
- Parse XML text, bytes or a byte-block stream into a record type with `parseString`, `parseBytes` and `parseStream`, converting while reading as jBallerina does.
- Map elements, attributes and text content to record fields, nested records, arrays, maps, `json`, `anydata` and `xml` fields, and open-record rest fields.
- Control the conversion with `SourceOptions`: drop or reject unexpected elements, match namespaces semantically or by prefix, prefix attribute field names, and rename the text field.
- Rename and namespace the target record's elements and attributes with `@xmldata:Name`, `@xmldata:Namespace` and `@xmldata:Attribute`.
- Validate an XML value against a record type with `validate`.

## Examples

```ballerina
import ballerina/data.xmldata;
import ballerina/io;

@xmldata:Name {value: "item"}
type Item record {|
    @xmldata:Attribute
    string code;
    @xmldata:Name {value: "item-name"}
    string itemName;
|};

type Book record {|
    int id;
    string[] tag;
|};

public function main() returns error? {
    Book book = check xmldata:parseString("<book><id>1</id><tag>a</tag><tag>b</tag></book>");
    io:println(book); // {"id":1,"tag":["a","b"]}

    Item item = check xmldata:parseAsType(xml `<item code="A1"><item-name>Pen</item-name></item>`, {}, Item);
    io:println(item.itemName); // Pen

    xmldata:Error? result = xmldata:validate(xml `<item><item-name>Pen</item-name></item>`, Item);
    io:println(result is xmldata:Error); // true
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
| Converting an XML value to a record type | Partially Supported | `parseAsType`. A record field with a default value is still required, because native code cannot read field defaults (ballerina-nutcracker/ballerina#1378). Union members are tried in a fixed order rather than in declaration order (#1380). Record values keep XML order rather than declaration order (#1381), and a field whose name needs escaping, such as `\#content`, keeps the backslash in its key (#1351). A field whose type is a union of a record and a singleton or enum, such as `Inner\|"none"` or `Inner\|Color`, is not converted from text that matches the singleton or an enum member: it returns `field '<name>' cannot be converted into the type '<type>'` for the record member's first field. jBallerina fails on the same input with a `NullPointerException` message ([ballerina-library#9426](https://github.com/ballerina-platform/ballerina-library/issues/9426)). `xml:fromString` does not accept an XML declaration (#1414), and `xml` templates cannot contain CDATA (#1412). |
| Parsing XML text into a record type | Partially Supported | `parseString`. Same field-default, union-order, record-and-singleton union, field-order and escaped-name gaps as `parseAsType`. |
| Parsing XML bytes into a record type | Partially Supported | `parseBytes`. Same gaps as `parseString`. |
| Parsing an XML byte-block stream into a record type | Partially Supported | `parseStream`. Same gaps as `parseString`. The inferred target type cannot be used under `trap` (#1385); pass it explicitly. |
| Projection and namespace options | Supported | `SourceOptions` with `allowDataProjection`, `useSemanticEquality`, `attributePrefix` and `textFieldName`. |
| Name, namespace and attribute annotations | Partially Supported | `@xmldata:Name`, `@xmldata:Namespace`, `@xmldata:Attribute`, `NameConfig` and `NamespaceConfig`. Applied only when the target record type is passed explicitly, as in `parseString(s, {}, Item)`, and only to that type and its own fields: annotations are not visible through an inferred typedesc or a nested type (#1136). |
| Validating XML against a record type | Partially Supported | `validate` with a record typedesc. Same gaps as `parseAsType`. |
| Validating XML against an XSD file | Not Yet Supported | `validate` with a file path returns an `Error`: no XSD validator is available to Go without cgo. |
| Error messages | Partially Supported | Message texts match jBallerina. A record type is printed structurally instead of by its declared name (#1354), and union and record types do not use Ballerina type syntax (#1386). Malformed XML is described by the Go XML tokenizer; see Notable Behavioural Changes. |
| Module error type | Supported | `xmldata:Error` is a `distinct` error. An inferred target type that wrongly includes it (#1383) is ignored. |
| Compile-time target type validation | Partially Supported | The interpreter has no compiler-plugin support (#453), so of the jBallerina diagnostics `XML_ERROR_201`–`XML_ERROR_211` only the duplicate-name check (`XML_ERROR_201`) is applied, when the type is used as a target; see Notable Behavioural Changes. Arguments of the parse functions are not type-checked at compile time (#1388). |
| Converting a map or record to XML | Not Yet Supported | `toXml`. It is written in Ballerina on top of `typeof` (#868), and record values do not keep their record type for annotation lookup (#1379). |
| Converting JSON to XML | Not Yet Supported | `fromJson` and `JsonOptions`. The Ballerina implementation needs list binding patterns (#752), `value:ensureType` (#1323), `value:clone` (#1337), `string:startsWith` (#1366) and `map:entries`/`map:toArray` (#1367, #573). |
| XPath queries | Not Yet Supported | `transform` and `XPathRawTemplate`. jBallerina evaluates XPath 3.1 with Saxon. |
| XSD element, sequence and choice rules | Not Yet Supported | `@xmldata:Element`, `@xmldata:Sequence`, `@xmldata:Choice`, `@xmldata:SequenceOrder` and their configuration types. Their rules are declared on nested record types, whose annotations do not reach native code (#1136). |
| Fields that accept any element | Not Yet Supported | `@xmldata:Any`. jBallerina matches elements by record type names, which are not available at runtime (#1354). |
| Constraint validation | Not Yet Supported | `SourceOptions.enableConstraintValidation` is accepted and ignored: `ballerina/constraint` is not ported yet, so no constraint annotations can be declared. |

### Notable Behavioural Changes

- **Malformed XML is described by the Go XML tokenizer.** jBallerina reports a syntax error with the Woodstox parser's message and position (for example `Unexpected EOF in prolog` followed by `at [row,col {unknown-source}]: [1,2]`); the Go-native version reports the error found by `encoding/xml`, the tokenizer the interpreter itself uses for XML, after the same `failed to parse xml: ` prefix.
- **A parse error for a union target is returned as a module error.** When the target type contains a union of records, jBallerina parses the text into an XML value first and returns that parse error as a plain `error`, which then fails its own `t|Error` return type and panics with a `TypeCastError`; the Go-native version returns an `xmldata:Error`.
- **The first missing required field is reported in field-name order.** When several required fields are absent, jBallerina reports the first one in Java `HashMap` order; the Go-native version reports the first one in code-point order of the field names, so the message is deterministic.
- **Compiler-plugin checks run when the type is used, or not at all.** jBallerina's compiler plugin rejects a record whose `@xmldata:Name` overrides clash, and a target that is not a record type such as `map<string>`, as compile errors. This port has no compiler plugin, so those programs compile: a clashing record makes the conversion functions return an `xmldata:Error` (`duplicate field '<name>'`), and a map target such as `map<string>`, being the same type at runtime as the record `record {| string...; |}`, is converted as that record by every parse function.
- **A required field that cannot hold `()` is never left as `()`.** When an empty element maps to a required field whose type does not contain `()`, such as `int|string`, or `xml` read by `parseString`, jBallerina leaves the field as `()` and returns a record value that does not belong to its type; the Go-native version returns `field '<name>' cannot be converted into the type '<type>'`.
- **A byte value is range-checked before it is narrowed.** jBallerina narrows the parsed value to a 32-bit `int` before checking the byte range, so a value such as `4294967297` is accepted as `1`; the Go-native version checks the full value, so it is handled like any other out-of-range byte such as `256` (an element value returns `'string' value '<value>' cannot be converted to 'byte'`).
- **A fixed-length array that cannot be filled is rejected.** When fewer elements than the array length are present and the element type has no filler value, such as a record with a required field, jBallerina fails with an internal `NullPointerException` message; the Go-native version returns `array size is not compatible with the expected size`.
- **A byte-block stream error is returned as a module error.** When the stream passed to `parseStream` returns an error, jBallerina returns that error itself, which does not belong to the `t|Error` return type; the Go-native version returns an `xmldata:Error` with the same message and closes the stream.
- **An absent fixed-length array field is checked like any other field.** When a record has a fixed-length array field, such as `Person[2] p?;`, and the XML has no element for it, jBallerina's `parseAsType` fails with an internal `NullPointerException` message whether or not the field is optional (ballerina-platform/ballerina-library#9431); the Go-native version leaves an optional field out and reports `required field '<name>' not present in XML` for a required one, as `parseString` does.
