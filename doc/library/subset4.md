# Supported ballerina library features

Subset 4 extends the released [subset 3](subset3.md) with the new `ballerina/zip` package for creating, reading, and extracting ZIP archives.

## [zip](https://github.com/ballerina-platform/module-ballerina-zip/blob/main/docs/spec/spec.md)

Creating, inspecting, and extracting ZIP archives. Types: `zip:ArchiveReader` and `zip:ArchiveWriter` (classes), `zip:Entry`, `zip:CompressOptions`, `zip:DecompressOptions`, `zip:ExtractionLimits`, the `CompressionMethod`, `CompressionLevel`, and `FileWriteMode` enums, and the `zip:Error` hierarchy.

| Function | Notes |
|---|---|
| `compress(sourcePath, targetPath, options?)` | Archive a file or a directory; refuses a target inside the source or the same file as it |
| `decompress(sourcePath, targetPath, options?)` | Extract every entry into a directory, created if missing |
| `listEntries(path)` | Metadata of every entry in the order stored |
| `new zip:ArchiveReader(path)` | Open an archive; `entries`, `getEntry`, `hasEntry`, `close` |
| `ArchiveReader.readEntry(name, targetType?)` | Content as `byte[]` or as a `stream<byte[], zip:Error?>` of 64 KiB chunks, chosen by the target type |
| `ArchiveReader.extractEntry(name, targetPath, options?)` / `extractAll(targetPath, options?)` | Extract with `FAIL_IF_EXISTS`, `REPLACE`, or `SKIP`, and `maxEntries`, `maxTotalSize`, and `maxCompressionRatio` limits; refuses unsafe entry names, links leading out of the target, and symbolic link entries |
| `new zip:ArchiveWriter(path, options?)` | Create an archive at the `NONE`, `FASTEST`, `DEFAULT`, or `BEST` level; refuses an existing file unless `overwrite` is set |
| `ArchiveWriter.addFile` / `addDirectory` / `addEntry` | Add a file, a directory tree (walked in name order, links skipped), or content given as `byte[]` or a stream |
| `ArchiveWriter.copyEntry(sourceArchive, entryName)` | Copy an entry from an open reader without decompressing it |
| `ArchiveWriter.close()` | Write the central directory; safe to call twice |

Zip64 archives are read and written, and entry names are decoded as UTF-8 or CP437 according to the archive's flag. `UnsupportedEntryError`, `UnsafePathError`, and `LimitExceededError` carry the entry name in their detail, but are declared `distinct zip:Error` without the `EntryErrorDetail` detail type, pending interpreter support for a `distinct` error intersection.
