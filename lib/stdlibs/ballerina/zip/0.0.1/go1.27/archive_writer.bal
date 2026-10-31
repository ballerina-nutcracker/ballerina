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

import ballerina/file;

# Represents a new ZIP archive being created. Entries are written in the order
# they are added, and the archive is completed by `close`. One writer belongs to
# one strand.
#
# ```ballerina
# zip:ArchiveWriter writer = check new ("./reports.zip");
# check writer.addFile("./summary.pdf");
# check writer.close();
# ```
// Not `isolated`, which would say an instance may be shared across strands. Writing one entry takes
// three calls into the archive underneath, which holds one entry open at a time, and the sequence
// cannot be held under a lock: Ballerina will not have the array or the stream the content comes from
// crossing into a lock statement. The channels of `ballerina/io` are stateful in the same way and
// none of them is isolated either, so the promise is dropped rather than paid for. One writer belongs
// to one strand, and nothing here tries to make sharing one work.
public class ArchiveWriter {

    private final boolean includeSourceDirectory;

    # Creates a new ZIP archive at the given path. A file already there is left as it is,
    # unless `overwrite` is set.
    #
    # + path - Path of the ZIP file to create
    # + options - Options controlling how the archive is created
    # + return - `()` on success, or an error
    public isolated function init(string path, CompressOptions options = {}) returns Error? {
        self.includeSourceDirectory = options.includeSourceDirectory;
        check nativeCreate(self, path, options.level, options.overwrite);
    }

    # Adds a file from the local file system.
    #
    # + sourcePath - Path of the file to add
    # + entryName - Name to store the entry under, defaulting to the file name
    # + return - `()` on success, or an error
    public isolated function addFile(string sourcePath, string? entryName = ()) returns Error? {
        string name = entryName is string ? entryName : check baseNameOf(sourcePath);
        check validateEntryName(name);
        // A trailing slash is a legal entry name, and what records a directory, so the implementation
        // underneath writes one and never reads the file. That would drop the content of the file
        // without a word, which is the same mistake `addEntry` refuses content for a directory over.
        if hasSuffix(name, "/") {
            return error Error(string `cannot add entry '${name}': a name ending in '/' denotes a directory ` +
                    "and cannot hold file content");
        }
        if isDirectory(sourcePath) {
            return error FileSystemError(
                    string `cannot add file '${sourcePath}': it is a directory; use 'addDirectory' instead`);
        }
        return nativeAddFile(self, name, sourcePath);
    }

    # Adds a directory and everything inside it from the local file system.
    #
    # + sourcePath - Path of the directory to add
    # + entryName - Name for the top level of the added tree, overriding `includeSourceDirectory`
    # + return - `()` on success, or an error
    public isolated function addDirectory(string sourcePath, string? entryName = ()) returns Error? {
        if entryName is string {
            check validateEntryName(entryName);
        }
        if !isDirectory(sourcePath) {
            return error FileSystemError(
                    string `cannot add directory '${sourcePath}': no directory exists at that path`);
        }
        // Section 5.2: a name given here settles the top level whichever way `includeSourceDirectory`
        // is set, which only shapes a call that names nothing. Reading the option first instead would
        // let a caller pass a name that silently did nothing.
        string prefix = "";
        if entryName is string {
            prefix = trimTrailingSlash(entryName);
        } else if self.includeSourceDirectory {
            prefix = trimTrailingSlash(check baseNameOf(sourcePath));
        }
        if prefix != "" {
            check validateEntryName(prefix);
        }
        return self.addTree(sourcePath, prefix);
    }

    // Adds a directory and what is under it, one level at a time. The directory itself is recorded
    // only when it has a name in the archive, which is what leaves the contents of the source
    // directory at the top level when nothing names it, and what leaves an empty one of those
    // adding nothing at all. Links met on the way down
    // are skipped rather than followed: a link can point anywhere, and one pointing up its own tree
    // never ends. A `sourcePath` the caller names is used as given, links and all, since that path
    // was their choice; only the ones found while walking are skipped.
    isolated function addTree(string sourcePath, string prefix) returns Error? {
        if prefix != "" {
            check nativeAddFile(self, prefix + "/", sourcePath);
        }
        file:MetaData[]|file:Error children = file:readDir(sourcePath);
        if children is file:Error {
            return error FileSystemError(
                    string `cannot add directory '${sourcePath}': the directory could not be read`);
        }
        // Walked in name order, so that the same tree always gives the same archive.
        file:MetaData[] ordered = from file:MetaData child in children
            order by child.absPath
            select child;
        foreach file:MetaData child in ordered {
            if isSymlink(child.absPath) {
                continue;
            }
            string base = check baseNameOf(child.absPath);
            string name = prefix == "" ? base : prefix + "/" + base;
            check validateEntryName(name);
            if child.dir {
                check self.addTree(child.absPath, name);
                continue;
            }
            check nativeAddFile(self, name, child.absPath);
        }
        return;
    }

    # Adds an entry with the given content, supplied as a byte array or as a stream.
    #
    # + entryName - Name to store the entry under
    # + content - Content of the entry
    # + return - `()` on success, or an error
    public isolated function addEntry(string entryName,
            byte[]|stream<byte[], error?> content) returns Error? {
        check validateEntryName(entryName);
        if content is byte[] {
            check refuseContentForDirectory(entryName, content);
            check nativeStartEntry(self, entryName);
            // Finished whichever way the writing goes, as in `addStreamEntry`, since an entry left
            // open is one the archive cannot be completed after.
            Error? written = self.writeChunk(content);
            Error? finished = nativeFinishEntry(self);
            if written is Error {
                return written;
            }
            return finished;
        }
        // The stream is driven from here, so the caller has no way of knowing how far it was read and
        // cannot be the one to close it. It is closed on the way out whatever happened, and the error
        // from the writing is the one returned, since that is what the caller has to act on.
        // Narrowed through a typed local: the interpreter loses the stream methods of a union narrowed
        // by an early return (ballerina-nutcracker/ballerina#1428).
        stream<byte[], error?> chunks = content;
        Error? result = self.addStreamEntry(entryName, chunks);
        error? shut = chunks.close();
        if result is Error {
            return result;
        }
        if shut is error {
            return error Error(string `cannot add entry '${entryName}': the content stream could not be closed`,
                    shut);
        }
        return;
    }

    isolated function addStreamEntry(string entryName, stream<byte[], error?> content) returns Error? {
        // A directory name is settled before the entry is opened, so that content given under one is
        // refused without an entry having been written for it, which is what the `byte[]` path does.
        // The whole stream is read for that: an empty chunk is content a directory may carry, so
        // reading only as far as the first tells nothing, and a later chunk would be refused with the
        // entry already open and about to be finished.
        if hasSuffix(entryName, "/") {
            check self.refuseStreamForDirectory(entryName, content);
            check nativeStartEntry(self, entryName);
            return nativeFinishEntry(self);
        }
        record {|byte[] value;|}|error? first = content.next();
        if first is error {
            return error Error(string `cannot add entry '${entryName}': the content stream could not be read`,
                    first);
        }
        check nativeStartEntry(self, entryName);
        // The entry is finished whatever the content turns out to be, so that a writer whose caller
        // carries on after an error is not left with an entry that was opened and never closed.
        Error? written = self.writeStream(entryName, content, first);
        Error? finished = nativeFinishEntry(self);
        if written is Error {
            return written;
        }
        return finished;
    }

    isolated function refuseStreamForDirectory(string entryName, stream<byte[], error?> content) returns Error? {
        record {|byte[] value;|}|error? chunk = content.next();
        while chunk !is () {
            if chunk is error {
                return error Error(
                        string `cannot add entry '${entryName}': the content stream could not be read`, chunk);
            }
            check refuseContentForDirectory(entryName, chunk.value);
            chunk = content.next();
        }
        return;
    }

    isolated function writeStream(string entryName, stream<byte[], error?> content,
            record {|byte[] value;|}? first) returns Error? {
        record {|byte[] value;|}|error? chunk = first;
        while chunk !is () {
            if chunk is error {
                return error Error(
                        string `cannot add entry '${entryName}': the content stream could not be read`, chunk);
            }
            check self.writeChunk(chunk.value);
            chunk = content.next();
        }
        return;
    }

    isolated function writeChunk(byte[] chunk) returns Error? {
        if chunk.length() == 0 {
            return;
        }
        return nativeWriteChunk(self, chunk);
    }

    # Copies an entry from another archive without decompressing it.
    #
    # + sourceArchive - Archive to copy the entry from
    # + entryName - Name of the entry within the source archive
    # + return - `()` on success, or an error
    public isolated function copyEntry(ArchiveReader sourceArchive, string entryName) returns Error? {
        // The name is checked here as well, so that a name this library refuses to extract is not
        // written into a new archive by way of a copy.
        check validateEntryName(entryName);
        return nativeCopyEntry(self, sourceArchive, entryName);
    }

    # Completes the archive and closes the underlying file.
    #
    # + return - `()` on success, or an error
    public isolated function close() returns Error? {
        return nativeCloseWriter(self);
    }
}

// A name ending in `/` is a directory entry, which section 7.2 says holds nothing. Content given for
// one is refused rather than written, which would leave an entry saying it is a directory and holding
// bytes all the same. An empty array is how an empty directory is recorded.
isolated function refuseContentForDirectory(string entryName, byte[] content) returns Error? {
    if content.length() > 0 && hasSuffix(entryName, "/") {
        return error Error(string `cannot add entry '${entryName}': a name ending in '/' denotes a directory, ` +
                "which holds no content");
    }
    return;
}

isolated function trimTrailingSlash(string name) returns string {
    if hasSuffix(name, "/") {
        return name.substring(0, name.length() - 1);
    }
    return name;
}
