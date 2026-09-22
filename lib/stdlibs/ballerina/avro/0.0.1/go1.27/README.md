# Ballerina Avro Library

## Overview

This module provides an easy way to convert Ballerina data into bytes according to an Avro schema, and to convert Avro-serialized bytes back into a specific Ballerina type.

The schema is supplied as a string when an `avro:Schema` value is created, and that one schema drives both directions. Serialization walks the schema against the given `anydata` value; deserialization decodes the payload and then binds the result to the type inferred from the call site.

The Go Native Interpreter supports the complete public surface — schema initialization, serialization, and deserialization — across every Avro type the specification maps to a Ballerina type.

## Key Functionalities

- Initialize a schema from an Avro schema definition string, including named types, namespaces, name references, and recursive schemas.
- Serialize Ballerina data into a byte array according to a schema.
- Deserialize an Avro-encoded byte array into a Ballerina type inferred from the call site, or given explicitly.
- Full Avro type coverage: null, boolean, int, long, float, double, bytes, string, record, enum, array, map, union, and fixed.

## Examples

```ballerina
import ballerina/avro;
import ballerina/io;

type Person record {|
    string name;
    int age;
|};

public function main() returns error? {
    avro:Schema schema = check new (string `{
        "type": "record", "name": "Person", "namespace": "example.avro",
        "fields": [
            {"name": "name", "type": "string"},
            {"name": "age", "type": "int"}
        ]}`);

    byte[] serialized = check schema.toAvro({name: "Ann", age: 30});
    io:println(serialized.length());

    Person person = check schema.fromAvro(serialized);
    io:println(person.name);
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
| Schema initialization from a definition string | Supported | |
| Serialization into a byte array | Supported | |
| Deserialization into an inferred target type | Supported | |
| Deserialization into an explicitly named target type | Supported | Pass the type as the `targetType` named argument when there is no contextually expected type. |
| Null, boolean, int, long, float, double | Supported | |
| Bytes and fixed | Supported | Both map to `byte[]`; a `fixed` value must carry exactly the declared number of bytes; a `fixed` schema must declare a size greater than zero. |
| String | Supported | |
| Record | Supported | |
| Enum | Supported | |
| Array | Supported | |
| Map | Supported | |
| Union | Supported | Branch selection prefers the branch matching the value's natural Avro type — see Notable Behavioural Changes. |
| Named type references and namespaces | Supported | |
| Recursive schemas | Supported | |
| Module error type | Partially Supported | `avro:Error` is declared, but as a plain `error` alias; the `distinct` type descriptor is not yet supported. |
| Readonly and intersection target types | Supported | `readonly & T` binds and freezes the decoded value, for a record, an array, and a tuple. |

### Notable Behavioural Changes

- **An unparseable schema returns an error instead of panicking.** jBallerina lets the underlying schema-parser exception escape `init`, so `new avro:Schema("not-a-schema")` panics even though `init` declares `returns Error?`; the Go-native version returns an `avro:Error` with the message `Avro schema parsing error` — this is what the module specification describes, and it stays inside the declared signature.
- **Union branch selection prefers the natural branch and applies one rule everywhere.** jBallerina uses two different rules: a union inside a record field goes through a tag table where a `float` or `double` branch also claims `int` and `decimal` values, while a top-level union bypasses that table entirely and is resolved by the underlying Avro library — which rejects `string`, `bytes`, `record`, `enum`, `array`, and `fixed` branches outright. The Go-native version applies one rule at every position: the first branch whose Avro type is the value's natural encoding wins, and only if no branch matches does the widening tag table apply. Consequently a top-level `["null","string"]` (and every other combination jBallerina rejects) works, and `["double","long"]` given an `int` selects the lossless `long` branch where a jBallerina record field would select `double`. A `bytes`/`fixed` branch and an `array` branch — or a `record` branch and a `map` branch — sharing a union are told apart by the value's own inherent type rather than by declaration order, since a Ballerina `byte[]` and a `T[]` both reach the union as the same list representation, and a record and a `map<T>` both reach it as the same mapping representation.
- **A `float` is not narrowed into an `int` or `long` schema.** jBallerina converts nothing in this case — the value falls through to Apache Avro's `GenericDatumWriter`, which casts to `java.lang.Number` and truncates toward zero, so `3.9` is stored as `3`. The Go-native version returns an `avro:Error` instead of writing a payload that does not represent the value. Conversions jBallerina performs deliberately are kept as they are: an `int` narrows into an `int` schema with the same wrapping `Long.intValue()` semantics, an `int` widens into a `float` or `double` schema, a `decimal` widens into a `double` schema, and a `string` schema stringifies whatever it is given.
- **A `bytes`/`fixed` schema requires a value whose own type is `byte[]`.** An `int[]` is rejected even when every value happens to fit in a byte (0-255) — including a bare list literal such as `schema.toAvro([1, 2, 3])`, which carries no `byte[]` type of its own since `toAvro` takes `anydata`; declare the value as `byte[]` first. jBallerina agrees this should be an error but does not implement it as one: passing a plain `int[]`, in range or not, throws an uncaught `NullPointerException` instead of returning an error.
- **Avro map keys are encoded in insertion order.** jBallerina iterates a Java `HashMap`, so the key order in an encoded Avro map is unspecified; the Go-native version writes keys in the Ballerina value's insertion order — the Avro encoding does not constrain map key order and readers are insensitive to it.
- **A `fixed` schema with `"size": 0` is rejected.** jBallerina's underlying Avro library accepts a zero-size `fixed` type, which always encodes to zero bytes. The Go-native version's underlying codec requires a size greater than zero and rejects the schema itself with an `avro:Error` from `new`. Zero-size `fixed` types have no practical use and are not expected to appear in real schemas.
