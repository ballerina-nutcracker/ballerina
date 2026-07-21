# Ballerina UUID Library

## Overview

The `ballerina/uuid` library provides functions for generating, validating, and converting Universally Unique Identifiers (UUIDs) as defined by RFC 4122. It supports UUID versions 1, 3, 4, and 5, as well as the nil UUID, and offers conversions between string, byte array, and record representations.

## Key Functionalities

- Generate type 1 (time-based), type 3 (MD5 namespace), type 4 (random), and type 5 (SHA-1 namespace) UUIDs as strings or records.
- Validate UUID strings and detect their RFC version.
- Convert UUIDs between string, byte array, and the structured `Uuid` record.
- Produce the nil UUID as a string or record.
- Use pre-defined namespace UUID constants (DNS, URL, OID, X.500, nil).

## Examples

```ballerina
import ballerina/io;
import ballerina/uuid;

public function main() returns error? {
    // Generate a type 4 (random) UUID
    string v4 = uuid:createType4AsString();
    io:println(uuid:validate(v4));     // true

    uuid:Version ver = check uuid:getVersion(v4);
    io:println(ver == uuid:V4);        // true

    // Deterministic type 3 UUID (MD5 namespace)
    string v3 = check uuid:createType3AsString(uuid:NAME_SPACE_DNS, "ballerina.io");
    io:println(uuid:validate(v3));     // true

    // Round-trip: string → record → string
    uuid:Uuid rec = check uuid:toRecord(v4);
    string back = check uuid:toString(rec);
    io:println(back == v4);            // true
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
| UUID type 1 generation — string | Supported | `createType1AsString()` |
| UUID type 1 generation — record | Supported | `createType1AsRecord()` |
| UUID type 3 generation — string (MD5 namespace) | Supported | `createType3AsString(namespace, name)` |
| UUID type 3 generation — record (MD5 namespace) | Supported | `createType3AsRecord(namespace, name)` |
| UUID type 4 generation — string | Supported | `createType4AsString()` |
| UUID type 4 generation — record | Supported | `createType4AsRecord()` |
| UUID type 5 generation — string (SHA-1 namespace) | Supported | `createType5AsString(namespace, name)` |
| UUID type 5 generation — record (SHA-1 namespace) | Supported | `createType5AsRecord(namespace, name)` |
| Random UUID generation | Supported | `createRandomUuid()` — alias for `createType4AsString()` |
| Nil UUID — string | Supported | `nilAsString()` |
| Nil UUID — record | Supported | `nilAsRecord()` |
| UUID validation | Supported | `validate(uuid)` |
| UUID version detection | Supported | `getVersion(uuid)` — supports V1, V3, V4, V5 |
| Conversion — UUID to bytes | Supported | `toBytes(string\|Uuid)` |
| Conversion — bytes or record to string | Supported | `toString(byte[]\|Uuid)` |
| Conversion — string or bytes to record | Supported | `toRecord(string\|byte[])` |
| UUID record type | Supported | `Uuid` record with all six fields (`timeLow`, `timeMid`, `timeHiAndVersion`, `clockSeqHiAndReserved`, `clockSeqLo`, `node`) |
| Version enum | Supported | `Version` with values `V1`, `V3`, `V4`, `V5` |
| Predefined namespace UUID constants | Supported | `NamespaceUUID` enum: `NAME_SPACE_DNS`, `NAME_SPACE_URL`, `NAME_SPACE_OID`, `NAME_SPACE_X500`, `NAME_SPACE_NIL` |
| Module-level error type | Partially Supported | `uuid:Error` is a plain `error` alias; `distinct` type descriptor not yet supported |

### Notable Behavioural Changes

- **Type 1 UUID node identifier — random bytes instead of MAC address.** jBallerina uses the MAC address of the host machine as the node identifier in type 1 UUIDs; the Go-native version generates a random 6-byte node ID per RFC 4122 §4.5 for portability and privacy. The UUID is still valid and passes `validate()`.
