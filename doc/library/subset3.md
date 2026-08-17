# Supported ballerina library features

Subset 3 extends the released [subset 2](subset2.md) with stream-based file read/write additions and byte channels in the `io` module, building on the language's new `stream` type, with client-side response data binding for the `http` module, and with the new `uuid` module.

## [io](https://github.com/ballerina-platform/module-ballerina-io/blob/master/docs/spec/spec.md)

Subset 2 covered console printing and whole-file I/O (string, lines, bytes,
JSON, XML). Subset 3 adds stream-based file reading and writing, built on the
language's new `stream` type, plus byte channels — the low-level, object-based
`ReadableByteChannel`/`WritableByteChannel` API for reading and writing bytes
incrementally, either from a file or from an in-memory byte array.

### Stream-based file read/write

| Feature | Notes |
|---|---|
| `fileReadLinesAsStream(path)` | Returns `stream<string, io:Error?>` yielding one line per `next()`; terminal carriage characters stripped, trailing empty line excluded |
| `fileReadBlocksAsStream(path, blockSize?)` | Returns `stream<io:Block, io:Error?>` yielding `byte[]` blocks of `blockSize` (default 4096); the final block may be shorter |
| `fileWriteLinesFromStream(path, lineStream, option?)` | Consumes a `stream<string, error?>`, appending `\n` after each line; `OVERWRITE`/`APPEND` supported |
| `fileWriteBlocksFromStream(path, byteStream, option?)` | Consumes a `stream<byte[], error?>`, concatenating blocks in order; `OVERWRITE`/`APPEND` supported |

`io:Block` is `readonly & byte[]`. The read-as-stream and write-from-stream functions read/write lazily (incrementally) rather than buffering the whole file; open errors surface at the `fileReadLinesAsStream`/`fileReadBlocksAsStream`/`fileWriteLinesFromStream`/`fileWriteBlocksFromStream` call, while read errors surface during a later `next()`. The write-from-stream functions widen their stream parameter's completion type to the generic `error?` (jBallerina uses `io:Error?`), so a stream held as `stream<_, error?>` — such as one bound directly from `fileReadBlocksAsStream` — can be written back; jBallerina rejects that. The returned streams are consumed with explicit `next()`/`close()` calls; iterating a stream with `foreach` or a query expression is not yet supported at the language level.

### Byte channels

| Feature | Notes |
|---|---|
| `openReadableFile(path)` | Returns `io:ReadableByteChannel\|io:Error` for streaming reads from a file |
| `openWritableFile(path, option?)` | Returns `io:WritableByteChannel\|io:Error` for streaming writes to a file; `OVERWRITE`/`APPEND` supported |
| `createReadableChannel(content)` | Returns `io:ReadableByteChannel\|io:Error` wrapping an in-memory `byte[]`, no file involved |
| `ReadableByteChannel.read(nBytes)` | Returns up to `nBytes` as `byte[]` — may return fewer bytes than requested (a single read, not a guaranteed full read); errors once the channel is exhausted |
| `ReadableByteChannel.readAll()` | Reads the remaining content of the channel to completion as `readonly & byte[]` |
| `ReadableByteChannel.blockStream(blockSize)` | Returns `stream<io:Block, io:Error?>\|io:Error` yielding `blockSize` `byte[]` blocks per `next()`; the final block may be shorter |
| `ReadableByteChannel.base64Encode()` | Base64-encodes the channel's remaining content into a new in-memory `io:ReadableByteChannel`; the source channel is drained but stays open |
| `ReadableByteChannel.base64Decode()` | Base64-decodes the channel's remaining content into a new in-memory `io:ReadableByteChannel`; final padding is optional, and malformed input panics |
| `ReadableByteChannel.close()` | Releases the channel's underlying resources; a second `close()` call errors |
| `WritableByteChannel.write(content, offset)` | Writes `content[offset:]`; `offset` is an index into `content`, not a file seek offset; returns the number of bytes written |
| `WritableByteChannel.close()` | Releases the channel's underlying resources; a second `close()` call errors |

CSV/record channels are out of scope for this subset and remain
`Not Yet Supported`.

### Data channels

| Feature | Notes |
|---|---|
| `new ReadableDataChannel(byteChannel, byteOrder?)` / `new WritableDataChannel(byteChannel, byteOrder?)` | Wrap a byte channel for binary-encoded data; `io:ByteOrder` is `BIG_ENDIAN` (default) or `LITTLE_ENDIAN` |
| `readInt16()` / `readInt32()` / `readInt64()` and `writeInt16(value)` / `writeInt32(value)` / `writeInt64(value)` | Fixed-width signed integers in the channel's byte order |
| `readFloat32()` / `readFloat64()` and `writeFloat32(value)` / `writeFloat64(value)` | IEEE 754 floats in the channel's byte order |
| `readBool()` / `writeBool(value)` | A single byte; `1` reads back as `true` |
| `readString(nBytes, encoding)` / `writeString(value, encoding)` | Reads/writes a string as `nBytes` bytes decoded/encoded with the given charset |
| `readVarInt()` / `writeVarInt(value)` | Variable-length integers in jBallerina's 7-bit-group wire format; the full `int` range round-trips (jBallerina itself breaks beyond 8 encoded bytes) |
| `close()` | Closes the data channel and the wrapped byte channel; a second `close()` call errors |

### Character channels

| Feature | Notes |
|---|---|
| `new ReadableCharacterChannel(byteChannel, charset)` | Wraps an `io:ReadableByteChannel`, decoding bytes with the given charset (e.g. `UTF-8`, `ISO-8859-1`); an unsupported charset panics |
| `ReadableCharacterChannel.read(numberOfChars)` | Reads up to `numberOfChars` characters; reading past the end returns an `io:Error` |
| `ReadableCharacterChannel.readString()` / `readAllLines()` | Reads the remaining content as a single string (lines joined with `\n`) or as a string array |
| `ReadableCharacterChannel.readJson()` / `readXml()` | Parses the remaining content as JSON / XML |
| `ReadableCharacterChannel.readProperty(key, defaultValue?)` / `readAllProperties()` | Reads `.properties`-format content (comments, `=`/`:`/whitespace separators, line continuations, and `\uXXXX` escapes) |
| `ReadableCharacterChannel.lineStream()` | Returns `stream<string, io:Error?>` yielding one line per `next()` |
| `ReadableCharacterChannel.close()` | Closes the character channel and the wrapped byte channel; a second `close()` call errors |
| `new WritableCharacterChannel(byteChannel, charset)` | Wraps an `io:WritableByteChannel`, encoding characters with the given charset; an unsupported charset panics |
| `WritableCharacterChannel.write(content, startOffset)` | Writes `content` from the given character offset; returns the number of bytes written |
| `WritableCharacterChannel.writeLine(content)` | Writes `content` followed by `\n` |
| `WritableCharacterChannel.writeJson(content)` / `writeXml(content, xmlDoctype?)` | Writes JSON / XML content; `writeXml` accepts an optional `io:XmlDoctype` whose `<!DOCTYPE ...>` line is emitted before the content |
| `WritableCharacterChannel.writeProperties(properties, comment)` | Writes a `map<string>` in `.properties` format with a leading comment and timestamp header |
| `WritableCharacterChannel.close()` | Closes the character channel and the wrapped byte channel; a second `close()` call errors |

The `LineStream` and `BlockStream` public helper classes are not declared;
`lineStream()` and `blockStream()` return plain stream values instead.

## [http](https://github.com/ballerina-platform/module-ballerina-http/blob/master/docs/spec/spec.md)

The client remote methods now bind the response payload directly to the contextually expected type instead of only returning an `http:Response`.

### Client — response data binding

Every remote method except `head` takes a trailing `targetType` parameter with an inferred typedesc default:

```ballerina
remote isolated function get(string path, map<string|string[]>? headers = (),
        TargetType targetType = <>) returns targetType|error;
```

`TargetType` is `typedesc<http:Response|anydata>`. The target is normally inferred from the contextually expected type, so a plain assignment is enough:

```ballerina
http:Client c = check new ("https://example.com");

Person p = check c->get("/person");        // binds the JSON body to a record
string text = check c->get("/greeting");   // binds a text/plain body
http:Response r = check c->get("/raw");    // no binding — the raw response
```

`var` provides no contextually expected type, so the target must be passed explicitly in that position:

```ballerina
var p = check c->get("/person", targetType = Person);
```

| Feature | Notes |
|---|---|
| `http:Response` target | Returned untouched, including for 4xx and 5xx responses. Any union containing `http:Response` behaves the same way |
| Status code mapping | With any other target, a 4xx or 5xx response returns an `error` whose message is the status code's reason phrase, or `status code <code>` when the code has no registered phrase; 1xx, 2xx, and 3xx responses are bound normally |
| `()` target | The payload is read and discarded |
| Nilable targets | An absent (empty) payload binds to `()` |

The builder is selected from the response `Content-Type`, matching jBallerina's media-type patterns. When the header is absent or unrecognised, the target type alone selects the builder.

| Content-Type | Supported target types |
|---|---|
| `application/json`, `text/json`, and `+json` / `.json` / `-json` suffixes | `json`, `map<json>`, records, record arrays, arrays, maps, and scalars (`int`, `float`, `decimal`, `boolean`, `string`), plus their nilable forms |
| `text/plain` | `string`, `byte[]`, and their nilable forms |
| `application/octet-stream` | `byte[]` and `byte[]?` |
| `application/x-www-form-urlencoded` | `map<string>`, `string`, and their nilable forms; repeated keys keep the last value |
| absent or unrecognised | `string`, `byte[]`, and their nilable forms are read directly; every other target is parsed as JSON |

A target that does not fit the response media type returns an `error` — for example a record target for a `text/plain` response. JSON conversion uses the same routine as `lang.value:fromJsonWithType`.

A target may also be strictly narrower than the type its builder produces — an enum or a singleton where the builder yields `string`, a closed all-string record where it yields `map<string>`, a tuple or a fixed-length array where it yields `byte[]`. The built payload is converted to that target with the same routine, so a body outside the narrower type returns an `error` rather than a value outside its declared type:

```ballerina
enum Colour { RED = "red", GREEN = "green" }

Colour c1 = check c->get("/colour");   // text/plain "red" — binds
Colour|error c2 = c->get("/text");     // text/plain "hello" — error
Colour? c3 = check c->get("/empty");   // text/plain "" — ()
```

The nilable form of such a target binds the same way: an absent body gives `()`, and a body that is present but does not fit the target is an `error`. Only the nilable form turns an absent body into `()` — a narrow target that is not nilable is handed the builder's empty value (`""`, `[]`, or `{}`), and rejects it unless the narrow type happens to admit it.

Not covered in this subset: `xml` targets and `application/xml` responses (not yet implemented — the runtime does have an `xml` type), `stream<http:SseEvent, error?>` targets, status code response records (`http:StatusCodeClient`, `getStatusCodeRecord()`), and the `validation` / `laxDataBinding` client configuration flags.

### Response

| Feature | Notes |
|---|---|
| `getTextPayload` | Now returns `string\|error`, matching jBallerina's signature; extraction failures (for example exceeding `responseLimits.maxEntityBodySize`) surface as an `error` instead of being returned through a `string` signature |

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
