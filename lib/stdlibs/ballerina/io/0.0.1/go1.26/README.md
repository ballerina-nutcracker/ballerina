# Ballerina IO Library

## Overview

This module provides I/O operations for Ballerina programs. The full jBallerina `io` module covers console output, file I/O (string, bytes, JSON, XML, CSV, lines), low-level byte/character/data channels, and stream-based reading and writing. The Go Native Interpreter currently supports console printing, file I/O (string, lines, bytes, JSON, XML), stream-based line/block reading and writing, byte channels (`read`, `readAll`, `blockStream`, `base64Encode`, `base64Decode`, `write`, `close`), and character channels (charset-aware string, line, JSON, XML, and `.properties` reading and writing).

## Key Functionalities

- Print `any` or `error` values to the standard output stream using `print` and `println`.
- Print to a specified output stream (stdout or stderr) using `fprint` and `fprintln`.
- Read file content as a string, line array, byte array, JSON, or XML using `fileReadString`, `fileReadLines`, `fileReadBytes`, `fileReadJson`, and `fileReadXml`.
- Write string, line array, byte array, JSON, or XML content to a file using `fileWriteString`, `fileWriteLines`, `fileWriteBytes`, `fileWriteJson`, and `fileWriteXml`.
- Read file content as a stream of lines or byte blocks using `fileReadLinesAsStream` and `fileReadBlocksAsStream`.
- Write a stream of lines or byte blocks to a file using `fileWriteLinesFromStream` and `fileWriteBlocksFromStream`.
- Control write behaviour with the `FileWriteOption` enum (`OVERWRITE` or `APPEND`).
- Read bytes from a file or an in-memory byte array using `io:ReadableByteChannel`, obtained via `openReadableFile` or `createReadableChannel`, with `read`, `readAll`, `blockStream`, `base64Encode`, `base64Decode`, and `close`.
- Write bytes to a file using `io:WritableByteChannel`, obtained via `openWritableFile`, with `write` and `close`.
- Read characters, lines, JSON, XML, or `.properties` content from a byte channel with a chosen charset using `io:ReadableCharacterChannel`.
- Write characters, lines, JSON, XML, or `.properties` content to a byte channel with a chosen charset using `io:WritableCharacterChannel`.

## Examples

```ballerina
import ballerina/io;

public function main() returns error? {
    io:println("Starting process...");
    io:print("Value: ", 42);
    io:fprint(io:stderr, "An unexpected error occurred");

    // Write and read a file
    check io:fileWriteString("/tmp/greet.txt", "Hello\nWorld");
    string content = check io:fileReadString("/tmp/greet.txt");
    io:println(content);

    // Append to a file
    check io:fileWriteString("/tmp/greet.txt", "\nAppended", io:APPEND);

    // Write and read lines
    check io:fileWriteLines("/tmp/lines.txt", ["Alpha", "Beta"]);
    string[] lines = check io:fileReadLines("/tmp/lines.txt");
    foreach string line in lines {
        io:println(line);
    }

    // Write and read bytes
    check io:fileWriteBytes("/tmp/data.bin", [72, 101, 108, 108, 111]);
    byte[] bytes = check io:fileReadBytes("/tmp/data.bin");
    io:println(bytes.length());

    // Write and read JSON
    check io:fileWriteJson("/tmp/data.json", {"name": "Alice", "age": 30});
    json result = check io:fileReadJson("/tmp/data.json");
    io:println(result);

    // Write and read XML
    check io:fileWriteXml("/tmp/data.xml", xml `<book><title>Clean Code</title></book>`);
    xml xmlResult = check io:fileReadXml("/tmp/data.xml");
    io:println(xmlResult);

    // Byte channels
    io:WritableByteChannel writer = check io:openWritableFile("/tmp/channel.bin");
    check writer.write([1, 2, 3, 4, 5], 0);
    check writer.close();

    io:ReadableByteChannel reader = check io:openReadableFile("/tmp/channel.bin");
    byte[] channelContent = check reader.readAll();
    io:println(channelContent);
    check reader.close();
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
| Print to standard output | Supported | |
| Print to standard output with a newline | Supported | |
| Print to a specified output stream | Supported | |
| Print to a specified output stream with a newline | Supported | |
| Console read | Not Yet Supported | `readln` (reads a line from stdin) is not implemented. |
| String template support in print functions | Not Yet Supported | `PrintableRawTemplate` type is not yet defined; string templates cannot be passed directly to print functions. As a consequence, the `Printable` type in this implementation is `any\|error` rather than jBallerina's `any\|error\|PrintableRawTemplate`. |
| File read — string | Supported | `fileReadString`. Line endings normalised to `\n`; trailing newline stripped. |
| File read — lines | Supported | `fileReadLines`. Terminal carriage characters stripped; trailing empty line excluded. |
| File read — bytes | Supported | `fileReadBytes`. Returns `readonly & byte[]`. |
| File read — JSON | Supported | `fileReadJson`. |
| File read — stream of lines | Supported | `fileReadLinesAsStream`. Returns `stream<string, Error?>`. Terminal carriage characters stripped; trailing empty line excluded. See Notable Behavioural Changes (manual `next()`/`close()` consumption). |
| File read — stream of blocks | Supported | `fileReadBlocksAsStream`. Returns `stream<Block, Error?>` where `Block` is `readonly & byte[]`. Default `blockSize` is 4096. See Notable Behavioural Changes. |
| File write — string | Supported | `fileWriteString`. `OVERWRITE` and `APPEND` modes supported. |
| File write — lines | Supported | `fileWriteLines`. `OVERWRITE` and `APPEND` modes supported; `\n` appended after each line. |
| File write — bytes | Supported | `fileWriteBytes`. `OVERWRITE` and `APPEND` modes supported. |
| File write — JSON | Supported | `fileWriteJson`. Always overwrites; JSON object keys sorted alphabetically. See Notable Behavioural Changes. |
| File write — stream of lines | Supported | `fileWriteLinesFromStream`. Consumes a `stream<string, error?>`; `\n` appended after each line; `OVERWRITE` and `APPEND` modes supported. Parameter completion type widened to the generic `error?` — see Notable Behavioural Changes. |
| File write — stream of blocks | Supported | `fileWriteBlocksFromStream`. Consumes a `stream<byte[], error?>`; blocks concatenated in order; `OVERWRITE` and `APPEND` modes supported. Parameter completion type widened to the generic `error?` — see Notable Behavioural Changes. |
| File I/O — XML | Supported | `fileReadXml`, `fileWriteXml`. `OVERWRITE` and `APPEND` modes supported. |
| File I/O — CSV | Not Yet Supported | `fileReadCsv`, `fileWriteCsv`, stream variants. `stream` type not yet supported; `typedesc` parameter handling complex. |
| File write option enum | Supported | `FileWriteOption`: `OVERWRITE` and `APPEND` constants. |
| Module-level error type | Partially Supported | `io:Error` declared as a plain `error` alias; `distinct` error subtypes (`FileNotFoundError`, `GenericError`, `AccessDeniedError`, `EofError`, `ConfigurationError`, `TypeMismatchError`) not yet supported. |
| Byte channels | Supported | `ReadableByteChannel`: `read`, `readAll` (returns `readonly & byte[]`), `blockStream`, `base64Encode`, `base64Decode`, `close`. `WritableByteChannel`: `write`, `close`. |
| Character channels | Supported | `ReadableCharacterChannel`: `read`, `readString`, `readAllLines`, `readJson`, `readXml`, `readProperty`, `readAllProperties`, `lineStream`, `close`, constructed as `new (byteChannel, charset)`. `WritableCharacterChannel`: `write` (returns bytes written, as in jBallerina), `writeLine`, `writeJson`, `writeXml` (with optional `XmlDoctype`), `writeProperties`, `close`. The `LineStream`/`BlockStream` public helper classes are not declared; `lineStream()`/`blockStream()` return plain stream values instead. |
| Data channels | Not Yet Supported | Not implemented. |
| CSV channels | Not Yet Supported | Not implemented. |
| Channel file open functions | Partially Supported | `openReadableFile`, `openWritableFile`, `createReadableChannel` supported. `openReadableCsvFile`, `openWritableCsvFile` not yet supported (depend on CSV channels). |

### Notable Behavioural Changes

- **`fileWriteJson` key ordering.** jBallerina writes JSON object keys in insertion order; the Go-native version writes them in **alphabetical order** — Go's `encoding/json` sorts map keys.
- **Streams are consumed via `next()`/`close()` only.** The returned streams are driven with explicit `.next()` and `.close()` calls. Iterating a stream with a `foreach` statement or a query (`from ... in`) expression is not yet supported at the language level, so those constructs cannot yet consume these streams.
- **Write-from-stream accepts a generic `error?` completion.** jBallerina declares `fileWriteLinesFromStream`/`fileWriteBlocksFromStream` with a `stream<_, io:Error?>` parameter, which rejects a stream held as `stream<_, error?>` (e.g. `stream<byte[], error?> s = check io:fileReadBlocksAsStream(p); check io:fileWriteBlocksFromStream(out, s);` fails to compile in jBallerina). This port widens the parameter completion type to the generic `error?`, so both `io:Error?` and plain `error?` completion streams are accepted. This is a strict superset — every jBallerina-valid call still compiles — and the return type remains the specific `io:Error?`.
