# Ballerina Protobuf Library

## Overview

This module provides APIs to represent the protobuf "well-known" types — `google.protobuf.Any`, `Struct`, `Timestamp`, `Duration`, `Empty`, and the scalar wrapper types (`BoolValue`, `BytesValue`, `FloatValue`, `Int64Value`, `StringValue`) — used by generated gRPC client/service code, plus a `Descriptor` annotation generated code attaches to message record types. The full jBallerina package also supports resolving arbitrary user-defined message records through that annotation. The Go Native Interpreter currently supports packing and unpacking values of the well-known types through `protobuf.types.any:pack`/`unpack`, and all of the plain type declarations across the `protobuf.types.*` submodules; descriptor-based handling of user-defined message records is not yet implemented.

## Key Functionalities

- `protobuf:Error`, `protobuf:MessageDescriptor`, and the `protobuf:Descriptor` annotation.
- `protobuf.types.any:Any`, `protobuf.types.any:pack`, and `protobuf.types.any:unpack` for `int`, `float`, `string`, `boolean`, `byte[]`, `()`, `time:Utc`, `time:Seconds`, and `map<anydata>` values.
- `protobuf.types.duration`, `protobuf.types.empty`, `protobuf.types.struct`, `protobuf.types.timestamp`, and `protobuf.types.wrappers` context record types.

## Examples

```ballerina
import ballerina/io;
import ballerina/protobuf.types.'any as pbany;

public function main() returns error? {
    pbany:Any packed = check pbany:pack(42);
    io:println(packed.typeUrl);

    int unpacked = check pbany:unpack(packed);
    io:println(unpacked);
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
| Base error type | Supported | |
| Message descriptor annotation | Partially Supported | Declared and attachable. `pack` cannot use it: it receives only the value, which carries no nominal type or annotations, and `typeof` is not implemented yet. `unpack` does receive the annotated typedesc and is not blocked by that, but descriptor handling is deferred until both directions work. |
| Any value representation | Supported | |
| Any context record types | Supported | |
| Packing a value into Any | Supported | Well-known types (scalars, `byte[]`, `()`, `time:Utc`, `time:Seconds`, `map<anydata>`) are fully supported, and `record {\|\|}` packs as `google.protobuf.Empty`. Any other message record falls back to `google.protobuf.Struct` — see the message descriptor annotation row. |
| Unpacking a value from Any | Supported | Well-known types are fully supported, including the inferred-typedesc target parameter. A type that does not match the packed value's type raises `TypeMismatchError`. |
| Any module error type | Partially Supported | Standalone `distinct error` rather than `distinct protobuf:Error`, so catching `protobuf:Error` will not catch this module's errors. Pending https://github.com/ballerina-nutcracker/ballerina/issues/777 (a module cannot import its own package's default module). |
| Type-mismatch error type | Supported | |
| Duration context record types | Supported | |
| Empty type and context record type | Supported | |
| Struct context record types | Supported | |
| Timestamp context record types | Supported | |
| Scalar wrapper context record types | Supported | |

### Notable Behavioural Changes

- **`map<anydata>` keys from an unpacked Struct are sorted alphabetically.** jBallerina's field order for a deserialized `Struct` reflects Java's map implementation; the Go-native version sorts keys alphabetically instead, since Go's map type does not preserve insertion or wire order.
