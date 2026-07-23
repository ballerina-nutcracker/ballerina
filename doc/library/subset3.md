# Supported ballerina library features

Subset 3 extends the released [subset 2](subset2.md) with stream-based file
read/write additions and byte channels in the `io` module, building on the
language's new `stream` type.

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
