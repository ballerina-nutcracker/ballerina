# Supported ballerina library features

Subset 3 extends the released [subset 2](subset2.md) with the `avro` module — Avro binary serialization and deserialization driven by an Avro schema string — plus stream-based file read/write additions and byte channels in the `io` module, building on the language's new `stream` type; client-side response data binding, XML payloads, `anydata` resource returns, and byte-stream and Server-Sent Event bodies for the `http` module; the new `ballerina/protobuf` package, providing the protobuf well-known types (`Any`, `Struct`, `Timestamp`, `Duration`, `Empty`, and the scalar wrapper types) used by generated gRPC client/service code; the new `file` module; and the new `ballerina/uuid` package, providing RFC 4122 UUID generation, validation, and conversion. It also adds the new `ballerina/ldap` package, a client for reading and modifying entries on an LDAP directory server, the new `ballerina/udp` package, providing connectionless and connection-oriented UDP clients and a UDP listener, the new `ballerina/mime` module for MIME media types, entities, and multipart bodies, the new `ballerina/task` package, providing one-time and recurring job scheduling, job management, and task services on a listener, and the new `ballerina/mqtt` package, an MQTT v5 client and listener for publishing and receiving messages through a broker.

## [avro](https://github.com/ballerina-platform/module-ballerina-avro/blob/master/docs/spec/spec.md)

Types: `avro:Schema` (class), `avro:Error`.

| Method | Notes |
|---|---|
| `new avro:Schema(schema)` | Parse an Avro schema definition string; returns `avro:Error?`. Named types, namespaces, name references and recursive schemas are supported. An unparseable schema returns an error rather than panicking |
| `Schema.toAvro(data)` | Serialize `anydata` into the Avro binary encoding; returns `byte[]\|avro:Error` |
| `Schema.fromAvro(data, targetType?)` | Deserialize `byte[]` into the type inferred from the call site, or the explicitly named `targetType`; returns `targetType\|avro:Error` |

### Avro-to-Ballerina type mapping

Every mapping the module specification defines is supported in both directions.

| Avro type | Ballerina type | Notes |
|---|---|---|
| `null` | `()` | Encodes to zero bytes |
| `boolean` | `boolean` | |
| `int`, `long` | `int` | Writing to an `int` schema narrows the Ballerina `int` value to 32 bits with wrapping, matching jBallerina; a `float` is rejected rather than truncated |
| `float`, `double` | `float` | Both also accept an `int`; `double` also accepts a `decimal` |
| `bytes`, `fixed` | `byte[]` | A `fixed` value must carry exactly the declared size |
| `string` | `string` | A value of any other type is stringified, matching jBallerina |
| `record` | `record` | Field order follows the schema; fields the schema does not declare are ignored |
| `enum` | `enum` | Also decodes to a plain `string` |
| `array` | array | Including arrays of records and nested arrays |
| `map` | `map` | Including maps of records, arrays, and nested maps |
| `union` | the corresponding union | The first branch matching the value's natural Avro type wins; a widening pass follows if none does |

### Target-type binding

`fromAvro` infers `targetType` from the contextually expected type, so `Person p = check schema.fromAvro(data);` binds the decoded payload to `Person`. Without a contextually expected type the compiler reports `cannot infer typedesc argument for parameter 'targetType'` — pass `targetType = Person` explicitly in that case.

Records, enums, tuples, singletons, `map<T>`, `T[]`, `json`, `map<json>`, `anydata` and nilable forms of all of these are accepted as targets, along with numeric widening from `int` to `float` and `decimal`. A `readonly &` intersection of any of these is accepted too and binds to a frozen value.
## [io](https://github.com/ballerina-platform/module-ballerina-io/blob/master/docs/spec/spec.md)

Subset 2 covered console printing and whole-file I/O (string, lines, bytes, JSON, XML). Subset 3 adds stream-based file reading and writing, built on the language's new `stream` type, plus byte channels — the low-level, object-based `ReadableByteChannel`/`WritableByteChannel` API for reading and writing bytes incrementally, either from a file or from an in-memory byte array.

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

CSV/record channels are out of scope for this subset and remain `Not Yet Supported`.

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

The `LineStream` and `BlockStream` public helper classes are not declared; `lineStream()` and `blockStream()` return plain stream values instead.

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
| `application/xml`, `text/xml`, and `+xml` / `.xml` / `-xml` suffixes | `xml` and `xml?`, any union admitting `xml`, and narrower subtypes such as `xml:Element` when the parsed value inhabits them |
| absent or unrecognised | `string`, `xml`, `byte[]`, and their nilable forms are read directly (in that order); every other target is parsed as JSON |

A target that does not fit the response media type returns an `error` — for example a record target for a `text/plain` response. JSON conversion uses the same routine as `lang.value:fromJsonWithType`.

A target may also be strictly narrower than the type its builder produces — an enum or a singleton where the builder yields `string`, a closed all-string record where it yields `map<string>`, a tuple or a fixed-length array where it yields `byte[]`. The built payload is converted to that target with the same routine, so a body outside the narrower type returns an `error` rather than a value outside its declared type:

```ballerina
enum Colour { RED = "red", GREEN = "green" }

Colour c1 = check c->get("/colour");   // text/plain "red" — binds
Colour|error c2 = c->get("/text");     // text/plain "hello" — error
Colour? c3 = check c->get("/empty");   // text/plain "" — ()
```

The nilable form of such a target binds the same way: an absent body gives `()`, and a body that is present but does not fit the target is an `error`. Only the nilable form turns an absent body into `()` — a narrow target that is not nilable is handed the builder's empty value (`""`, `[]`, or `{}`), and rejects it unless the narrow type happens to admit it.

An `xml` target is the one builder that rejects an empty body outright: there is no empty `xml` value to bind, so a non-nilable `xml` target returns an error named `NoContentError` with the message `No content`. The nilable form `xml?` still gives `()`.

A `stream<http:SseEvent, error?>` target is covered under [Server-Sent Events](#server-sent-events) below.

Not covered in this subset: status code response records (`http:StatusCodeClient`, `getStatusCodeRecord()`), and the `validation` / `laxDataBinding` client configuration flags.

### Response

| Feature | Notes |
|---|---|
| `getTextPayload` | Now returns `string\|error`, matching jBallerina's signature; extraction failures (for example exceeding `responseLimits.maxEntityBodySize`) surface as an `error` instead of being returned through a `string` signature |
| `setXmlPayload` / `getXmlPayload` | Available on both `http:Request` and `http:Response`. `setXmlPayload` defaults the Content-Type to `application/xml`, keeping an already-set Content-Type when no override is passed |

### Outbound XML

`RequestMessage` is now `anydata|http:Request`, so an `xml` value can be sent directly and is inferred as `application/xml`:

```ballerina
xml echoed = check c->post("/echo", xml `<p>hi</p>`);
```

Resource functions may now return any `anydata` value in addition to `http:Response`, `error`, and `()`. The Content-Type is inferred from the returned value's type — `xml` gives `application/xml`, `string` gives `text/plain`, `byte[]` gives `application/octet-stream`, and everything else is serialised as `application/json`. A non-nil return is answered with 201 for a `post` accessor and 200 otherwise; `()` stays 202 and an `error` stays a 500 envelope.

```ballerina
resource function get album() returns xml {
    return xml `<album><title>Kind of Blue</title></album>`;
}
```

### Byte streams

Request and response bodies can be sent and read as `stream<byte[], io:Error?>`:

| Feature | Notes |
|---|---|
| `setByteStream` | Available on both `http:Request` and `http:Response`. Defaults the Content-Type to `application/octet-stream`, keeping an already-set Content-Type when no override is passed. The body is sent chunked, and a response body is written block by block as the stream produces it |
| `getByteStream` | Reads a body received from the network, filling every array except the last to `arraySize` (default 8192). Returns an `error` for a non-positive `arraySize`, for a body set by a payload setter or already read by a payload getter, and for an empty or multipart body |
| `setPayload` | Accepts a byte stream on both `http:Request` and `http:Response` |
| `RequestMessage` | Now `anydata\|http:Request\|mime:Entity[]\|stream<byte[], io:Error?>`, so a byte stream can be passed directly to `post`, `put`, `patch`, `delete`, and `execute` |

```ballerina
stream<byte[], io:Error?> upload = check io:fileReadBlocksAsStream("./data.bin");
http:Response res = check c->post("/upload", upload);
stream<byte[], io:Error?> download = check res.getByteStream();
```

### Server-Sent Events

A resource function may return `stream<http:SseEvent, error?>` (or `stream<http:SseEvent, error>`) to serve Server-Sent Events. Each event is flushed as soon as it is produced, the response is sent as `text/event-stream` with `Cache-Control: no-cache,public` unless one is already set, and an error from the event stream ends it with a final `error` event. `http:Response` also has `setSseEventStream`, and `setPayload` accepts an event stream.

On the client, a `stream<http:SseEvent, error?>` target, or `getSseEventStream()` on a received response, reads a `text/event-stream` body as `http:SseEvent` records, handing each event on as soon as it is complete:

```ballerina
stream<http:SseEvent, error?> events = check c->get("/events");
record {|http:SseEvent value;|}? event = check events.next();
while event is record {|http:SseEvent value;|} {
    io:println(event.value.data);
    event = check events.next();
}
```
## [mqtt](https://github.com/ballerina-platform/module-ballerina-mqtt/blob/main/docs/spec/spec.md)

Types: `mqtt:Client`, `mqtt:Listener`, `mqtt:Caller` (classes), `mqtt:Service`, `mqtt:Message`, `mqtt:MessageProperties`, `mqtt:DeliveryToken`, `mqtt:Subscription`, `mqtt:ClientConfiguration`, `mqtt:ListenerConfiguration`, `mqtt:ConnectionConfiguration`, `mqtt:WillDetails`, `mqtt:SecureSocket`, `mqtt:CertKey`, `mqtt:Protocol`, `mqtt:Error`, `mqtt:ErrorDetails`, and the `mqtt:DEFAULT_URL` constant.

### Client

| Method | Notes |
|---|---|
| `new mqtt:Client(serverUri, clientId, *config)` | Connects to a `tcp://` or `ssl://` broker URI; returns `mqtt:Error?` |
| `Client->publish(topic, message)` | Publishes with QoS 0, 1 or 2 and waits for delivery to complete; returns `mqtt:DeliveryToken\|mqtt:Error`. An invalid QoS or topic returns an `mqtt:Error` (jBallerina panics) |
| `Client->subscribe(subscriptions)` | Accepts `string`, `string[]`, `mqtt:Subscription` or `mqtt:Subscription[]`; topic filters may use `+` and `#` |
| `Client->receive()` | Returns `stream<mqtt:Message, error?>`; every stream of a client shares one message queue. Read it with `next()` — `foreach` over a stream is not supported by the interpreter yet |
| `Client->isConnected()` / `disconnect()` / `reconnect()` / `close()` | Same connection-state rules and error reason codes as jBallerina; `reconnect()` returns before the reconnect completes |

### Listener and services

| Feature | Notes |
|---|---|
| `new mqtt:Listener(serverUri, clientId, subscriptions, *config)` | Connects on initialization; subscribes when started. `manualAcks` defers acknowledgement to `mqtt:Caller->complete()` |
| `attach` / `detach` / `'start` / `gracefulStop` / `immediateStop` | Usable directly or through module-level `listener` and `service on` declarations. `detach` unsubscribes the listener's topics but keeps the connection, and attaching again re-subscribes (jBallerina disconnects) |
| `onMessage(mqtt:Message)` / `onMessage(mqtt:Message, mqtt:Caller)` | Invoked once per message, one message at a time |
| `onError(mqtt:Error)` | Invoked when the network connection is lost, the service has no `onMessage`, or `onMessage` panics (jBallerina prints the panic); errors with no `onError` are printed to stderr |
| `onComplete(mqtt:DeliveryToken)` | Invoked when a `Caller->respond()` reply has been delivered |
| `Caller->respond(message)` / `Caller->complete()` | Replies to the request's `responseTopic` with its `correlationData`, and acknowledges the message in `manualAcks` mode. Both return an `mqtt:Error` when the reply or acknowledgement fails (jBallerina returns `()`) |

### Connection and security

| Feature | Notes |
|---|---|
| Authentication | `username` and `password` |
| Connection options | `keepAliveInterval`, `connectionTimeout`, `cleanStart`, `serverUris` (tried in order), `automaticReconnect`, `maxReconnectDelay`, and `willDetails` |
| TLS / mutual TLS | `SecureSocket.cert` as a PEM file path, `SecureSocket.key` as an `mqtt:CertKey` with an unencrypted PEM key, and `SecureSocket.protocol` |

Not covered in this subset: `crypto:TrustStore` / `crypto:KeyStore` TLS and encrypted private keys, and `ws://` / `wss://` broker URIs. jBallerina's compile-time service validation is applied when a service is attached: an invalid service method signature makes `attach` return an `mqtt:Error` with the compiler plugin's message.

## [protobuf](https://github.com/ballerina-platform/module-ballerina-protobuf/blob/master/docs/spec/spec.md)

`ballerina/protobuf` provides the type declarations for the protobuf well-known types used by generated gRPC client/service code, plus packing and unpacking values into `protobuf.types.any:Any` for the well-known types. The separate `.proto`-to-Ballerina code generator that produces gRPC client/service stubs and `@protobuf:Descriptor`-annotated message records is not part of this package and is out of scope.

| Feature | Notes |
|---|---|
| `protobuf:Error`, `protobuf:MessageDescriptor`, `protobuf:Descriptor` annotation | Declared and attachable, but not yet used to resolve a message record. `pack` cannot reach the annotation — it receives only the value, and `typeof` is not implemented yet; `unpack` does receive the annotated typedesc and is not blocked by that. An arbitrary `record {}` value passed to `pack`/`unpack` is still handled, via the `google.protobuf.Struct` fallback described below, rather than resolved through its own descriptor |
| `protobuf.types.any:pack(message)` | Supported for `int`, `float`, `string`, `boolean`, `byte[]`, `()`, `time:Utc`, `time:Seconds`, and `map<anydata>`. The closed empty record `record {\|\|}` packs as `google.protobuf.Empty`; any other `record {}` value is packed as `google.protobuf.Struct` rather than resolved via its own descriptor — see the package README |
| `protobuf.types.any:unpack(anyValue, targetTypeOfAny = <>)` | Supported for the same well-known types, using the interpreter's inferred-typedesc-parameter support; a mismatched target raises `protobuf.types.any:TypeMismatchError` |
| `protobuf.types.any:Any`, `ContextAny`, `ContextAnyStream` | Supported |
| `protobuf.types.duration`, `protobuf.types.empty`, `protobuf.types.struct`, `protobuf.types.timestamp`, `protobuf.types.wrappers` context record types | Supported |

Not covered in this subset: arbitrary user-defined message record (de)serialization via the `@protobuf:Descriptor` annotation — blocked on `typeof` support rather than on the library — and the `.proto`-to-Ballerina code generator (`ballerina/grpc` and its tooling are not ported).

## [file](https://github.com/ballerina-platform/module-ballerina-file/blob/master/docs/spec/spec.md)

File, directory, and path manipulation utilities.

| Feature | Notes |
|---|---|
| `create` / `remove` / `rename` / `copy` | Create, remove (non-recursive and recursive), rename/move, and copy (with `REPLACE_EXISTING`, `COPY_ATTRIBUTES`, `NO_FOLLOW_LINKS` options) files and directories |
| `getMetaData` | File size, modification time, permissions, and type |
| `readDir` | Read directory contents |
| `createTemp` / `createTempDir` | Create a temporary file / directory |
| `test` | Test file/directory properties: `EXISTS`, `IS_DIR`, `IS_SYMLINK`, `READABLE`, `WRITABLE` |
| `getCurrentDir` | Get the current working directory |
| `getAbsolutePath` / `isAbsolutePath` / `basename` / `parentPath` / `normalizePath` / `splitPath` / `joinPath` / `relativePath` | Cross-platform path manipulation |
| `Listener` / `Service` | Directory change listener: attaches a service whose `onCreate`/`onModify`/`onDelete` remote methods are dispatched on filesystem changes under the configured `path` (optionally `recursive`) |

`file:Error`'s `distinct` subtypes (`FileNotFoundError`, `PermissionError`, etc.) are declared as plain type aliases of `Error` instead — they are structurally identical at runtime, so `error is file:FileNotFoundError`-style checks don't narrow. `file:Service` is likewise declared as a plain (non-`distinct`) `service object {}` marker instead of jBallerina's `distinct service object {}`. Unlike jBallerina, `gracefulStop()` closes the underlying OS watch immediately (same as `immediateStop()`) rather than leaving it running until process exit, and `attach()` returns its "at least one resource required" validation error through its `error?` return type instead of throwing it.

## [uuid](https://github.com/ballerina-platform/module-ballerina-uuid/blob/master/docs/spec/spec.md)

RFC 4122 UUID generation, validation, version detection, and conversions
between string, byte array, and the `Uuid` record.

| Function | Notes |
|---|---|
| `createType1AsString` / `createType1AsRecord` | Time-based UUID (type 1) in the RFC 4122 field layout (`time_low` first) with a random 6-byte node identifier (RFC 4122 §4.5) |
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

## [ldap](https://github.com/ballerina-platform/module-ballerina-ldap/blob/master/docs/spec/spec.md)

Types: `ldap:Client` (client class), `ldap:Error`.

| Method | Notes |
|---|---|
| `new ldap:Client(config)` | Connects and performs a simple bind with `domainName` as the bind DN; returns `ldap:Error` on a connection, TLS or bind failure. `config` is a plain record parameter, so pass a record literal rather than named arguments |
| `Client->add(dN, entry)` | Adds an entry; returns `ldap:LdapResponse\|ldap:Error` |
| `Client->delete(dN)` | Deletes an entry |
| `Client->modify(dN, entry)` | Replaces each given attribute |
| `Client->modifyDn(currentDn, newRdn, deleteOldRdn?)` | Renames an entry |
| `Client->compare(dN, attributeName, assertionValue)` | Returns `boolean\|ldap:Error` |
| `Client->getEntry(dN, attributes?, targetType?)` | Returns the entry bound to the type inferred from the call site. Attributes to fetch are inferred from a closed target record's fields when not given. A declared field is converted from the attribute's string value; a required field with no matching attribute or a value that can't be converted returns `ldap:Error` |
| `Client->search(baseDn, filter, scope, attributes?)` | Returns `ldap:SearchResult\|ldap:Error`; no matching entry returns `ldap:Error` |
| `Client->searchWithType(baseDn, filter, scope, attributes?, targetType?)` | As `search`, binding each entry to the element type of `targetType` as `getEntry` does |
| `Client->close()` | Sends an unbind request and closes the connection |
| `Client->isConnected()` | Returns whether the connection is open |

TLS is selected by supplying `clientSecureSocket`. The server certificate is trusted through a PEM certificate file path in `cert`; a `crypto:TrustStore` returns `ldap:Error`. `verifyHostName` and `tlsVersions` are supported.

Not covered in this subset: the `ldap:Error` detail record (`error:detail()` is not implemented yet, so the result code is only visible in the message), and `distinct` error types.

## [udp](https://github.com/ballerina-platform/module-ballerina-udp/blob/master/docs/spec/spec.md)

Types: `udp:Client`, `udp:ConnectClient`, `udp:Listener`, `udp:Caller` (classes), `udp:Service`, `udp:Datagram`, `udp:ClientConfiguration`, `udp:ConnectClientConfiguration`, `udp:ListenerConfiguration`, `udp:Error`.

| Feature | Notes |
|---|---|
| `new udp:Client(config)` | Connectionless client bound to `localHost` (all interfaces when omitted) with a read `timeout` |
| `Client->sendDatagram(datagram)` / `Client->receiveDatagram()` / `Client->close()` | Payloads larger than 8192 bytes are sent as multiple datagrams, as in jBallerina. `receiveDatagram()` returns a `readonly & udp:Datagram` whose `remoteHost`/`remotePort` are the sender's |
| `new udp:ConnectClient(remoteHost, remotePort, config)` | Connection-oriented client that only exchanges datagrams with the given peer |
| `ConnectClient->writeBytes(data)` / `ConnectClient->readBytes()` / `ConnectClient->close()` | Payloads larger than 8192 bytes are sent as multiple datagrams, as in jBallerina. `readBytes()` returns one datagram as `readonly & byte[]` and does not reassemble a fragmented write, so callers must reassemble application-level messages |
| `new udp:Listener(localPort, config)` | Binds on `start()`; `attach`, `detach`, `start`, `gracefulStop` and `immediateStop` are supported, and a stopped listener can be started again |
| `udp:Service` | Declares exactly one of `onBytes(readonly & byte[] data, udp:Caller caller)` or `onDatagram(readonly & udp:Datagram datagram, udp:Caller caller)`, plus an optional `onError(udp:Error err)`. The `Caller` parameter is optional and parameters may appear in either order. A returned `byte[]` or `udp:Datagram` is sent as a reply; the service is validated when it is attached |
| `Caller->sendBytes(data)` / `Caller->sendDatagram(datagram)` | Reply to the sender or send to any destination |

`config` is a plain record parameter, so pass a record literal (`{localHost: "x"}`) rather than named arguments. `remoteHost` is reported as an IP address without a reverse-DNS lookup. Service methods are checked by `Listener.attach()` instead of at compile time. `udp:Service` is a plain (non-`distinct`) service object type and `udp:Error` is an alias of `error`.

Not covered in this subset: a listener restricted to one remote peer (`ListenerConfiguration.remoteHost`/`remotePort`) and `distinct` error types.

## [mime](https://github.com/ballerina-platform/module-ballerina-mime/blob/master/docs/spec/spec.md)

`ballerina/mime` provides media type parsing, content disposition handling, MIME entity header/body management, multipart bodies, and Base64 encoding/decoding.

Types: `mime:MediaType`, `mime:ContentDisposition`, `mime:Entity` (class); `mime:Error` and subtypes (`InvalidContentTypeError`, `ParserError`, `HeaderNotFoundError`, `EncodeError`, `DecodeError`) as plain `error` aliases — `distinct` is not yet supported.

| Feature | Notes |
|---|---|
| `getMediaType(contentType)` / `getContentDispositionObject(contentDisposition)` | Parse a `Content-Type` / `Content-Disposition` header value |
| Entity header and content metadata | `setHeader`/`getHeader`/`getHeaders`/`getHeaderNames`/`addHeader`/`removeHeader`/`removeAllHeaders`/`hasHeader`; `setContentType`/`getContentType`/`setContentId`/`getContentId`/`setContentLength`/`getContentLength`/`setContentDisposition`/`getContentDisposition` |
| Entity body — text, JSON, XML, byte array | `setText`/`getText`, `setJson`/`getJson`, `setXml`/`getXml`, `setByteArray`/`getByteArray`; every accessor lazily converts from whatever the body was actually set as, matching jBallerina's data-source model |
| `setBody(string\|xml\|json\|byte[]\|Entity[])` | Generic body dispatch by argument type |
| `setFileAsEntityBody(filePath, contentType?)` | Sets the file as a lazy byte-channel data source, read on demand by whichever accessor materializes the body first, matching jBallerina; a failed file open panics, matching jBallerina's own `checkpanic io:openReadableFile` |
| Multipart bodies | `setBodyParts`/`getBodyParts` — `message/*` bodies return a `ParserError`; jBallerina silently returns an empty array for these instead |
| `base64Encode`/`base64Decode`/`base64EncodeBlob`/`base64DecodeBlob` | Accept `string`, `byte[]`, or an `io:ReadableByteChannel` (read fully, result handed back as a freshly-constructed channel; charset does not apply to the channel form, matching jBallerina) |

## [task](https://github.com/ballerina-platform/module-ballerina-task/blob/master/docs/spec/spec.md)

Scheduling of `task:Job` objects once or at a fixed interval, with the error and waiting policies and job management of jBallerina's Quartz-based scheduler.

Types: `task:Job`, `task:JobId`, `task:TaskPolicy`, `task:ErrorPolicy`, `task:WaitingPolicy`, `task:Service`, `task:RetryConfiguration`, `task:Listener` (class), `task:Error`.

| Function | Notes |
|---|---|
| `scheduleOneTimeJob(job, triggerTime)` | Runs the job once at a future `time:Civil`; a panic in `execute` is logged and ignored |
| `scheduleJobRecurByFrequency(job, interval, maxCount?, startTime?, endTime?, taskPolicy?)` | Runs the job every `interval` seconds, optionally bounded by a count and a start/end window |
| `pauseJob` / `resumeJob` / `pauseAllJobs` / `resumeAllJobs` | A resumed job applies its waiting policy to the runs it missed: `WAIT` fires them all, `IGNORE` and `LOG_AND_IGNORE` skip them |
| `unscheduleJob` / `getRunningJobs` | Unscheduling the last job shuts the scheduler down; running jobs are listed in ascending id order |
| `configureWorkerPool(workerCount?, waitingTime?)` | `waitingTime` is how late a run may be before the waiting policy treats it as missed |
| `getTimeInMillies(time)` | Epoch milliseconds of a future `time:Civil` |
| `new task:Listener(trigger = ...)`, `attach` / `detach`, `start` / `gracefulStop` / `immediateStop` | Runs each attached `task:Service` on the trigger, retrying failed executions per `retryConfig`; `execute` must be declared `public`; warm-backup task coordination is not yet supported |
