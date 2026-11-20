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

// What section 7.1 allows a name inside an archive to be. The same check runs in both directions: a
// name read from an archive is refused before anything is written to disk, and a name handed to the
// writer is refused rather than quietly corrected, so the library never writes one of these either.
isolated function validateEntryName(string name) returns Error? {
    if name == "" {
        return error UnsafePathError("invalid entry name: the name is empty", entryName = name);
    }
    if name.includes("\\") {
        return error UnsafePathError(string `invalid entry name '${name}': a name cannot contain '\'`,
                entryName = name);
    }
    if name.includes("\u{0000}") {
        return error UnsafePathError(string `invalid entry name '${name}': a name cannot contain a zero byte`,
                entryName = name);
    }
    if hasPrefix(name, "/") {
        return error UnsafePathError(string `invalid entry name '${name}': a name cannot be absolute`,
                entryName = name);
    }
    if hasDriveLetter(name) {
        return error UnsafePathError(string `invalid entry name '${name}': a name cannot start with a drive letter`,
                entryName = name);
    }
    // A ':' names a drive at the start of a name and an alternate data stream anywhere else, so it is
    // refused wherever it appears. Opening 'notes.txt:evil' on NTFS writes a stream of 'notes.txt'
    // rather than a file of that name, which puts the content somewhere the caller cannot see it.
    if name.includes(":") {
        return error UnsafePathError(string `invalid entry name '${name}': a name cannot contain ':'`,
                entryName = name);
    }
    foreach string segment in segmentsOf(name) {
        if segment == "." || segment == ".." {
            return error UnsafePathError(
                    string `invalid entry name '${name}': a name cannot contain a '${segment}' segment`,
                    entryName = name);
        }
    }
    return;
}

isolated function hasDriveLetter(string name) returns boolean {
    if name.length() < 2 || name.substring(1, 2) != ":" {
        return false;
    }
    string first = name.substring(0, 1);
    return (first >= "a" && first <= "z") || (first >= "A" && first <= "Z");
}

isolated function segmentsOf(string name) returns string[] {
    string[] segments = [];
    int segmentStart = 0;
    int index = 0;
    int length = name.length();
    while index < length {
        if name.substring(index, index + 1) == "/" {
            segments.push(name.substring(segmentStart, index));
            segmentStart = index + 1;
        }
        index += 1;
    }
    segments.push(name.substring(segmentStart, length));
    return segments;
}

// `string:startsWith` and `string:endsWith` are not yet available in this interpreter's langlib.
isolated function hasPrefix(string text, string prefix) returns boolean {
    return text.length() >= prefix.length() && text.substring(0, prefix.length()) == prefix;
}

isolated function hasSuffix(string text, string suffix) returns boolean {
    int length = text.length();
    return length >= suffix.length() && text.substring(length - suffix.length(), length) == suffix;
}
