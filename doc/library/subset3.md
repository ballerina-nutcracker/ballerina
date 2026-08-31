# Supported ballerina library features

Subset 3 extends the released [subset 2](subset2.md) with the `avro` module —
Avro binary serialization and deserialization driven by an Avro schema string —
plus stream-based file read/write additions and byte channels in the `io`
module, building on the language's new `stream` type, and with client-side
response data binding, XML payloads, and `anydata` resource returns for the
`http` module.

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

`fromAvro` infers `targetType` from the contextually expected type, so
`Person p = check schema.fromAvro(data);` binds the decoded payload to `Person`.
Without a contextually expected type the compiler reports `cannot infer typedesc
argument for parameter 'targetType'` — pass `targetType = Person` explicitly in
that case.

Records, enums, tuples, singletons, `map<T>`, `T[]`, `json`, `map<json>`,
`anydata` and nilable forms of all of these are accepted as targets, along with
numeric widening from `int` to `float` and `decimal`. A `readonly &`
intersection of any of these is accepted too and binds to a frozen value.

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

The client can also address a resource with the client resource access syntax (`client->/albums/[id]`) rather than only through a remote method with a string path.

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

Not covered in this subset: `stream<http:SseEvent, error?>` targets, status code response records (`http:StatusCodeClient`, `getStatusCodeRecord()`), and the `validation` / `laxDataBinding` client configuration flags.

### Client — resource methods

All seven accessors are declared as resource methods on `http:Client`, so a request can be written as a path rather than as a string argument:

```ballerina
resource isolated function get [PathParamType... path](map<string|string[]>? headers = (),
        *QueryParams params) returns Response|error = external;
```

`PathParamType` is `boolean|int|float|decimal|string`. Each segment of the call site's path becomes one element of `path`, whether it was written as a name or as a computed segment, and `get` is the accessor used when the call site names none:

```ballerina
http:Client c = check new ("https://example.com");

http:Response a = check c->/albums/[42]/tracks;      // GET /albums/42/tracks
http:Response b = check c->/albums.post(payload);    // POST /albums
http:Response d = check c->/albums/[42].delete();    // DELETE /albums/42
http:Response e = check c->/;                        // GET /
```

`post`, `put`, `patch` and `delete` take a leading `message`, plus `headers` and `mediaType`, exactly as their remote counterparts do; `delete`'s `message` is defaultable. `head` and `options` take only `headers`.

Query parameters are named arguments, collected by the `*QueryParams` included record parameter:

```ballerina
public type QueryParamType SimpleQueryParamType[]|SimpleQueryParamType;

public type QueryParams record {|
    never headers?;
    never targetType?;
    never message?;
    never mediaType?;
    QueryParamType...;
|};
```

The four `never` fields reserve the names the method's own parameters use, so `headers`, `targetType`, `message` and `mediaType` cannot be smuggled in as query parameters.

| Call | Request target |
|---|---|
| `c->/albums(tag = "rock", count = 3)` | `/albums?tag=rock&count=3` |
| `c->/albums(tags = ["rock", "jazz"])` | `/albums?tags=rock,jazz` |
| `c->/albums(q = "a b")` | `/albums?q=a+b` |
| `c->/albums(q = "a&b=c")` | `/albums?q=a%26b%3Dc` |
| `c->/p/[1.0]/[1.0e10]` | `/p/1.0/1e10` |
| `c->/p/[d]` where `decimal d = 2.500` | `/p/2.500` |

An array-valued query parameter is rendered as a single comma-joined pair rather than as a repeated key, and parameters keep the order the named arguments were written in.

Two renderings deliberately differ from jBallerina's wire format, in both cases by preferring the Go-native or language-consistent behaviour over byte-level parity. Each is recorded in the [`http` README](../../lib/stdlibs/ballerina/http/0.0.1/go1.26/README.md) under **Notable Behavioural Changes** and is open to revisiting:

- **Values use Ballerina's `toString`.** jBallerina formats a segment with Java's `String.valueOf`, so a `float` goes out as `Double.toString` gives it — `1.0E10`. Here every segment and query value goes through Ballerina's own `toString`, giving `1e10`. `int`, `boolean`, `decimal` and `string` are identical either way; a whole float still keeps its `.0` and a decimal still keeps its trailing zeros.
- **Query components are escaped.** jBallerina concatenates the target and lets the transport encode only the space, so a value containing `&` or `=` silently splits into extra parameters. Here keys and values are escaped with `url.QueryEscape`, so such a value stays one parameter — at the cost of a space rendering as `+` rather than `%20`.

Not covered in this subset: the `targetType` parameter. jBallerina's resource methods are dependently typed on `TargetType targetType = <>`, which the interpreter does not yet support for resource methods (#825), so these methods return the raw `http:Response` and a target of any other type is a compile error. Use the remote method form when the payload needs binding. The `@http:Query` annotation for renaming a query parameter on the wire is also not supported.

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
