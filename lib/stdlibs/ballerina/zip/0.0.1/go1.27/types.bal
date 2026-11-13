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

import ballerina/time;

# Compression method used for an entry.
public enum CompressionMethod {
    # Stored without compression
    STORE,
    # Compressed with the DEFLATE algorithm
    DEFLATE,
    # Stored by a method this library cannot decompress
    OTHER
}

# Level of compression applied when creating an archive.
public enum CompressionLevel {
    # No compression, entries are stored as they are
    NONE,
    # Fastest compression, largest output
    FASTEST,
    # Balanced compression, the default
    DEFAULT,
    # Best compression, slowest
    BEST
}

# What to do when a file already exists at the target of an extraction.
public enum FileWriteMode {
    # Stop with an error
    FAIL_IF_EXISTS,
    # Replace the existing file
    REPLACE,
    # Leave the existing file untouched and continue
    SKIP
}

# Metadata of a single file or directory inside a ZIP archive.
#
# + name - Path of the entry within the archive, separated by `/`
# + isDirectory - Whether the entry is a directory
# + isSymlink - Whether the entry is a symbolic link
# + uncompressedSize - Size of the content in bytes, before compression
# + compressedSize - Size of the content in bytes, as stored in the archive
# + method - Compression method used for the entry
# + modifiedTime - Last modified time of the entry
# + crc32 - CRC-32 checksum of the uncompressed content
# + comment - Comment attached to the entry
# + unixMode - Unix mode as recorded in the archive, when it records one
public type Entry record {|
    string name;
    boolean isDirectory;
    boolean isSymlink;
    int uncompressedSize;
    int compressedSize;
    CompressionMethod method;
    time:Utc modifiedTime;
    int crc32;
    string comment?;
    int unixMode?;
|};

# Limits applied while extracting, to guard against archives that expand to an
# unreasonable size. An absent field means no limit.
#
# + maxEntries - Maximum number of entries to extract
# + maxTotalSize - Maximum total uncompressed size in bytes
# + maxCompressionRatio - Maximum allowed uncompressed to compressed size ratio for one entry
public type ExtractionLimits record {|
    int maxEntries?;
    int maxTotalSize?;
    int maxCompressionRatio?;
|};

# Options for creating a ZIP archive.
#
# + level - Compression level applied to the entries
# + includeSourceDirectory - Whether the source directory itself becomes the root entry
# + overwrite - Whether a file already at the target path is replaced
public type CompressOptions record {|
    CompressionLevel level = DEFAULT;
    boolean includeSourceDirectory = true;
    boolean overwrite = false;
|};

# Options for extracting a ZIP archive.
#
# + fileWriteMode - What to do when a file already exists in the target directory
# + limits - Limits applied while extracting
public type DecompressOptions record {|
    FileWriteMode fileWriteMode = FAIL_IF_EXISTS;
    ExtractionLimits limits = {};
|};
