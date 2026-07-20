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

`io:Block` is declared as `byte[]` rather than jBallerina's `readonly & byte[]` (`readonly &` intersections are not yet supported). The read-as-stream and write-from-stream functions read/write lazily (incrementally) rather than buffering the whole file; open errors surface at the `fileReadLinesAsStream`/`fileReadBlocksAsStream`/`fileWriteLinesFromStream`/`fileWriteBlocksFromStream` call, while read errors surface during a later `next()`. The write-from-stream functions widen their stream parameter's completion type to the generic `error?` (jBallerina uses `io:Error?`), so a stream held as `stream<_, error?>` — such as one bound directly from `fileReadBlocksAsStream` — can be written back; jBallerina rejects that. The returned streams are consumed with explicit `next()`/`close()` calls; iterating a stream with `foreach` or a query expression is not yet supported at the language level.

### Byte channels

| Feature | Notes |
|---|---|
| `openReadableFile(path)` | Returns `io:ReadableByteChannel\|io:Error` for streaming reads from a file |
| `openWritableFile(path, option?)` | Returns `io:WritableByteChannel\|io:Error` for streaming writes to a file; `OVERWRITE`/`APPEND` supported |
| `createReadableChannel(content)` | Returns `io:ReadableByteChannel\|io:Error` wrapping an in-memory `byte[]`, no file involved |
| `ReadableByteChannel.read(nBytes)` | Returns up to `nBytes` as `byte[]` — may return fewer bytes than requested (a single read, not a guaranteed full read); errors once the channel is exhausted |
| `ReadableByteChannel.readAll()` | Reads the remaining content of the channel to completion as `byte[]` |
| `ReadableByteChannel.blockStream(blockSize)` | Returns `stream<io:Block, io:Error?>\|io:Error` yielding `blockSize` `byte[]` blocks per `next()`; the final block may be shorter |
| `ReadableByteChannel.close()` | Releases the channel's underlying resources; a second `close()` call errors |
| `WritableByteChannel.write(content, offset)` | Writes `content[offset:]`; `offset` is an index into `content`, not a file seek offset; returns the number of bytes written |
| `WritableByteChannel.close()` | Releases the channel's underlying resources; a second `close()` call errors |

`ReadableByteChannel.base64Encode()`/`base64Decode()` are not implemented in
this subset. Character channels, data channels, and CSV/record channels are
also out of scope for this subset and remain `Not Yet Supported`.
