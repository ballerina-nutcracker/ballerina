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

# Creates a ZIP archive from a file or a directory.
#
# ```ballerina
# check zip:compress("./reports", "./reports.zip");
# ```
#
# + sourcePath - Path of the file or directory to archive
# + targetPath - Path of the ZIP file to create
# + options - Options controlling how the archive is created
# + return - `()` on success, or an error
public isolated function compress(string sourcePath, string targetPath,
        CompressOptions options = {}) returns Error? {
    // Checked before the writer is created, since under `overwrite` creating it truncates whatever
    // is at the target.
    check verifyTargetOutsideSource(sourcePath, targetPath);
    boolean directory = isDirectory(sourcePath);
    ArchiveWriter writer = check new (targetPath, options);
    Error? added = directory ? writer.addDirectory(sourcePath) : writer.addFile(sourcePath);
    Error? closed = writer.close();
    if added is Error {
        return added;
    }
    return closed;
}

// Section 6: an archive created inside the directory being walked would end up holding a part of
// itself, so such a call is refused before anything is written.
isolated function verifyTargetOutsideSource(string sourcePath, string targetPath) returns Error? {
    string resolvedSource = check realPath(sourcePath);
    // A target that is already there is resolved outright, so that a link standing where the archive
    // is to be written is judged by what it points at rather than by its own name. One that is not
    // there yet cannot be resolved, so its name is taken against its resolved parent.
    string resolvedTarget = pathExists(targetPath)
        ? check realPath(targetPath)
        : check joinPath(check realPath(check parentOf(targetPath)), check baseNameOf(targetPath));
    if resolvedTarget == resolvedSource {
        return error FileSystemError(
                string `cannot create the archive '${targetPath}': it is the same file as the source '${sourcePath}'`);
    }
    // Two hard links to one file are two names that resolve to themselves, so the paths above compare
    // unequal while naming the same bytes. Under `overwrite` creating the writer would truncate the
    // source, so the file system is asked whenever both names are already there.
    if pathExists(targetPath) && check isSameFile(resolvedSource, resolvedTarget) {
        return error FileSystemError(
                string `cannot create the archive '${targetPath}': it is the same file as the source '${sourcePath}'`);
    }
    if hasPrefix(resolvedTarget, withinPrefix(resolvedSource)) {
        return error FileSystemError(
                string `cannot create the archive '${targetPath}': it is inside the source directory '${sourcePath}'`);
    }
    return;
}

# Extracts every entry of a ZIP archive into a directory.
#
# ```ballerina
# check zip:decompress("./reports.zip", "./reports");
# ```
#
# + sourcePath - Path of the ZIP file to extract
# + targetPath - Path of the directory to extract into, created if it is missing
# + options - Options controlling how the archive is extracted
# + return - `()` on success, or an error
public isolated function decompress(string sourcePath, string targetPath,
        DecompressOptions options = {}) returns Error? {
    ArchiveReader reader = check new (sourcePath);
    Error? result = reader.extractAll(targetPath, options);
    Error? closed = reader.close();
    if result is Error {
        return result;
    }
    return closed;
}

# Lists the entries of a ZIP archive without extracting it.
#
# + path - Path of the ZIP file to read
# + return - Metadata of every entry, in the order stored, or an error
public isolated function listEntries(string path) returns Entry[]|Error {
    ArchiveReader reader = check new (path);
    Entry[]|Error entries = reader.entries();
    Error? closed = reader.close();
    if entries is Error {
        return entries;
    }
    if closed is Error {
        return closed;
    }
    return entries;
}
