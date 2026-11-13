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
| Converting an XML value to a record type | Partially Supported | `parseAsType`. A record field with a default value is still required, because native code cannot read field defaults (ballerina-nutcracker/ballerina#1378). Union members are tried in a fixed order rather than in declaration order, because union types are unordered and native code cannot see the order they were declared in. Record values keep XML order rather than declaration order (#1381), and a field whose name needs escaping, such as `\#content`, keeps the backslash in its key (#1351). A field whose type is a union of a record and a simple type, such as `Inner\|int`, gives `{}` for text content when the record member has no required field, because the record member is tried first; jBallerina returns the same `{}`, and fails with a `NullPointerException` message when the record member has a required field ([ballerina-library#9426](https://github.com/ballerina-platform/ballerina-library/issues/9426)). `xml:fromString` does not accept an XML declaration (#1414), and `xml` templates cannot contain CDATA (#1412). |
| Parsing XML text into a record type | Partially Supported | `parseString`. Same field-default, union-order, record-and-simple-type union, field-order and escaped-name gaps as `parseAsType`. |
| Parsing XML bytes into a record type | Partially Supported | `parseBytes`. Same gaps as `parseString`. |
| Parsing an XML byte-block stream into a record type | Partially Supported | `parseStream`. Same gaps as `parseString`. |
| Projection and namespace options | Supported | `SourceOptions` with `allowDataProjection`, `useSemanticEquality`, `attributePrefix` and `textFieldName`. |
| Name, namespace and attribute annotations | Partially Supported | `@xmldata:Name`, `@xmldata:Namespace`, `@xmldata:Attribute`, `NameConfig` and `NamespaceConfig`. Applied only when the target record type is passed explicitly, as in `parseString(s, {}, Item)`, and only to that type and its own fields: annotations are not visible through an inferred typedesc or a nested type (#1136). |
| Validating XML against a record type | Partially Supported | `validate` with a record typedesc. Same gaps as `parseAsType`. |
| Validating XML against an XSD file | Not Yet Supported | `validate` with a file path returns an `Error`: no XSD validator is available to Go without cgo. |
| Error messages | Partially Supported | Message texts match jBallerina. A record type is printed structurally instead of by its declared name (#1354), and other anonymous types are printed in the interpreter's type notation, such as `(nil\|boolean\|int)` for `(int\|boolean)?`. Malformed XML is described by the Go XML tokenizer; see Notable Behavioural Changes. |
| Module error type | Supported | `xmldata:Error` is a `distinct` error. An inferred target type that wrongly includes it (#1383) is ignored. |
| Compile-time target type validation | Partially Supported | The jBallerina compiler plugin is not ported, so of its diagnostics `XML_ERROR_201`–`XML_ERROR_211` only the duplicate-name check (`XML_ERROR_201`) is applied, when the type is used as a target; see Notable Behavioural Changes. |
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
- **A byte-block stream error is returned.** jBallerina's `parseStream` treats an error from the source stream as the end of the input, so a document that is complete before the error converts successfully, one that is cut short is reported as malformed XML, and the error itself is never reported; the Go-native version returns the stream's error as an `xmldata:Error` and closes the stream (ballerina-platform/ballerina-library#9695).
- **An absent fixed-length array field is checked like any other field.** When a record has a fixed-length array field, such as `Person[2] p?;`, and the XML has no element for it, jBallerina's `parseAsType` fails with an internal `NullPointerException` message whether or not the field is optional (ballerina-platform/ballerina-library#9431); the Go-native version leaves an optional field out and reports `required field '<name>' not present in XML` for a required one, as `parseString` does.
- **A scalar rest field never holds a map.** When an element with attributes maps to an open-record rest field whose type cannot hold a map, such as `int` in `record {| int...; |}`, jBallerina stores the attributes and the text as a map such as `{"x":1,"#content":5}` and returns a record value that does not belong to its type; `parseString` does the same for an empty or nested element, and `parseAsType` panics with an `InherentTypeViolation` for an `int[]` rest field (ballerina-platform/ballerina-library#6907). The Go-native version drops those attributes as it does for a declared `int` field, so `<a x="1">5</a>` gives `{"a":5}`, and returns `field '<name>' cannot be converted into the type '<type>'` for an element without text.
- **A field element with the same name as its record's element does not end the record.** When a record's element contains a field element of the same name, such as `<item><item>text</item><name>x</name></item>` for `record {| string item?; string name; |}`, jBallerina's `parseString` treats the inner end tag as the end of the record and returns `required field 'name' not present in XML`, and its `parseAsType` returns `incompatible type expected anydata or json` when the record is an open record's rest type ([ballerina-library#9503](https://github.com/ballerina-platform/ballerina-library/issues/9503)); the Go-native version ends the record at its own end tag.
- **Text split by several comments or by CDATA sections is joined.** jBallerina's `parseString` skips only one comment between two pieces of text and stores a CDATA section apart from the text next to it, so `x<!--a--><!--b-->y` gives `y`, `x<![CDATA[y]]>z` gives `z` and two adjacent CDATA sections keep only the last ([ballerina-library#9504](https://github.com/ballerina-platform/ballerina-library/issues/9504)); the Go-native version joins all the character data of the element, as `parseAsType` does.
- **Projection skips the whole content of an unmapped element.** With `allowDataProjection`, jBallerina's `parseString` drops an unmapped element but still reads its children, so `<r><ignored><name>bad</name></ignored></r>` binds `bad` to the enclosing record's `name` field, and a later `<name>` returns `expected 'string' value for the field 'name' found 'array' value` ([ballerina-library#9505](https://github.com/ballerina-platform/ballerina-library/issues/9505)); the Go-native version skips the element with everything inside it, as `parseAsType` does.
- **An attribute value that does not convert to its field type is an error.** For a field declared with `@xmldata:Attribute`, jBallerina drops an attribute whose value does not convert to the field type, so `<r id="not-an-int"/>` gives `{"id":0}` for a required `int id` and `{}` for an optional one ([ballerina-library#9506](https://github.com/ballerina-platform/ballerina-library/issues/9506)); the Go-native version returns the conversion error, such as `'string' value 'not-an-int' cannot be converted to 'int'`.
- **An `xml:Text` field never holds an element.** jBallerina's `parseAsType` stores the field's element itself, such as `<t>plain</t>`, in an `xml:Text` field and returns a record value that does not belong to its type ([ballerina-library#9507](https://github.com/ballerina-platform/ballerina-library/issues/9507)); the Go-native version returns `'xml' value '<t>plain</t>' cannot be converted to 'xml:Text'`, as both implementations do for an `xml:Text[]` field.
- **A renamed `json` or `anydata` field keeps its field name.** For a `json`, `anydata` or `json[]` field renamed with `@xmldata:Name`, jBallerina's `parseAsType` stores the value under the element name and leaves the field `()`, such as `{"data":null,"x-data":1}`, a value that does not belong to the closed record type ([ballerina-library#9508](https://github.com/ballerina-platform/ballerina-library/issues/9508)); the Go-native version stores the value under the field name, as `parseString` does.
- **A repeated element that fits only another array member of a union is an error.** For a field such as `int[]|string[]`, jBallerina's `parseAsType` discards the values converted so far when a later element fits only the other member, so `<v>1</v><v>x</v>` gives `["x"]`, and for `int[]|string` `<v>1</v><v>z</v>` gives `"z"` ([ballerina-library#9509](https://github.com/ballerina-platform/ballerina-library/issues/9509)); the Go-native version returns `field '<name>' cannot be converted into the type '<type>'`.
- **A repeated element does not overwrite an earlier value of an array-or-scalar union.** For a field such as `int[]|string`, jBallerina's `parseAsType` lets a repeated element that fits only the scalar member replace the value of the earlier one, so `<a>z</a><a>2</a>` gives `"2"` and `z` is lost, while `parseString` reports an error ([ballerina-library#9511](https://github.com/ballerina-platform/ballerina-library/issues/9511)); the Go-native version returns the same `found 'array' value` error from both functions.
- **`int` subtype and `byte` fields are converted with a range check.** jBallerina's `parseString` rejects an element mapped to an `int` subtype such as `int:Signed8`, or to `byte`, with `invalid rest type`, its `parseAsType` rejects an `int` subtype element with `unsupported input type`, and both replace an `int` subtype attribute value with `0` ([ballerina-library#9510](https://github.com/ballerina-platform/ballerina-library/issues/9510)); the Go-native version converts the value and checks it against the type's range, so `200` for `int:Signed8` returns `'string' value '200' cannot be converted to 'lang.int:Signed8'`.
- **A recursive record target is parsed.** jBallerina's `parseString` fails with `failed to parse xml: 'null'` when the target record refers to itself, as in `type Node record { Node a?; }`, even for `<a><a></a></a>`, while an equivalent non-recursive record parses; the Go-native version parses into the recursive record, giving `{"a":{}}` ([ballerina-library#9635](https://github.com/ballerina-platform/ballerina-library/issues/9635)).
- **Every parse path applies the 1000-level element depth limit.** jBallerina's `parseAsType` has no element depth limit, so a 1001-level `xml` value converts and a value nested about 2000 levels or more fails with a `failed to get xml as string: ...` stack-overflow error, and its `parseString` with a target that contains a union panics with a `TypeCastError` instead of returning the depth-limit error; the Go-native version returns `failed to parse xml: Maximum Element Depth limit (1000) Exceeded` from `parseString`, `parseBytes`, `parseStream` and `parseAsType` for every target type ([ballerina-library#9643](https://github.com/ballerina-platform/ballerina-library/issues/9643)).
