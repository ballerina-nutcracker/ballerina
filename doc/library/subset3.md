# Supported ballerina library features

Subset 3 extends the released [subset 2](subset2.md) (`crypto`, `io` file I/O,
`log`, `os`, `random`, `math.vector`, `time`, `url`, and the `http`
server/listener) with the `uuid` module.

## [uuid](https://github.com/ballerina-platform/module-ballerina-uuid/blob/master/docs/spec/spec.md)

RFC 4122 UUID generation, validation, version detection, and conversions
between string, byte array, and the `Uuid` record.

| Function | Notes |
|---|---|
| `createType1AsString` / `createType1AsRecord` | Time-based UUID (type 1); the node identifier is a random 6-byte value (RFC 4122 §4.5) rather than the host MAC address |
| `createType3AsString` / `createType3AsRecord` | Namespace + name UUID (type 3, MD5) |
| `createType4AsString` / `createType4AsRecord` | Random UUID (type 4) |
| `createType5AsString` / `createType5AsRecord` | Namespace + name UUID (type 5, SHA-1) |
| `createRandomUuid` | Alias for `createType4AsString` |
| `nilAsString` / `nilAsRecord` | The nil UUID (`00000000-0000-0000-0000-000000000000`) |
| `validate` | Tests whether a string is a well-formed UUID |
| `getVersion` | Detects the RFC version (`V1`, `V3`, `V4`, `V5`) of a UUID string |
| `toBytes` / `toString` / `toRecord` | Convert between a UUID string, `byte[]`, and the `Uuid` record; `byte[]` inputs must be exactly 16 bytes |

Predefined `NamespaceUUID` constants (`NAME_SPACE_DNS`, `NAME_SPACE_URL`,
`NAME_SPACE_OID`, `NAME_SPACE_X500`, `NAME_SPACE_NIL`) are available for type
3/5 generation. The `Uuid` record is `readonly` with `int:Unsigned32` /
`int:Unsigned16` / `int:Unsigned8` fields, matching jBallerina's contract.

`uuid:Error` is a plain `error` alias; the `distinct` error subtype is not yet
supported.
