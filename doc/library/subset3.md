# Supported ballerina library features

Subset 3 extends the released [subset 2](subset2.md) with stream-based file
read/write additions to `io` that build on the language's new `stream` type.

## [io](https://github.com/ballerina-platform/module-ballerina-io/blob/master/docs/spec/spec.md)

Subset 2 covered console printing and whole-file I/O (string, lines, bytes,
JSON, XML). Subset 3 adds stream-based file reading and writing, built on the
language's new `stream` type.

| Feature | Notes |
|---|---|
| `fileReadLinesAsStream(path)` | Returns `stream<string, io:Error?>` yielding one line per `next()`; terminal carriage characters stripped, trailing empty line excluded |
| `fileReadBlocksAsStream(path, blockSize?)` | Returns `stream<io:Block, io:Error?>` yielding `byte[]` blocks of `blockSize` (default 4096); the final block may be shorter |
| `fileWriteLinesFromStream(path, lineStream, option?)` | Consumes a `stream<string, error?>`, appending `\n` after each line; `OVERWRITE`/`APPEND` supported |
| `fileWriteBlocksFromStream(path, byteStream, option?)` | Consumes a `stream<byte[], error?>`, concatenating blocks in order; `OVERWRITE`/`APPEND` supported |

`io:Block` is declared as `byte[]` rather than jBallerina's `readonly & byte[]` (`readonly &` intersections are not yet supported). The read-as-stream and write-from-stream functions read/write lazily (incrementally) rather than buffering the whole file; open errors surface at the `fileReadLinesAsStream`/`fileReadBlocksAsStream`/`fileWriteLinesFromStream`/`fileWriteBlocksFromStream` call, while read errors surface during a later `next()`. The write-from-stream functions widen their stream parameter's completion type to the generic `error?` (jBallerina uses `io:Error?`), so a stream held as `stream<_, error?>` — such as one bound directly from `fileReadBlocksAsStream` — can be written back; jBallerina rejects that. The returned streams are consumed with explicit `next()`/`close()` calls; iterating a stream with `foreach` or a query expression is not yet supported at the language level.
