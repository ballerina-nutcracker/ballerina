// Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
//
// WSO2 LLC. licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except
// in compliance with the License.
//
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

import ballerina/os;
import ballerina/time;

// ---------------------------------------------------------------------------
// Error types  (from errors.bal)
// ---------------------------------------------------------------------------

// Represents file system related errors.
public type Error error;

// Represents an error that occurs when a file system operation is denied due to invalidity.
public type InvalidOperationError Error;

// Represents an error that occurs when a file system operation is denied due to the absence of file permission.
public type PermissionError Error;

// Represents an error that occurs when a file system operation fails.
public type FileSystemError Error;

// Represents an error which occurs when the file/directory does not exist in the given file path.
public type FileNotFoundError Error;

// Represents an error which occurs when the file in the given file path is not a symbolic link.
public type NotLinkError Error;

// Represents an IO error which occurs when trying to access the file in the given file path.
public type IOError Error;

// Represents a security error which occurs when trying to access the file in the given file path.
public type SecurityError Error;

// Represents an error which occurs when the given file path is invalid.
public type InvalidPathError Error;

// Represents an error which occurs when the given pattern is not a valid file path pattern.
public type InvalidPatternError Error;

// Represents an error which occurs when the given target file path cannot be derived relative to the base file path.
public type RelativePathError Error;

// Represents an error which occurs in the UNC path.
public type UNCPathError Error;

// Represents a generic error for the file path.
public type GenericError Error;

// ---------------------------------------------------------------------------
// Common types and enumerations  (from common.bal)
// ---------------------------------------------------------------------------

// Represents an event which will trigger when there is a change to the listening directory.
//
// + name - Absolute file URI for triggered event
// + operation - Triggered event action. This can be create, delete or modify
public type FileEvent record {|
    string name;
    string operation;
|};

// Represents the options that can be passed to the `normalizePath` function.
//
// + CLEAN - Get the shortest path name equivalent to the given path by eliminating multiple separators, '.', and '..'
// + SYMLINK - Evaluate a symlink
// + NORMCASE - Normalize the case of a pathname. On windows, all the characters are converted to lowercase and "/" is
// converted to "\".
public enum NormOption {
    CLEAN,
    SYMLINK,
    NORMCASE
}

// Represents options that can be used when creating or removing directories.
//
// + RECURSIVE - Create non-existing parent directories or remove all the files inside the given directory
// + NON_RECURSIVE - Create/remove only the given directory
public enum DirOption {
    RECURSIVE,
    NON_RECURSIVE
}

// Represents the options that can be passed to the test function.
//
// + EXISTS - Test whether a file path exists
// + IS_DIR - Test whether a file path is a directory
// + IS_SYMLINK - Test whether a file path is a symlink
// + READABLE - Test whether a file path is readable
// + WRITABLE - Test whether a file path is writable
public enum TestOption {
    EXISTS,
    IS_DIR,
    IS_SYMLINK,
    READABLE,
    WRITABLE
}

// Represents options that can be used when copying files/directories.
//
// + REPLACE_EXISTING - Replace the target path if it already exists
// + COPY_ATTRIBUTES - Copy the file attributes as well to the target
// + NO_FOLLOW_LINKS - If source is a symlink, only the link is copied, not the target of the link
public enum CopyOption {
    REPLACE_EXISTING,
    COPY_ATTRIBUTES,
    NO_FOLLOW_LINKS
}

// ---------------------------------------------------------------------------
// Metadata record  (from meta_data.bal)
// ---------------------------------------------------------------------------

// Metadata record contains metadata information of a file.
// This record is returned by getMetaData function.
//
// + absPath - Absolute path of the file
// + size - Size of the file (in bytes)
// + modifiedTime - The last modified time of the file
// + dir - Whether the file is a directory or not
// + readable - Whether the file is readable or not
// + writable - Whether the file is writable or not
public type MetaData record {|
    string absPath;
    int size;
    time:Utc modifiedTime;
    boolean dir;
    boolean readable;
    boolean writable;
|};

// ---------------------------------------------------------------------------
// Path utilities  (from path.bal)
// ---------------------------------------------------------------------------

final boolean isWindows = os:getEnv("OS") != "";

isolated function _initPathSep() returns string {
    if isWindows {
        return "\\";
    }
    return "/";
}

isolated function _initPathListSep() returns string {
    if isWindows {
        return ";";
    }
    return ":";
}

public final string pathSeparator = _initPathSep();
public final string pathListSeparator = _initPathListSep();

type RootInfo record {|
    string root;
    int offset;
|};

# Retrieves the absolute path from the provided location.
#
# + path - String value of the file path free from potential malicious codes
# + return - The absolute path reference or else a `file:Error` if the path cannot be derived
public isolated function getAbsolutePath(string path) returns string|Error = external;

# Reports whether the path is absolute.
# A path is absolute if it is independent of the current directory.
# On Unix, a path is absolute if it starts with the root.
# On Windows, a path is absolute if it has a prefix and starts with the root: c:\windows.
#
# + path - String value of the file path
# + return - `true` if path is absolute, `false` otherwise, or else an `file:Error`
#            occurred if the path is invalid
public isolated function isAbsolutePath(string path) returns boolean|Error {
    if path.length() <= 0 {
        return false;
    }
    if isWindows {
        return check getVolumnNameLength(path) > 0;
    } else {
        return check charAt(path, 0) == "/";
    }
}

# Retrieves the base name of the file from the provided location,
# which is the last element of the path.
# Trailing path separators are removed before extracting the last element.
#
# + path - String value of file path
# + return - The name of the file or else a `file:Error` if the path is invalid
public isolated function basename(string path) returns string|Error {
    string validatedPath = check parse(path);
    int[] offsetIndexes = check getOffsetIndexes(validatedPath);
    int count = offsetIndexes.length();
    if count == 0 {
        return "";
    }
    if (count == 1 && validatedPath.length() > 0) {
        if !(check isAbsolutePath(validatedPath)) {
            return validatedPath;
        }
    }
    int lastOffset = offsetIndexes[count - 1];
    return validatedPath.substring(lastOffset, validatedPath.length());
}

# Returns the enclosing parent directory.
# If the path is empty, parent returns ".".
# The returned path does not end in a separator unless it is the root directory.
#
# + path - String value of the file/directory path
# + return - Path of the parent directory or else a `file:Error`
#            if an error occurred while getting the parent directory
public isolated function parentPath(string path) returns string|Error {
    string validatedPath = check parse(path);
    int[] offsetIndexes = check getOffsetIndexes(validatedPath);
    int count = offsetIndexes.length();
    if count == 0 {
        return "";
    }
    int len = offsetIndexes[count-1] - 1;
    if len < 0 {
        return "";
    }
    RootInfo rootInfo = check getRoot(validatedPath);
    string root = rootInfo.root;
    int offset = rootInfo.offset;
    if (len < offset) {
        return root;
    }
    return validatedPath.substring(0, len);
}

# Normalizes a path value.
#
# + path - String value of the file path
# + option - Normalization option. Supported options are,
#  `CLEAN` - Get the shortest path name equivalent to the given path by eliminating multiple separators, '.', and '..',
#  `SYMLINK` - Evaluate a symlink,
#  `NORMCASE` - Normalize the case of a pathname. On windows, all the characters are converted to lowercase and "/" is
#               converted to "\\".
# + return - Normalized file path or else a `file:Error` if the path is invalid
public isolated function normalizePath(string path, NormOption option) returns string|Error {
    match option {

        CLEAN => {
            string validatedPath = check parse(path);
            int[] offsetIndexes = check getOffsetIndexes(validatedPath);
            int count = offsetIndexes.length();
            if (count == 0 || isEmpty(validatedPath)) {
                return validatedPath;
            }

            RootInfo rootInfo = check getRoot(validatedPath);
            string root = rootInfo.root;
            int offset = rootInfo.offset;
            string c0 = check charAt(path, 0);

            int i = 0;
            string[] parts = [];
            boolean[] ignore = [];
            boolean[] parentRef = [];
            int remaining = count;
            while i < count {
                int begin = offsetIndexes[i];
                int length;
                ignore[i] = false;
                parentRef[i] = false;
                if i == (count - 1) {
                    length = validatedPath.length() - begin;
                    parts[i] = validatedPath.substring(begin, validatedPath.length());
                } else {
                    length = offsetIndexes[i + 1] - begin - 1;
                    parts[i] = validatedPath.substring(begin, offsetIndexes[i + 1] - 1);
                }
                if (check charAt(validatedPath, begin) == ".") {
                    if length == 1 {
                        ignore[i] = true;
                        remaining = remaining - 1;
                    } else if (length == 2 && check charAt(validatedPath, begin + 1) == ".") {
                        parentRef[i] = true;
                        int j = i - 1;
                        boolean hasPrevious = false;
                        while j >= 0 {
                            if (ignore.length() > 0 && !parentRef[j] && !ignore[j]) {
                                ignore[j] = true;
                                remaining = remaining - 1;
                                hasPrevious = true;
                                break;
                            }
                            j = j - 1;
                        }
                        if (hasPrevious || (offset > 0) || isSlash(c0)) {
                            ignore[i] = true;
                            remaining = remaining - 1;
                        }
                    }
                }
                i = i + 1;
            }

            if remaining == count {
                return validatedPath;
            }

            if remaining == 0 {
                return root;
            }

            string normalizedPath = "";
            if root != "" {
                normalizedPath = normalizedPath + root;
            }
            i = 0;
            while i < count {
                if (!ignore[i] && (offset <= offsetIndexes[i])) {
                    normalizedPath = normalizedPath + parts[i] + pathSeparator;
                }
                i = i + 1;
            }
            return parse(normalizedPath);
        }

        SYMLINK => {
            return resolve(path);
        }

        NORMCASE => {
            if isWindows {
                string lowerCasePath = path.toLowerAscii();
                return replaceFwdSlashWithBackslash(lowerCasePath);
            }
            return path;
        }
    }
}

# Splits a list of paths joined by the OS-specific path separator.
#
# + path - String value of the file path
# + return - String array of the path components or else a `file:Error` if the path is invalid
public isolated function splitPath(string path) returns string[]|Error {
    string validatedPath = check parse(path);
    int[] offsetIndexes = check getOffsetIndexes(validatedPath);
    int count = offsetIndexes.length();

    string[] parts = [];
    int i = 0;
    while i < count {
        int begin = offsetIndexes[i];
        if i == (count - 1) {
            parts[i] = check parse(validatedPath.substring(begin, validatedPath.length()));
        } else {
            parts[i] = check parse(validatedPath.substring(begin, offsetIndexes[i + 1] - 1));
        }
        i = i + 1;
    }
    return parts;
}

# Joins any number of path elements into a single path.
#
# + parts - String values of the file path parts
# + return - String value of the file path or else a `file:Error` if the parts are invalid
public isolated function joinPath(string... parts) returns string|Error {
    if isWindows {
        return check buildWindowsPath(parts);
    } else {
        return check buildUnixPath(parts);
    }
}

# Returns a relative path, which is logically equivalent to the target path when joined to the base path with an
# intervening separator.
# An error is returned if the target path cannot be made relative to the base path.
#
# + base - String value of the base file path
# + target - String value of the target file path
# + return - The target path relative to the base path, or else an
#            `file:Error` if target path cannot be made relative to the base path
public isolated function relativePath(string base, string target) returns string|Error {
    string cleanBase = check normalizePath(base, CLEAN);
    string cleanTarget = check normalizePath(target, CLEAN);
    if isSamePath(cleanBase, cleanTarget) {
        return ".";
    }
    RootInfo baseInfo = check getRoot(cleanBase);
    string baseRoot = baseInfo.root;
    int baseOffset = baseInfo.offset;
    RootInfo targetInfo = check getRoot(cleanTarget);
    string targetRoot = targetInfo.root;
    int targetOffset = targetInfo.offset;
    if !isSamePath(baseRoot, targetRoot) {
        return error RelativePathError("Can't make: " + target + " relative to " + base);
    }
    int b0 = baseOffset;
    int bi = baseOffset;
    int t0 = targetOffset;
    int ti = targetOffset;
    int bl = cleanBase.length();
    int tl = cleanTarget.length();
    while true {
        while bi < bl {
            if isSlash(check charAt(cleanBase, bi)) {
                break;
            }
            bi = bi + 1;
        }
        while ti < tl {
            if isSlash(check charAt(cleanTarget, ti)) {
                break;
            }
            ti = ti + 1;
        }
        if !isSamePath(cleanBase.substring(b0, bi), cleanTarget.substring(t0, ti)) {
            break;
        }
        if bi < bl {
           bi = bi + 1;
        }
        if ti < tl {
            ti = ti + 1;
        }
        b0 = bi;
        t0 = ti;
    }
    if cleanBase.substring(b0, bi) == ".." {
        return error RelativePathError("Can't make: " + target + " relative to " + base);
    }
    if b0 != bl {
        string remainder = cleanBase.substring(b0, bl);
        int[] offsets = check getOffsetIndexes(remainder);
        int noSeparators = offsets.length() - 1;
        string relativePath = "..";
        int i = 0;
        while i < noSeparators {
            relativePath = relativePath + pathSeparator + "..";
            i = i + 1;
        }
        if t0 != tl {
            relativePath = relativePath + pathSeparator + cleanTarget.substring(t0, tl);
        }
        return relativePath;
    }
    return cleanTarget.substring(t0, tl);
}

# Returns the file path after the evaluation of any symbolic links.
# If the path is relative, the result will be relative to the current directory
# unless one of the components is an absolute symbolic link.
#
# + path - Security-validated string value of the file path
# + return - Resolved file path or else a `file:Error` if the path is invalid
isolated function resolve(string path) returns string|Error = external;

isolated function parse(string input) returns string|Error {
    if input.length() <= 0 {
        return input;
    }
    if isWindows {
        RootInfo ri = check getRoot(input);
        string root = ri.root;
        int offset = ri.offset;
        return root + check parseWindowsPath(input, offset);
    } else {
        int n = input.length();
        string prevC = "";
        int i = 0;
        while i < n {
            string c = check charAt(input, i);
            if ((c == "/") && (prevC == "/")) {
                return parsePosixPath(input, i - 1);
            }
            prevC = c;
            i = i + 1;
        }
        if prevC == "/" {
            return parsePosixPath(input, n - 1);
        }
        return input;
    }
}

isolated function getRoot(string input) returns RootInfo|Error {
    if isWindows {
        return getWindowsRoot(input);
    } else {
        return getUnixRoot(input);
    }
}

isolated function isSlash(string c) returns boolean {
    if isWindows {
        return isWindowsSlash(c);
    } else {
        return isPosixSlash(c);
    }
}

isolated function nextNonSlashIndex(string path, int offset, int end) returns int|Error {
    int off = offset;
    while off < end {
        if !isSlash(check charAt(path, off)) {
            break;
        }
        off = off + 1;
    }
    return off;
}

isolated function nextSlashIndex(string path, int offset, int end) returns int|Error {
    int off = offset;
    while off < end {
        if isSlash(check charAt(path, off)) {
            break;
        }
        off = off + 1;
    }
    return off;
}

isolated function isLetter(string c) returns boolean {
    return (c >= "a" && c <= "z") || (c >= "A" && c <= "Z");
}

isolated function isUNC(string path) returns boolean|Error {
    return check getVolumnNameLength(path) > 2;
}

isolated function isEmpty(string path) returns boolean {
    return path.length() == 0;
}

isolated function getOffsetIndexes(string path) returns int[]|Error {
    if isWindows {
        return check getWindowsOffsetIndex(path);
    } else {
        return check getUnixOffsetIndex(path);
    }
}

isolated function charAt(string input, int index) returns string|Error = external;

isolated function isSamePath(string base, string target) returns boolean {
    if isWindows {
        return base.equalsIgnoreCaseAscii(target);
    } else {
        return base == target;
    }
}

isolated function replaceFwdSlashWithBackslash(string path) returns string {
    string result = "";
    int i = 0;
    int n = path.length();
    while i < n {
        string c = path.substring(i, i + 1);
        if c == "/" {
            result = result + "\\";
        } else {
            result = result + c;
        }
        i = i + 1;
    }
    return result;
}

// ---------------------------------------------------------------------------
// Unix path utilities  (from unix_path.bal)
// ---------------------------------------------------------------------------

isolated function buildUnixPath(string[] parts) returns string|Error {
    int count = parts.length();
    if count <= 0 {
        return "";
    }
    int i = 0;
    while i < count {
        if parts[i] != "" {
            break;
        }
        i = i + 1;
    }
    if i == count {
        return "";
    }
    string finalPath = parts[i];
    i = i + 1;
    while (i < count) {
        finalPath = finalPath + "/" + parts[i];
        i = i + 1;
    }
    return parse(finalPath);
}

isolated function getUnixRoot(string input) returns RootInfo|Error {
    int length = input.length();
    int offset = 0;
    string root = "";
    if (length > 0 && isSlash(check charAt(input, 0))) {
        root = pathSeparator;
        offset = 1;
    }
    return {root: root, offset: offset};
}

isolated function getUnixOffsetIndex(string path) returns int[]|Error {
    int[] offsetIndexes = [];
    int index = 0;
    int count = 0;
    if isEmpty(path) {
        offsetIndexes[count] = 0;
        count = count + 1;
    } else {
        while index < path.length() {
            string cn = check charAt(path, index);
            if (cn == "/") {
                index = index + 1;
            } else {
                offsetIndexes[count] = index;
                count = count + 1;
                index = index + 1;
                while(index < path.length()) {
                    if ((check charAt(path, index)) == "/") {
                        break;
                    }
                    index = index + 1;
                }
            }
        }
    }
    return offsetIndexes;
}

isolated function isPosixSlash(string|byte c) returns boolean {
    return c == "/";
}

isolated function parsePosixPath(string input, int off) returns string|Error {
    int n = input.length();
    while n > 0 {
        string cn = check charAt(input, n-1);
        if(cn != "/") {
            break;
        }
        n = n-1;
    }
    if n == 0 {
        return "/";
    }
    string normalizedPath = "";
    if off > 0 {
        normalizedPath = normalizedPath + input.substring(0, off);
    }
    string prevC = "";
    int i = off;
    while  i < n {
        string c = check charAt(input, i);
        if c == "/" && prevC == "/" {
            i = i + 1;
            continue;
        }
        normalizedPath = normalizedPath + c;
        prevC = c;
        i = i + 1;
    }
    return normalizedPath;
}

// ---------------------------------------------------------------------------
// Windows path utilities  (from windows_path.bal)
// ---------------------------------------------------------------------------

isolated function buildWindowsPath(string[] parts) returns string|Error {
    int count = parts.length();
    if count <= 0 {
        return "";
    }
    int i = 0;
    while i < count {
        if parts[i] != "" {
            break;
        }
        i = i + 1;
    }
    if i == count {
        return "";
    }
    string firstNonEmptyPart = parts[i];

    if firstNonEmptyPart.length() == 2 {
        string c0 = check charAt(firstNonEmptyPart, 0);
        string c1 = check charAt(firstNonEmptyPart, 1);
        if (isLetter(c0) && c1.equalsIgnoreCaseAscii(":")) {
            i = i + 1;
            while (i < count) {
                if (parts[i] != "") {
                    break;
                }
                i = i + 1;
            }
            string tail;
            if (i < count) {
                tail = parts[i];
                i = i + 1;
            } else {
                return normalizePath(firstNonEmptyPart, CLEAN);
            }

            while i < count {
                if (parts[i] != "") {
                    tail = tail + "\\" + parts[i];
                }
                i = i + 1;
            }
            return firstNonEmptyPart + check normalizePath(tail, CLEAN);
        }
    }

    string head = firstNonEmptyPart;
    if check isUNC(head) {
        string finalPath = firstNonEmptyPart;
        i = i + 1;
        while i < count {
            finalPath = finalPath + "\\" + parts[i];
            i = i + 1;
        }
        return normalizePath(finalPath, CLEAN);
    }

    i = i + 1;
    string tail;
    if i < count {
        tail = parts[i];
        i = i + 1;
    } else {
        return normalizePath(firstNonEmptyPart, CLEAN);
    }

    while i < count {
        if parts[i] != "" {
            tail = tail + "\\" + parts[i];
        }
        i = i + 1;
    }
    string normalizedHead = check normalizePath(head, CLEAN);
    string normalizedTail = check normalizePath(tail, CLEAN);

    if tail == "" {
        return normalizedHead;
    }
    int index = check nextNonSlashIndex(normalizedTail, 0, normalizedTail.length());
    if index > 0 {
        normalizedTail = normalizedTail.substring(index, normalizedTail.length());
    }

    if check charAt(normalizedHead, normalizedHead.length() - 1) == pathSeparator {
        return normalizedHead + normalizedTail;
    }
    return normalizedHead + pathSeparator + normalizedTail;
}

isolated function getWindowsRoot(string input) returns RootInfo|Error {
    int length = input.length();
    int offset = 0;
    string root = "";
    if length > 1 {
        string c0 = check charAt(input, 0);
        string c1 = check charAt(input, 1);
        int next = 2;
        if isSlash(c0) && isSlash(c1) {
            boolean unc = check isUNC(input);
            if !unc {
                return error UNCPathError("Invalid UNC path: " + input);
            }
            offset = check nextNonSlashIndex(input, next, length);
            next = check nextSlashIndex(input, offset, length);
            if offset == next {
                return error UNCPathError("Hostname is missing in UNC path: " + input);
            }
            string host = input.substring(offset, next);
            offset = check nextNonSlashIndex(input, next, length);
            next = check nextSlashIndex(input, offset, length);
            if offset == next {
                return error UNCPathError("Sharename is missing in UNC path: " + input);
            }
            root = "\\\\" + host + "\\" + input.substring(offset, next) + "\\";
            offset = next;
        } else if isSlash(c0) {
            root = "\\";
            offset = 1;
        } else {
            if isLetter(c0) && c1.equalsIgnoreCaseAscii(":") {
                if (input.length() > 2 && isSlash(check charAt(input, 2))) {
                    string c2 = check charAt(input, 2);
                    if c2 == "\\" {
                        root = input.substring(0, 3);
                    } else {
                        root = input.substring(0, 2) + "\\";
                    }
                    offset = 3;
                } else {
                    root = input.substring(0, 2);
                    offset = 2;
                }
            }
        }
    } else if length > 0 && isSlash(check charAt(input, 0)) {
            root = "\\";
            offset = 1;
    }
    return {root: root, offset: offset};
}

isolated function getWindowsOffsetIndex(string path) returns int[]|Error {
    int[] offsetIndexes = [];
    int index = 0;
    int count = 0;
    if isEmpty(path) {
        offsetIndexes[count] = 0;
        count = count + 1;
    } else {
        RootInfo wr = check getWindowsRoot(path);
        index = wr.offset;
        while(index < path.length()) {
            string cn = check charAt(path, index);
            if cn == "/" || cn == "\\" {
                index = index + 1;
            } else {
                offsetIndexes[count] = index;
                count = count + 1;
                index = index + 1;
                while index < path.length() {
                    string value = check charAt(path, index);
                    if (value == "/" || value == "\\") {
                        break;
                    }
                    index = index + 1;
                }
            }
        }
    }
    return offsetIndexes;
}

isolated function isWindowsSlash(string c) returns boolean {
    return c == "\\" || c == "/";
}

isolated function getVolumnNameLength(string path) returns int|Error {
    if path.length() < 2 {
        return 0;
    }
    string c0 = check charAt(path, 0);
    string c1 = check charAt(path, 1);
    if isLetter(c0) && c1 == ":" {
        return 2;
    }
    int size = path.length();
    if size < 5 {
        return 0;
    }
    string c2 = check charAt(path, 2);
    if (size >= 5 && isSlash(c0) && isSlash(c1) && !isSlash(c2) && c2 != ".") {
        int n = 3;
        while n < size-1 {
            string cn = check charAt(path, n);
            if isSlash(cn) {
                n = n + 1;
                cn = check charAt(path, n);
                if !isSlash(cn) {
                    if cn == "." {
                        break;
                    }

                    while n < size {
                        if isSlash(cn) {
                            break;
                        }
                        n = n + 1;
                    }
                    return n;
                }
                break;
            }
            n = n + 1;
        }
    }
    return 0;
}

isolated function parseWindowsPath(string path, int off) returns string|Error {
    string normalizedPath = "";
    int length = path.length();
    int offset = check nextNonSlashIndex(path, off, length);
    int startIndex = offset;
    while offset < length {
        string c = check charAt(path, offset);
        if isSlash(c) {
            normalizedPath = normalizedPath + path.substring(startIndex, offset);
            offset = check nextNonSlashIndex(path, offset, length);
            if (offset != length) {
                normalizedPath = normalizedPath + "\\";
            }
            startIndex = offset;
        } else {
            offset = offset + 1;
        }
    }
    if startIndex != offset {
        normalizedPath = normalizedPath + path.substring(startIndex, offset);
    }
    return normalizedPath;
}

// ---------------------------------------------------------------------------
// File system operations  (from file.bal)
// ---------------------------------------------------------------------------

# Returns the current working directory.
#
# + return - Current working directory or else an empty string if the current working directory cannot be determined
public isolated function getCurrentDir() returns string = external;

# Creates a new directory with the specified name.
#
# + dir - Directory name
# + option - Indicates whether the `createDir` should create non-existing parent directories. The default is only to
#            create the given current directory.
# + return - A `file:Error` if the directory creation failed
public isolated function createDir(string dir, DirOption option = NON_RECURSIVE) returns Error? = external;

# Removes the specified file or directory.
#
# + path - String value of the file/directory path
# + option - Indicates whether the `remove` should recursively remove all the files inside the given directory
# + return - An `file:Error` if failed to remove
public isolated function remove(string path, DirOption option = NON_RECURSIVE) returns Error? = external;

# Renames (moves) the old path with the new path.
# If the new path already exists and it is not a directory, this replaces the file.
#
# + oldPath - String value of the old file path
# + newPath - String value of the new file path
# + return - An `file:Error` if failed to rename
public isolated function rename(string oldPath, string newPath) returns Error? = external;

# Creates a file in the specified file path.
# Truncates if the file already exists in the given path.
#
# + path - String value of the file path
# + return - A `file:Error` if file creation failed
public isolated function create(string path) returns Error? = external;

isolated function getRawMetaData(string path) returns MetaData|Error = external;

# Returns the metadata information of the file specified in the file path.
#
# + path - String value of the file path.
# + return - The `MetaData` instance with the file metadata or else a `file:Error`
public isolated function getMetaData(string path) returns MetaData|Error {
    return getRawMetaData(path);
}

isolated function readDirRaw(string path) returns MetaData[]|Error = external;

# Reads the directory and returns a list of metadata of files and directories
# inside the specified directory.
#
# + path - String value of the directory path
# + return - The `MetaData` array or else a `file:Error` if there is an error
public isolated function readDir(string path) returns MetaData[]|Error {
    return readDirRaw(path);
}

# Copy the file/directory in the old path to the new path.
#
# + sourcePath - String value of the old file path
# + destinationPath - String value of the new file path
# + options - Parameter to denote how the copy operation should be done. Supported options are,
#  `REPLACE_EXISTING` - Replace the target path if it already exists,
#  `COPY_ATTRIBUTES` - Copy the file attributes as well to the target,
#  `NO_FOLLOW_LINKS` - If source is a symlink, only the link is copied, not the target of the link.
# + return - An `file:Error` if failed to copy
public isolated function copy(string sourcePath, string destinationPath, CopyOption... options) returns Error? = external;

# Creates a temporary file.
#
# + suffix - Optional file suffix
# + prefix - Optional file prefix
# + dir - The directory path where the temp file should be created. If not specified,
#         temp file will be created in the default temp directory of the OS.
# + return - Temporary file path or else a `file:Error` if there is an error
public isolated function createTemp(string? suffix = (), string? prefix = (), string? dir = ()) returns string|Error = external;

# Creates a temporary directory.
#
# + suffix - Optional directory suffix
# + prefix - Optional directory prefix
# + dir - The directory path where the temp directory should be created. If not specified, temp directory
#         will be created in the default temp directory of the OS.
# + return - Temporary directory path or else a `file:Error` if there is an error
public isolated function createTempDir(string? suffix = (), string? prefix = (), string? dir = ()) returns string|Error = external;

# Tests a file path against a test condition.
#
# + path - String value of the file path
# + testOption - The option to be tested upon the path. Supported options are,
#  `EXISTS` - Test whether a file path exists,
#  `IS_DIR` - Test whether a file path is a directory,
#  `IS_SYMLINK` - Test whether a file path is a symlink,
#  `READABLE` - Test whether a file path is readable,
#  `WRITABLE` - Test whether a file path is writable.
# + return - True/false depending on the option to be tested or else a `file:Error` if there is an error
public isolated function test(string path, TestOption testOption) returns boolean|Error = external;
