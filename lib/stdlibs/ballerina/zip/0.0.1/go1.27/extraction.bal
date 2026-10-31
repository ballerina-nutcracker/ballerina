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

// Passed to `nativeExtractEntry` when nothing caps the bytes an entry may write.
const NO_BYTE_LIMIT = -1;

// Passed to `nativeExtractEntry` when nothing caps how far an entry may expand.
const NO_RATIO_LIMIT = 0;

// What `nativeExtractEntry` gives back in place of a byte count when a limit stops the write.
const BYTES_EXCEEDED = -1;
const RATIO_EXCEEDED = -2;

const NANOS_PER_SECOND = 1000000000;

isolated function validateLimits(ExtractionLimits limits) returns Error? {
    int? maxEntries = limits.maxEntries;
    if maxEntries is int && maxEntries <= 0 {
        return error Error("invalid 'maxEntries': the value must be positive; omit it to place no limit on " +
                "the number of entries");
    }
    int? maxTotalSize = limits.maxTotalSize;
    if maxTotalSize is int && maxTotalSize <= 0 {
        return error Error("invalid 'maxTotalSize': the value must be positive; omit it to place no limit on " +
                "the total size");
    }
    int? maxCompressionRatio = limits.maxCompressionRatio;
    if maxCompressionRatio is int && maxCompressionRatio <= 0 {
        return error Error("invalid 'maxCompressionRatio': the value must be positive; omit it to place no limit " +
                "on how far an entry may expand");
    }
    return;
}

isolated function destinationPath(string root, string name) returns string|Error {
    string joined = root;
    foreach string segment in segmentsOf(name) {
        if segment != "" {
            string|Error next = joinPath(joined, segment);
            if next is Error {
                return error FileSystemError(
                        string `cannot extract entry '${name}': the target path could not be resolved`);
            }
            joined = next;
        }
    }
    return joined;
}

// Checks that a path, with every link along it followed, sits inside the directory the caller chose.
isolated function verifyWithin(string root, string path, string entryName) returns Error? {
    string real = check realPath(path);
    if real == root || hasPrefix(real, withinPrefix(root)) {
        return;
    }
    return error UnsafePathError(
            string `cannot extract entry '${entryName}': the target path '${path}' is outside the target directory`,
            entryName = entryName);
}

isolated function createDirectories(string path, string entryName) returns Error? {
    if isDirectory(path) {
        return;
    }
    file:Error? created = file:createDir(path, file:RECURSIVE);
    if created is file:Error {
        if entryName == "" {
            return error FileSystemError(string `cannot create the target directory '${path}'`);
        }
        return error FileSystemError(
                string `cannot extract entry '${entryName}': directory '${path}' could not be created`);
    }
    return;
}

// Creates a directory and the directories above it one level at a time, checking before each level is
// created that the level above it is inside the target directory. A directory is never created
// through a link that leads out of the target directory.
isolated function createDirectoriesWithin(string root, string path, string entryName) returns Error? {
    if pathExists(path) {
        if !isDirectory(path) {
            string occupied = string `cannot extract entry '${entryName}': a file already exists at '${path}',` +
                " where a directory is required";
            return error FileSystemError(occupied);
        }
        return verifyWithin(root, path, entryName);
    }
    string parent = check parentOf(path);
    check createDirectoriesWithin(root, parent, entryName);
    check createDirectories(path, entryName);
    return verifyWithin(root, path, entryName);
}

isolated function removePath(string path, string entryName) returns Error? {
    file:Error? removed = file:remove(path);
    if removed is file:Error {
        return error FileSystemError(
                string `cannot extract entry '${entryName}': the existing file at '${path}' could not be removed`);
    }
    return;
}

// Works out how many bytes an entry may write before the total size limit is passed, which is what
// is left of the caller's budget. The compression ratio is not part of this ceiling: it is applied
// against the compressed bytes an entry is actually made of rather than the size the archive claims
// for it, which only the implementation can count. The total size limit applies to `extractAll` only.
isolated function byteCeiling(ExtractionLimits limits, int extracted, boolean withTotal) returns int {
    int? maxTotalSize = limits.maxTotalSize;
    if !withTotal || maxTotalSize is () {
        return NO_BYTE_LIMIT;
    }
    int remaining = maxTotalSize - extracted;
    return remaining < 0 ? 0 : remaining;
}

// The ceiling a write is held to. An omitted limit reaches the implementation as zero, which it takes
// as no ceiling, so the two ways of saying the same thing meet here rather than at each call.
isolated function ratioCeiling(ExtractionLimits limits) returns int {
    return limits.maxCompressionRatio ?: NO_RATIO_LIMIT;
}

// Whether what `nativeExtractEntry` gave back is a limit stopping the write rather than a byte count. Both
// outcomes are named, where jBallerina asks for a negative number, since the interpreter reports a constant
// that is never referenced as unused (ballerina-nutcracker/ballerina#1120).
isolated function isLimitOutcome(int written) returns boolean {
    return written == BYTES_EXCEEDED || written == RATIO_EXCEEDED;
}

// Turns what `nativeExtractEntry` gives back in place of a byte count into the error that belongs
// to it. The write itself says which limit stopped it, so nothing has to be inferred here.
isolated function limitError(Entry entry, ExtractionLimits limits, int outcome) returns LimitExceededError {
    if outcome == BYTES_EXCEEDED {
        int maxTotalSize = limits.maxTotalSize ?: 0;
        string total = string `cannot extract entry '${entry.name}': the extracted size would exceed ` +
                string `the ${maxTotalSize} bytes allowed`;
        return error LimitExceededError(total, entryName = entry.name);
    }
    int ratio = ratioCeiling(limits);
    string tooBig = string `cannot extract entry '${entry.name}': it expands to more than ${ratio} ` +
            "times its compressed size";
    return error LimitExceededError(tooBig, entryName = entry.name);
}

isolated function symlinkError(string name) returns UnsupportedEntryError {
    return error UnsupportedEntryError(string `cannot extract entry '${name}': symbolic links are not supported`,
            entryName = name);
}

// Gives an extracted file the time the archive records for it. A recorded mode is not applied; the
// file keeps the permissions the platform gives a new file, as section 4.4 of the specification says.
isolated function applyModifiedTime(string path, Entry entry) {
    [int, decimal] modified = entry.modifiedTime;
    nativeSetModifiedTime(path, modified[0], <int>(modified[1] * <decimal>NANOS_PER_SECOND));
}
