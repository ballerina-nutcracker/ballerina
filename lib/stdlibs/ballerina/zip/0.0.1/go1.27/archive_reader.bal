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

# Represents an existing ZIP archive opened for reading.
# ```ballerina
# zip:ArchiveReader archive = check new ("./reports.zip");
# zip:Entry[] entries = check archive.entries();
# check archive.close();
# ```
public class ArchiveReader {

    # Opens an existing ZIP archive. The archive stays open until `close` is called.
    #
    # + path - Path of the ZIP file to open
    # + return - `()` on success, or an error if the file is missing or not a valid archive
    public isolated function init(string path) returns Error? {
        check nativeOpen(self, path);
    }

    # Returns the metadata of every entry, in the order stored in the archive.
    #
    # + return - Metadata of every entry, or an error
    public isolated function entries() returns Entry[]|Error {
        return nativeEntries(self);
    }

    # Returns the metadata of a single entry.
    #
    # + name - Name of the entry within the archive
    # + return - Metadata of the entry, or an error if it does not exist
    public isolated function getEntry(string name) returns Entry|Error {
        int index = check self.indexOf(name);
        return nativeEntryAt(self, index);
    }

    # Tells whether the archive holds an entry with the given name.
    #
    # + name - Name of the entry within the archive
    # + return - `true` if the archive holds such an entry, or an error
    public isolated function hasEntry(string name) returns boolean|Error {
        return check nativeIndexOf(self, name) >= 0;
    }

    // Section 4.2 has a name answered from the index the archive already holds, so the position comes
    // from there rather than from a record built for every entry to be scanned through.
    isolated function indexOf(string name) returns int|Error {
        int index = check nativeIndexOf(self, name);
        if index < 0 {
            return error EntryNotFoundError(string `the archive contains no entry named '${name}'`);
        }
        return index;
    }

    # Reads the content of an entry, as a byte array or as a stream of chunks.
    #
    # + name - Name of the entry within the archive
    # + targetType - Type to read the content into
    # + return - Content of the entry, or an error
    public isolated function readEntry(string name,
            typedesc<byte[]|stream<byte[], Error?>> targetType = <>) returns targetType|Error = external;

    # Extracts a single entry to the given file path.
    #
    # + name - Name of the entry within the archive
    # + targetPath - Path of the file to write
    # + options - Options controlling how the entry is extracted
    # + return - `()` on success, or an error
    public isolated function extractEntry(string name, string targetPath,
            DecompressOptions options = {}) returns Error? {
        check validateLimits(options.limits);
        int index = check self.indexOf(name);
        Entry entry = check nativeEntryAt(self, index);
        check validateEntryName(entry.name);
        if entry.isSymlink {
            return symlinkError(entry.name);
        }
        if entry.isDirectory {
            check createDirectories(targetPath, entry.name);
            applyModifiedTime(targetPath, entry);
            return;
        }
        // Section 4.5: the mode answers for a file already sitting there, and a directory is always
        // reused rather than written over. Asked first, so that `SKIP` does not drop the entry without
        // a word and `REPLACE` does not take an empty directory away.
        if isDirectory(targetPath) {
            return error FileSystemError(
                    string `cannot extract entry '${entry.name}': a directory already exists at '${targetPath}'`);
        }
        if pathExists(targetPath) {
            if options.fileWriteMode == FAIL_IF_EXISTS {
                return error FileSystemError(
                        string `cannot extract entry '${entry.name}': file '${targetPath}' already exists`);
            }
            if options.fileWriteMode == SKIP {
                return;
            }
            check removePath(targetPath, entry.name);
        }
        // Of the three limits only the compression ratio bears on a single entry, so extracting one
        // entry at a time cannot sidestep it.
        int written = check nativeExtractEntry(self, index, targetPath,
                byteCeiling(options.limits, 0, false), ratioCeiling(options.limits));
        if isLimitOutcome(written) {
            return limitError(entry, options.limits, written);
        }
        applyModifiedTime(targetPath, entry);
        return;
    }

    # Extracts every entry into the given directory.
    #
    # + targetPath - Path of the directory to extract into, created if it is missing
    # + options - Options controlling how the archive is extracted
    # + return - `()` on success, or an error
    public isolated function extractAll(string targetPath, DecompressOptions options = {}) returns Error? {
        check validateLimits(options.limits);
        Entry[] entries = check self.entries();
        check createDirectories(targetPath, "");
        string root = check realPath(targetPath);
        int extracted = 0;
        [string, Entry][] directories = [];
        foreach int index in 0 ..< entries.length() {
            Entry entry = entries[index];
            int? maxEntries = options.limits.maxEntries;
            if maxEntries is int && index + 1 > maxEntries {
                return error LimitExceededError(
                        string `cannot extract the archive: it contains more than the ${maxEntries} entries allowed`,
                        entryName = entry.name);
            }
            check validateEntryName(entry.name);
            string destination = check destinationPath(root, entry.name);
            // Asked before the entry is taken for a directory, since an entry can be marked as both
            // and a link is refused whatever else it claims to be. `extract` asks in the same order.
            if entry.isSymlink {
                return symlinkError(entry.name);
            }
            if entry.isDirectory {
                check createDirectoriesWithin(root, destination, entry.name);
                directories.push([destination, entry]);
                continue;
            }
            check createDirectoriesWithin(root, check parentOf(destination), entry.name);
            // Asked before the mode, for the reason given in `extractEntry`.
            if isDirectory(destination) {
                return error FileSystemError(
                        string `cannot extract entry '${entry.name}': a directory already exists at '${destination}'`);
            }
            if pathExists(destination) {
                if options.fileWriteMode == FAIL_IF_EXISTS {
                    return error FileSystemError(
                            string `cannot extract entry '${entry.name}': file '${destination}' already exists`);
                }
                if options.fileWriteMode == SKIP {
                    continue;
                }
                check removePath(destination, entry.name);
            }
            int written = check nativeExtractEntry(self, index, destination,
                    byteCeiling(options.limits, extracted, true), ratioCeiling(options.limits));
            if isLimitOutcome(written) {
                return limitError(entry, options.limits, written);
            }
            extracted += written;
            applyModifiedTime(destination, entry);
        }
        // The time of a directory is set once everything below it is written, since creating an entry
        // inside a directory changes the time of the directory itself.
        foreach [string, Entry] directory in directories {
            applyModifiedTime(directory[0], directory[1]);
        }
        return;
    }

    # Closes the archive and releases the underlying file.
    #
    # + return - `()` on success, or an error
    public isolated function close() returns Error? {
        return nativeCloseArchive(self);
    }
}
