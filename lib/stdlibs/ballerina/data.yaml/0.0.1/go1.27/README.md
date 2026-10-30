# Ballerina Data YAML Library

## Overview

The `data.yaml` module converts between YAML and Ballerina values. It parses YAML text, bytes and byte-block streams into the `anydata` subtype inferred from the call site, binding records, maps, arrays, tuples, unions and scalars with optional data projection, and writes `anydata` values as YAML in block or flow style. It supports the failsafe, JSON and core schemas, anchors and aliases, explicit tags, YAML streams of several documents, field name annotations and constraint validation. The Go Native Interpreter currently supports parsing, binding and writing, with field name annotations limited to the record type the typedesc names and without constraint validation.

## Key Functionalities

- Parse a YAML string, byte array or byte-block stream into records, maps, arrays, tuples, unions, scalars, `json` or `anydata`.
- Read block and flow collections, every scalar style, comments, anchors and aliases, explicit tags, directives and several documents.
- Bind several documents to a list or tuple, optionally matching documents to tuple members by type.
- Control data projection, nil handling, anchor redefinition and duplicate keys.
- Write a value as YAML with configurable indentation, flow style, quoting, canonical tags and document streams.

## Examples

```ballerina
import ballerina/data.yaml;
import ballerina/io;

type Server record {|
    string host;
    int port;
    string[] tags;
|};

public function main() returns error? {
    Server server = check yaml:parseString(string `host: localhost
port: 8080
tags: [api, v1]`);
    io:println(server);

    int[] documents = check yaml:parseString("--- 1\n--- 2");
    io:println(documents);

    io:println(check yaml:toYamlString({name: "demo", ports: [80, 443]}));
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
| Parse a YAML string | Supported | `parseString` |
| Parse a byte array | Supported | `parseBytes` decodes UTF-8 and replaces malformed bytes with U+FFFD, as jBallerina's default charset does. |
| Parse a byte-block stream | Supported | `parseStream` |
| Write a value as YAML | Partially Supported | `toYamlString`. The `blockLevel` option of `WriteConfig` has no effect, because its meaning is unclear: applying the `ballerina/yaml` meaning with its default of `1` would write every nested collection in flow style ([ballerina-library#9417](https://github.com/ballerina-platform/ballerina-library/issues/9417)). |
| Block and flow collections, scalar styles, comments and documents | Supported | |
| Anchors and aliases | Supported | `allowAnchorRedefinition`. Anchors are scoped to their document. |
| Explicit tags | Supported | As in jBallerina, only `!!str`, `!!int`, `!!float`, `!!bool`, `!!null`, `!!seq` and `!!map` are accepted, and the failsafe schema accepts none. |
| Directives | Supported | As in jBallerina, a `%TAG` directive may only name the `tag:yaml.org,2002:` prefix. |
| Explicit keys | Supported | As in jBallerina, a key must be a scalar. |
| Schemas | Supported | `FAILSAFE_SCHEMA`, `JSON_SCHEMA`, `CORE_SCHEMA`. As in jBallerina, plain scalars are converted by the expected type, and the schema decides which plain words a string may not be. Quoted, block and `!!str` scalars are strings. |
| Record targets | Partially Supported | Open, closed, optional, nilable and rest fields are supported. Field default values are not filled in, so a defaulted field without a key is reported as missing (ballerina-nutcracker/ballerina#1378), and declared fields follow the key order rather than the declaration order (ballerina-nutcracker/ballerina#1381). |
| Map, array, tuple, json and anydata targets | Supported | |
| Union targets | Partially Supported | Plain scalars try the members in jBallerina's priority order, and quoted, block and `!!str` scalars try `string` first. Collections try the members in a fixed order instead of the declaration order, which native code cannot see (ballerina-nutcracker/ballerina#1380). |
| Read-only targets | Supported | |
| Several documents | Supported | A list, tuple, `json`, `anydata` or union with such a member binds every document; any other type binds the first. |
| Data projection options | Supported | `allowDataProjection`, `nilAsOptionalField`, `absentAsNilableType`, `enableYamlStreamReorder` |
| Duplicate keys | Supported | `allowMapEntryRedefinition` |
| Field name annotation | Partially Supported | `@yaml:Name` renames fields of the record type a typedesc names, such as `yaml:parseString(s, {}, Person)`, and other annotations are ignored. An inferred typedesc, and records nested in the target, carry no field annotations (ballerina-nutcracker/ballerina#1136, ballerina-nutcracker/ballerina#1379). |
| Constraint validation | Not Yet Supported | `enableConstraintValidation` is accepted and ignored until `ballerina/constraint` is ported (ballerina-nutcracker/ballerina#732). |
| Compile-time validation of target types | Not Yet Supported | The jBallerina compiler plugin diagnostics are not emitted. An inferred target type that is not a subtype of `anydata` is not rejected at compile time (ballerina-nutcracker/ballerina#1388). |
| Type names in error messages | Partially Supported | Types are shown in their structural form, such as `{\| int... \|}` for `map<int>`, because declared type names are not available at runtime (ballerina-nutcracker/ballerina#1354). |
| Syntax error messages | Partially Supported | Errors use jBallerina's messages and line and column format, but the position, and for some malformed inputs the message, can differ because the parser is a separate implementation. |
| Module error type | Supported | `yaml:Error` is a `distinct` error, as in jBallerina. |

### Notable Behavioural Changes

- **Errors are returned instead of panics.** jBallerina panics with a Java exception for many inputs, such as non-numeric text for an `int`, `float`, `decimal` or `byte` target, a malformed `!!int` or `!!float` value, a `!!str` tag on an empty node, a mapping bound to a list type, two empty documents, a negative `indentationPolicy` and `()` in a value passed to `toYamlString`, and with a `TypeCastError` for a boolean or null scalar such as `true` in a `string[]` target; the Go-native version returns a `yaml:Error` for each, and writes `()` as `null`.
- **Scalars fold and chomp as YAML 1.2 specifies.** jBallerina joins the lines of a double-quoted scalar without a space, drops empty lines from single-quoted, plain and folded scalars, drops leading empty lines and kept trailing lines of block scalars, and misplaces content with an indentation indicator inside a collection; the Go-native version applies the YAML 1.2 rules, so `"x\n  y"` is `x y` rather than `xy`. A block scalar that ends the input still ends with a line break, as in jBallerina.
- **The escape `\\` is one backslash.** jBallerina turns `\\` in a double-quoted scalar into two backslashes; the Go-native version produces one.
- **Values at the root of a list target keep their text.** When the target can hold several documents, jBallerina first reads each value as JSON and converts that JSON's text, so `[007, 1.50]` binds to `string[]` as `["7", "1.5"]` and decimals lose precision; the Go-native version converts the YAML text directly, giving `["007", "1.50"]`.
- **toYamlString quotes strings that would not read back unchanged.** jBallerina writes strings as they are, replaces a newline with the letter `n`, and never escapes embedded quotes, so `""`, `true`, `123`, `a: b` or `- x` come back as different values; the Go-native version writes such strings double-quoted with escapes, and `''` for a single quote inside single quotes. It also quotes the YAML 1.2 core schema number forms such as `.inf`, `.NaN`, `0x1F` and `0o17`, which jBallerina leaves bare and other YAML 1.2 readers take as numbers.
- **toYamlString quotes the document-end marker and flow indicators in flow style.** jBallerina writes the string `...` unquoted, so it reads back as an empty document, and with `flowStyle` writes strings such as `a,b` or `c]d` unquoted inside a flow collection, so `[a,b]` reads back as two values or fails to parse; the Go-native version quotes them. See [ballerina-library#9394](https://github.com/ballerina-platform/ballerina-library/issues/9394) and [ballerina-library#9395](https://github.com/ballerina-platform/ballerina-library/issues/9395).
- **The schema option decides which strings toYamlString quotes.** jBallerina ignores the `schema` field of `WriteConfig`; the Go-native version quotes a string that the schema would read as null, a boolean or a number, so with `CORE_SCHEMA`, the default, `true`, `True`, `~`, `1` and `0x1F` are quoted, with `JSON_SCHEMA` only `null`, `true`, `false` and JSON numbers such as `1` and `1.5` are, and with `FAILSAFE_SCHEMA`, where every plain scalar is a string, none of them is. See [ballerina-library#9417](https://github.com/ballerina-platform/ballerina-library/issues/9417).
- **Only `@yaml:Name` renames a record field.** jBallerina renames a field with any annotation whose name ends in `Name`, so a field with only an annotation such as `@FieldName {value: "other"}` from another module binds the key `other`; the Go-native version ignores such annotations and binds the key `name`. See [ballerina-library#9414](https://github.com/ballerina-platform/ballerina-library/issues/9414).
- **Zero indentation is rejected for nested block collections.** With `indentationPolicy: 0`, jBallerina writes a nested mapping or sequence at its parent's indentation, so `{parent: {child: 1}}` reads back as `{"parent":null,"child":1}`; the Go-native version returns `indentationPolicy must be greater than 0 to write a nested block collection`. Output without such nesting is unchanged. See [ballerina-library#9396](https://github.com/ballerina-platform/ballerina-library/issues/9396).
- **toYamlString writes empty collections in flow style.** jBallerina writes an empty list as a bare `-` and an empty mapping as nothing, so `{a: [], b: {}}` reads back as `{"a":[null],"b":null}` and `{}` as an empty string that does not parse; the Go-native version writes `[]` and `{}`.
- **Resource limits return an error.** jBallerina has no limit on nesting depth or alias expansion and recurses into a value that contains itself, failing with a Java stack overflow or running out of memory; the Go-native version returns `exceeded max depth of 10000` for collections nested more than 10000 deep, `document contains excessive aliasing` when aliases expand to far more nodes than the document holds, as go-yaml does, and `cannot write a value that has a cyclic reference` from `toYamlString`. It also rejects an `indentationPolicy` greater than 1024.
- **Anchors are scoped to their document.** jBallerina keeps anchors across the documents of a stream, so an alias can refer to an earlier document and, with `allowAnchorRedefinition: false`, an anchor name cannot be reused in a later document; the Go-native version follows the YAML specification and starts each document with no anchors.
- **Quoted, block and `!!str` scalars are strings.** jBallerina converts the text of every scalar by the expected type alone, so with a `json`, `anydata` or union target that contains `string` it reads `'1'` as `1`, `"true"` as `true` and the literal block `|\n  12` as `12.0`, rejects a quoted `"true"` or `"null"` for a `string` target, so a string written quoted, such as `"1"` from `toYamlString`, does not read back as a string; the Go-native version follows YAML 1.2, which resolves every non-plain scalar and every `!!str` scalar to a string, so such scalars bind to `string` first and are accepted by `string` targets. Explicit `int`, `float`, `decimal`, `boolean` and `()` targets still convert the text, so `'1'` still binds to `int`. See [ballerina-library#9424](https://github.com/ballerina-platform/ballerina-library/issues/9424).
- **Tagged values must fit the expected type.** jBallerina stores a value with an `!!int`, `!!float`, `!!bool` or `!!null` tag without checking it, so a `map<int>` can hold `true`; the Go-native version reports the value as incompatible. It also accepts `!!seq` and `!!map` on matching collections, which jBallerina rejects.
- **Integer subtype ranges are checked on the full value.** jBallerina checks the `int:Signed8`, `int:Signed16`, `int:Unsigned8` and `int:Unsigned16` ranges on the value truncated to 32 bits; the Go-native version rejects any value outside the range.
- **A map or rest field value that does not convert is dropped.** jBallerina drops such a value unless the conversion fails with a number format exception, in which case it panics; the Go-native version drops every value that does not convert.
- **An empty value for a non-nilable type is an error.** jBallerina stores `()` for `a:` whatever the field type; the Go-native version reports it as incompatible unless the type allows nil. With `nilAsOptionalField`, an empty value for an optional field leaves the field absent, as `null` does. jBallerina does the same for `age:` followed by a line break, but panics on `{age: }` and reports a syntax error when `age:` ends the input.
- **An error from the source stream is returned.** jBallerina's `parseStream` ignores an error from the byte-block stream and parses the bytes read so far; the Go-native version returns the stream's error message prefixed with `Error occurred while reading the stream:`.
- **A leading byte-order mark is ignored.** jBallerina rejects input that starts with U+FEFF; the Go-native version skips it.
- **A tab before the first key is rejected.** jBallerina reads `"\thost: tab"` as `{"host":"tab"}`; the Go-native version follows the YAML specification, which does not allow tabs in indentation, and reports `cannot have tab as an indentation`.
- **More valid YAML is accepted.** jBallerina rejects an anchor on a flow collection used as a mapping value, a second explicit `?` key in a mapping, an empty mapping value at the end of the input, and anonymous read-only record types; the Go-native version accepts them. A merge key `<<` is read as an ordinary key rather than causing an error, since neither version merges mappings.
- **Tuple reordering matches documents by conversion.** With `enableYamlStreamReorder`, jBallerina matches a document to a tuple member by comparing Java type hashes; the Go-native version gives each document to the first unfilled member it converts to, which gives the same result for record and basic member types.
- **An array target with data projection disabled binds normally.** jBallerina reports `array size is not compatible with the expected size` for an open array at the root, such as `int[]` from `[1]`, when `allowDataProjection` is `false`; closed arrays and tuples bind in both. The Go-native version reports it only when the list has more members than a closed list type allows.
- **`isStream` on a non-list value writes nested sequences normally.** jBallerina drops the `-` of a nested block sequence, or panics, when `isStream` is set and the value is not a list; the Go-native version writes the value as it would without `isStream`.
