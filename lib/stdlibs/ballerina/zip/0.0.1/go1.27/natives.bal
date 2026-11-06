// Copyright (c) 2026 WSO2 LLC. (http://www.wso2.com).
//
// WSO2 LLC. licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except
// in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

// This is the whole surface the module depends on from the underlying ZIP
// implementation. Everything else - path safety, extraction limits, walking
// directories, mapping errors - stays in Ballerina, so replacing the
// implementation means porting only this file.
//
// The open archive is kept by the native code on the `ArchiveReader` or
// `ArchiveWriter` it belongs to, so each function is handed that object.

isolated function nativeOpen(ArchiveReader reader, string path) returns Error? = external;

isolated function nativeEntries(ArchiveReader reader) returns Entry[]|Error = external;

// The position of the first entry with the given name, or a negative number when there is none. The
// index the implementation already keeps answers this, so asking a name of the archive does not mean
// building a record for every entry it holds.
isolated function nativeIndexOf(ArchiveReader reader, string name) returns int|Error = external;

isolated function nativeEntryAt(ArchiveReader reader, int index) returns Entry|Error = external;

// Writes the content of the entry at `entryIndex` to `targetPath` and returns the number of bytes
// written. The entry is named by its position, so that the caller decides which of two entries with
// the same name is written. Writing stops before `byteLimit` is passed and `BYTES_EXCEEDED` is
// returned instead; a negative `byteLimit` means no ceiling. It also stops, with `RATIO_EXCEEDED`,
// once the entry has produced more than `maxCompressionRatio` bytes for every compressed byte taken
// from the archive; a ratio of zero or less means no ceiling. The ratio is counted there rather than
// here because only the implementation knows how much compressed input an entry has consumed, and
// the size the archive claims for it can be untrue. Passing a ceiling is reported rather than raised
// as an error, so that the `LimitExceededError` and its message stay in Ballerina.
isolated function nativeExtractEntry(ArchiveReader reader, int entryIndex, string targetPath,
        int byteLimit, int maxCompressionRatio) returns int|Error = external;

isolated function nativeCloseArchive(ArchiveReader reader) returns Error? = external;

// Creates the archive file. Unless `overwrite` is set, a file already at the path is left as it is
// and a `FileSystemError` comes back. Whether the path is free is decided by the operation that
// creates the file rather than by a check made before it, so nothing can appear in between.
isolated function nativeCreate(ArchiveWriter writer, string path, string level, boolean overwrite)
        returns Error? = external;

// Writes the content of the file at `sourcePath` under `entryName`, or a directory entry holding
// nothing when the name ends in `/`. The time of the source is recorded with it, and its permissions
// where the platform keeps any. Which files are walked, and which of them are skipped, is decided by
// the caller.
isolated function nativeAddFile(ArchiveWriter writer, string entryName, string sourcePath) returns Error? = external;

isolated function nativeStartEntry(ArchiveWriter writer, string entryName) returns Error? = external;

isolated function nativeWriteChunk(ArchiveWriter writer, byte[] content) returns Error? = external;

isolated function nativeFinishEntry(ArchiveWriter writer) returns Error? = external;

// Writes the first entry named `entryName` of `sourceArchive` into the archive being written, exactly
// as it is stored there. The content is never decoded, so an entry this library cannot read is
// carried across all the same, and the compression of the writer does not apply to it.
isolated function nativeCopyEntry(ArchiveWriter writer, ArchiveReader sourceArchive, string entryName)
        returns Error? = external;

isolated function nativeCloseWriter(ArchiveWriter writer) returns Error? = external;

// Builds the errors the native code returns, since the distinct error types can only be constructed
// here. `entryName` is recorded in the detail of the error types that carry one.
isolated function nativeError(string kind, string message, string entryName) returns Error {
    if kind == "InvalidArchiveError" {
        return error InvalidArchiveError(message);
    }
    if kind == "EntryNotFoundError" {
        return error EntryNotFoundError(message);
    }
    if kind == "UnsupportedEntryError" {
        return error UnsupportedEntryError(message, entryName = entryName);
    }
    if kind == "FileSystemError" {
        return error FileSystemError(message);
    }
    return error Error(message);
}

// Builds an `Entry` for the native code, which cannot construct a value of a record type declared here.
isolated function newEntry(string name, boolean isDirectory, boolean isSymlink, int uncompressedSize,
        int compressedSize, string method, int seconds, decimal fraction, int crc32, string? comment,
        int? unixMode) returns Entry {
    Entry entry = {
        name,
        isDirectory,
        isSymlink,
        uncompressedSize,
        compressedSize,
        method: <CompressionMethod>method,
        modifiedTime: [seconds, fraction],
        crc32
    };
    if comment is string {
        entry.comment = comment;
    }
    if unixMode is int {
        entry.unixMode = unixMode;
    }
    return entry;
}

isolated function newEntries() returns Entry[] {
    return [];
}
