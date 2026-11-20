# Ballerina Zip Library

## Overview

This module provides APIs to create, inspect, and extract ZIP archives. It reads archives through `zip:ArchiveReader`, which lists entries, reads their content as a byte array or a stream, and extracts them with protection against unsafe entry names and extraction limits against archives that expand to an unreasonable size. It creates archives through `zip:ArchiveWriter`, which adds files, directories, in-memory or streamed content, and entries copied raw from another archive. The `compress`, `decompress`, and `listEntries` functions cover the common cases in one call.

The Go Native Interpreter supports the whole module; the three error types that name an entry carry their entry name without the `EntryErrorDetail` detail type.

## Key Functionalities

- Compress a file or a directory into an archive, and decompress an archive into a directory.
- List the entries of an archive with their names, sizes, compression method, CRC-32, modification time, comment, and Unix mode.
- Read an entry as a byte array or as a stream of chunks.
- Extract one entry or the whole archive, with a choice of what to do about files already there, and limits on the entry count, total size, and compression ratio.
- Refuse entry names that would write outside the target directory, and symbolic link entries.
- Create an archive from files, directories, byte arrays, and streams, at four compression levels.
- Copy an entry from one archive to another without decompressing it.
- Read and write Zip64 archives; read CP437 and UTF-8 entry names.

## Examples

```ballerina
import ballerina/io;
import ballerina/zip;

public function main() returns error? {
    check zip:compress("./reports", "./reports.zip");
    foreach zip:Entry entry in check zip:listEntries("./reports.zip") {
        io:println(entry.name, " ", entry.uncompressedSize);
    }

    zip:ArchiveReader reader = check new ("./reports.zip");
    byte[] summary = check reader.readEntry("reports/summary.txt");
    io:println(check string:fromBytes(summary));
    check reader.extractAll("./restored", {limits: {maxTotalSize: 10000000}});
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
| Compress a file or directory | Supported | |
| Decompress an archive | Supported | |
| List the entries of an archive | Supported | |
| Open an archive for reading | Supported | |
| Entry metadata | Supported | |
| Single entry lookup and existence check | Supported | |
| Read an entry as a byte array | Supported | |
| Read an entry as a stream | Supported | |
| Extract a single entry | Supported | |
| Extract every entry | Supported | |
| What to do about an existing file | Supported | |
| Extraction limits | Supported | For a deflated entry, the point where a limit stops the write can differ; see Notable Behavioural Changes |
| Unsafe entry name and symbolic link refusal | Supported | |
| Close a reader | Supported | |
| Create an archive | Supported | |
| Add a file | Supported | |
| Add a directory tree | Supported | |
| Add an entry from a byte array | Supported | |
| Add an entry from a stream | Supported | |
| Copy an entry between archives | Supported | |
| Compression levels | Supported | Deflated output differs from jBallerina's; see Notable Behavioural Changes |
| Close a writer | Supported | |
| Zip64 archives | Supported | |
| CP437 and UTF-8 entry names | Supported | |
| Options and entry types | Supported | `CompressOptions`, `DecompressOptions`, `ExtractionLimits`, `CompressionMethod`, `CompressionLevel`, `FileWriteMode` |
| Module-level error types | Supported | `Error`, `InvalidArchiveError`, `EntryNotFoundError`, and `FileSystemError` are `distinct`, as in jBallerina |
| Errors about a single entry | Partially Supported | `UnsupportedEntryError`, `UnsafePathError`, and `LimitExceededError` are declared `distinct Error` instead of `distinct (Error & error<EntryErrorDetail>)`, since this interpreter cannot yet make an error intersection `distinct` (ballerina-nutcracker/ballerina#1427). They still carry `entryName` in their detail, but its static type is not `EntryErrorDetail` |

### Notable Behavioural Changes

- **Compressed output differs.** jBallerina deflates entries with zlib; the Go-native version uses Go's `compress/flate`, which encodes the same content at the same level into different bytes — the content and CRC-32 of each entry are the same, but `compressedSize` and the size of the archive are not, and for very small entries the Go output is a few bytes larger.
- **Extraction limits on deflated entries stop at a different point.** jBallerina inflates a deflated entry 512 compressed bytes at a time and judges `maxTotalSize` and `maxCompressionRatio` against whatever each step produces; the Go-native version judges each full 8 KiB of output — both stop with the same `LimitExceededError` and message, but a deflated entry can trip a ratio limit after a different number of bytes, and the partial file a `maxTotalSize` limit leaves behind can be a different size or absent. Stored entries stop at the same point.
